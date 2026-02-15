package xforms

import (
	"strings"
	"testing"
)

func TestForm_Validate_SampleDetectsPatternMismatch(t *testing.T) {
	f, err := Parse(strings.NewReader(sampleXFormsXML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	err = f.Validate()
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
	}
	if len(ve.FieldErrors) == 0 {
		t.Fatalf("expected at least one field error")
	}
	if _, ok := ve.FieldErrors["field_PERSONAL_ACCOUNT"]; !ok {
		t.Fatalf("expected error for field_PERSONAL_ACCOUNT, got: %#v", ve.FieldErrors)
	}
}

func TestForm_Validate_SamplePassesWhenValueFixed(t *testing.T) {
	xml := strings.Replace(sampleXFormsXML, ">012345678<", ">0123456789<", 1)
	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if err := f.Validate(); err != nil {
		t.Fatalf("expected no validation error, got %T: %v", err, err)
	}
}

func TestForm_Validate_RequiredAndBoolean(t *testing.T) {
	const xml = `<?xml version="1.0" encoding="UTF-8"?>
<xhtml:html xmlns:xforms="http://www.w3.org/2002/xforms" xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xhtml:head>
    <xforms:model>
      <xsd:schema targetNamespace="urn:test">
        <xsd:element name="data">
          <xsd:complexType>
            <xsd:all>
              <xsd:element name="flag" nillable="false" type="xsd:boolean"/>
            </xsd:all>
          </xsd:complexType>
        </xsd:element>
      </xsd:schema>
      <xforms:instance>
        <data>
          <flag>notabool</flag>
        </data>
      </xforms:instance>
      <xforms:bind nodeset="flag" required="true" relevant="true()" readonly="false" type="xsd:boolean"/>
    </xforms:model>
  </xhtml:head>
  <xhtml:body>
    <xforms:input ref="flag">
      <xforms:label>Flag</xforms:label>
    </xforms:input>
  </xhtml:body>
</xhtml:html>
`
	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	err = f.Validate()
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if _, ok := ve.FieldErrors["flag"]; !ok {
		t.Fatalf("expected error for flag, got %#v", ve.FieldErrors)
	}
}

func TestForm_Validate_CornerCases(t *testing.T) {
	tests := []struct {
		name          string
		xml           string
		wantErrFields []string
		wantErrSubAny []string
	}{
		{
			name: "minLength_uses_runes",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <simpleType name="T">
          <restriction base="xsd:string">
            <minLength value="2"/>
          </restriction>
        </simpleType>
        <element name="data">
          <complexType>
            <all>
              <element name="s" nillable="false" type="T"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance>
        <data><s>я</s></data>
      </instance>
      <bind nodeset="s" relevant="true()" required="true" type="T"/>
    </model>
  </head>
  <body><input ref="s"><label>S</label></input></body>
</html>`,
			wantErrFields: []string{"s"},
			wantErrSubAny: []string{"length must be >="},
		},
		{
			name: "enumeration_rejects_other_values",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <simpleType name="E">
          <restriction base="xsd:string">
            <enumeration value="a"/>
            <enumeration value="b"/>
          </restriction>
        </simpleType>
        <element name="data">
          <complexType>
            <all>
              <element name="v" nillable="false" type="E"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance>
        <data><v>c</v></data>
      </instance>
      <bind nodeset="v" relevant="true()" required="true" type="E"/>
    </model>
  </head>
  <body><input ref="v"><label>V</label></input></body>
</html>`,
			wantErrFields: []string{"v"},
			wantErrSubAny: []string{"enumeration"},
		},
		{
			name: "xsi_nil_not_nillable",
			xml: `<?xml version="1.0"?>
<html xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <element name="data">
          <complexType>
            <all>
              <element name="n" nillable="false" type="xsd:string"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance>
        <data><n xsi:nil="true"></n></data>
      </instance>
      <bind nodeset="n" relevant="true()" required="false" type="xsd:string"/>
    </model>
  </head>
  <body><input ref="n"><label>N</label></input></body>
</html>`,
			wantErrFields: []string{"n"},
			wantErrSubAny: []string{"not nillable"},
		},
		{
			name: "relevant_false_skips_validation",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <simpleType name="T">
          <restriction base="xsd:string">
            <pattern value="^\\d+$"/>
          </restriction>
        </simpleType>
        <element name="data">
          <complexType>
            <all>
              <element name="x" nillable="false" type="T"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance>
        <data><x>not-a-number</x></data>
      </instance>
      <bind nodeset="x" relevant="false()" required="true" type="T"/>
    </model>
  </head>
  <body><input ref="x"><label>X</label></input></body>
</html>`,
			wantErrFields: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			err = f.Validate()

			if len(tt.wantErrFields) == 0 {
				if err != nil {
					t.Fatalf("expected no error, got %T: %v", err, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			ve, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			for _, field := range tt.wantErrFields {
				if _, ok := ve.FieldErrors[field]; !ok {
					t.Fatalf("expected error for field %q, got %#v", field, ve.FieldErrors)
				}
			}
			if len(tt.wantErrSubAny) > 0 {
				flat := strings.Join(flattenFieldErrors(ve), " | ")
				found := false
				for _, sub := range tt.wantErrSubAny {
					if strings.Contains(flat, sub) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected any of %v in errors, got %q", tt.wantErrSubAny, flat)
				}
			}
		})
	}
}

func TestValidate_DecimalBounds_AndInvalidSchemaPattern(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		field   string
		wantSub string
	}{
		{
			name: "decimal_maxExclusive",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <simpleType name="D">
          <restriction base="xsd:decimal">
            <maxExclusive value="10.0"/>
          </restriction>
        </simpleType>
        <element name="data">
          <complexType><all><element name="d" nillable="false" type="D"/></all></complexType>
        </element>
      </schema>
      <instance><data><d>10.0</d></data></instance>
      <bind nodeset="d" relevant="true()" required="true" type="D"/>
    </model>
  </head>
  <body><input ref="d"><label>D</label></input></body>
</html>`,
			field:   "d",
			wantSub: "must be <",
		},
		{
			name: "invalid_schema_pattern",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <simpleType name="P">
          <restriction base="xsd:string">
            <pattern value="["/>
          </restriction>
        </simpleType>
        <element name="data">
          <complexType><all><element name="p" nillable="false" type="P"/></all></complexType>
        </element>
      </schema>
      <instance><data><p>x</p></data></instance>
      <bind nodeset="p" relevant="true()" required="true" type="P"/>
    </model>
  </head>
  <body><input ref="p"><label>P</label></input></body>
</html>`,
			field:   "p",
			wantSub: "invalid schema pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			err = f.Validate()
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			ve, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T", err)
			}
			msgs := strings.Join(ve.FieldErrors[tt.field], " | ")
			if !strings.Contains(msgs, tt.wantSub) {
				t.Fatalf("expected %q in field errors, got %q", tt.wantSub, msgs)
			}
		})
	}
}

func TestValidationError_Error_IsReadable(t *testing.T) {
	ve := &ValidationError{
		FieldErrors: map[string][]string{
			"b": {"err"},
			"a": {"err"},
		},
	}
	s := ve.Error()
	if !strings.Contains(s, "2 field(s)") || !strings.Contains(s, "a") {
		t.Fatalf("unexpected Error() string: %q", s)
	}
}

func TestSchemaDeclForInstanceField_FallsBackToTopLevelWhenNoRoot(t *testing.T) {
	f := &Form{
		Schema: FormSchema{
			Elements: map[string]*FormSchemaElementDecl{
				"x": {Name: "x", TypeQName: "xsd:string"},
			},
		},
		Instance: FormInstance{}, // Root.Local is empty
	}
	if decl := f.schemaDeclForInstanceField("x"); decl == nil || decl.Name != "x" {
		t.Fatalf("expected top-level decl for x, got %#v", decl)
	}
}

func flattenFieldErrors(ve *ValidationError) []string {
	var out []string
	for _, msgs := range ve.FieldErrors {
		out = append(out, msgs...)
	}
	return out
}
