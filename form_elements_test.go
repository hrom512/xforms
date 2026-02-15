package xforms

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func TestForm_Elements_Sample(t *testing.T) {
	f, err := Parse(strings.NewReader(sampleXFormsXML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	sp := func(s string) *string { return &s }

	want := []FormElement{
		&FieldGroup{
			Label: "",
			Elements: []FormElement{
				&TextInput{
					Name:     "field_PERSONAL_ACCOUNT",
					Label:    "Personal account number:",
					ExtType:  sp("PERSONAL_ACCOUNT"),
					Required: true,
					Readonly: false,
					SimpleType: &FormSchemaSimpleType{
						Name:          "PERSONAL_ACCOUNT",
						BaseTypeQName: "xsd:string",
						Pattern:       sp("^\\d{10}$"),
					},
					Alert: "Incorrect personal account number format!",
					Help:  "Example of completion: 1234567890",
					Hint:  "",
					Value: sp("012345678"),
				},
			},
		},
	}

	got := f.Elements()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Elements() mismatch (-want +got):\n%s", diff)
	}
}

func TestForm_Elements_SelectAndOutput_AndSkipUnsupported(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<html xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <element name="data">
          <complexType>
            <all>
              <element name="choice" nillable="false" type="xsd:string"/>
              <element name="msg" nillable="false" type="xsd:string"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance>
        <data>
          <choice>b</choice>
          <msg>Hello</msg>
        </data>
      </instance>
      <bind nodeset="choice" relevant="true()" readonly="false" required="true" type="xsd:string"/>
      <bind nodeset="msg" relevant="true()" readonly="true" required="false" type="xsd:string"/>
    </model>
  </head>
  <body>
    <input id="no-ref">
      <label>Ignored</label>
    </input>
    <select1 id="choice" ref="choice">
      <label>Pick</label>
      <item><label>A</label><value>a</value></item>
      <item><label>B</label><value>b</value></item>
    </select1>
    <output id="m" ref="msg">
      <label>Static label ignored by ref</label>
    </output>
    <submit id="unsupported"><label>Skip</label></submit>
  </body>
</html>`

	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	want := []FormElement{
		&SelectInput{
			Name:       "choice",
			Label:      "Pick",
			ExtType:    nil,
			Required:   true,
			Readonly:   false,
			SimpleType: &FormSchemaSimpleType{Name: "string", BaseTypeQName: "xsd:string"},
			Options: []SelectOption{
				{Label: "A", Value: "a"},
				{Label: "B", Value: "b"},
			},
			Value: &SelectOption{Label: "B", Value: "b"},
		},
		&TextMessage{
			Message: "Hello",
			ID:      strPtr("m"),
		},
	}

	got := f.Elements()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Elements() mismatch (-want +got):\n%s", diff)
	}
}

func TestForm_Elements_ResolveSchemaType_FallbackToSchemaDecl(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <element name="data">
          <complexType>
            <all>
              <element name="flag" nillable="false" type="xsd:boolean"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance>
        <data><flag>true</flag></data>
      </instance>
      <bind nodeset="flag" relevant="true()" required="true" readonly="false"/>
    </model>
  </head>
  <body>
    <input ref="flag"><label>Flag</label></input>
  </body>
</html>`

	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	got := f.Elements()
	want := []FormElement{
		&CheckboxInput{
			Name:       "flag",
			Label:      "Flag",
			Required:   true,
			Readonly:   false,
			SimpleType: &FormSchemaSimpleType{Name: "boolean", BaseTypeQName: "xsd:boolean"},
			Value:      boolPtr(true),
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Elements() mismatch (-want +got):\n%s", diff)
	}
}

func TestElements_UsesSchemaDeclTypeWhenBindTypeMissing(t *testing.T) {
	// Bind has no type; schema element decl provides xsd:decimal => should produce DecimalInput.
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <element name="data">
          <complexType>
            <all>
              <element name="amt" nillable="false" type="xsd:decimal"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance><data><amt>1.25</amt></data></instance>
      <bind nodeset="amt" relevant="true()" required="true"/>
    </model>
  </head>
  <body><input ref="amt"><label>Amt</label></input></body>
</html>`

	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	want := []FormElement{
		&DecimalInput{
			Name:       "amt",
			Label:      "Amt",
			ExtType:    nil,
			Required:   true,
			Readonly:   false,
			SimpleType: &FormSchemaSimpleType{Name: "decimal", BaseTypeQName: "xsd:decimal"},
			Value:      mustDecimalPtr("1.25"),
		},
	}
	got := f.Elements()

	decimalPtrComparer := cmp.Comparer(func(a, b *decimal.Decimal) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.String() == b.String()
	})
	if diff := cmp.Diff(want, got, decimalPtrComparer); diff != "" {
		t.Fatalf("Elements() mismatch (-want +got):\n%s", diff)
	}
}

func TestElements_BindTypeQNameOverridesSchemaDecl(t *testing.T) {
	// Schema declares x as string, but bind forces xsd:boolean.
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <element name="data">
          <complexType>
            <all>
              <element name="x" nillable="false" type="xsd:string"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance><data><x>true</x></data></instance>
      <bind nodeset="x" relevant="true()" required="true" type="xsd:boolean"/>
    </model>
  </head>
  <body><input ref="x"><label>X</label></input></body>
</html>`

	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	want := []FormElement{
		&CheckboxInput{
			Name:       "x",
			Label:      "X",
			ExtType:    nil,
			Required:   true,
			Readonly:   false,
			SimpleType: &FormSchemaSimpleType{Name: "boolean", BaseTypeQName: "xsd:boolean"},
			Value:      boolPtr(true),
		},
	}

	got := f.Elements()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Elements() mismatch (-want +got):\n%s", diff)
	}
}

func TestElements_TopLevelSchemaElementDeclFallback_AndMissingInstanceValue(t *testing.T) {
	// schema defines a top-level element 'x' (not under instance root),
	// bind has no type => resolveSchemaType should fall back to Schema.Elements["x"].
	// instance does not contain x => Value must be nil.
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <element name="x" nillable="false" type="xsd:string"/>
      </schema>
      <instance><data></data></instance>
      <bind nodeset="x" relevant="true()" required="false"/>
    </model>
  </head>
  <body><input ref="x"><label>X</label></input></body>
</html>`

	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	got := f.Elements()
	want := []FormElement{
		&TextInput{
			Name:       "x",
			Label:      "X",
			Required:   false,
			Readonly:   false,
			SimpleType: &FormSchemaSimpleType{Name: "string", BaseTypeQName: "xsd:string"},
			Value:      nil,
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Elements() mismatch (-want +got):\n%s", diff)
	}
}

func TestElements_ComplexTypeBindType_ProducesComplexInputWithComplexType(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <complexType name="C">
          <all>
            <element name="a" type="xsd:string"/>
          </all>
        </complexType>
        <element name="data">
          <complexType>
            <all>
              <element name="obj" nillable="false" type="C"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance>
        <data>
          <obj>
            <a>some value</a>
          </obj>
        </data>
      </instance>
      <bind nodeset="obj" relevant="true()" required="true" type="C"/>
    </model>
  </head>
  <body><input ref="obj"><label>Obj</label></input></body>
</html>`

	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	want := []FormElement{
		&ComplexInput{
			Name:     "obj",
			Label:    "Obj",
			ExtType:  nil,
			Required: true,
			Readonly: false,
			ComplexType: &FormSchemaComplexType{
				Name: "C",
				All: []FormSchemaElementDecl{
					{Name: "a", TypeQName: "xsd:string"},
				},
			},
			Alert: "",
			Help:  "",
			Hint:  "",
			RawValue: strPtr("<a>some value</a>"),
			Value: map[string]string{
				"a": "some value",
			},
		},
	}

	got := f.Elements()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Elements() mismatch (-want +got):\n%s", diff)
	}
}

func TestMarkerMethods_FormElements(t *testing.T) {
	// These are marker methods; calling them increases coverage without changing behavior.
	(&TextInput{}).FormElement()
	(&DecimalInput{}).FormElement()
	(&CheckboxInput{}).FormElement()
	(&SelectInput{}).FormElement()
	(&ComplexInput{}).FormElement()
	(&TextMessage{}).FormElement()
	(&FieldGroup{}).FormElement()
}

// helpers

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

func mustDecimalPtr(s string) *decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return &d
}
