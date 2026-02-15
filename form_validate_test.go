package xforms

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func TestForm_Validate_SampleDetectsPatternMismatch(t *testing.T) {
	f, err := Parse(strings.NewReader(sampleXFormsXML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	err = f.Validate()
	got := asValidationError(err)
	want := &ValidationError{
		FieldErrors: map[string][]string{
			"field_PERSONAL_ACCOUNT": {"does not match required pattern"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
	}
}

func TestForm_Validate_SamplePassesWhenValueFixed(t *testing.T) {
	xml := strings.Replace(sampleXFormsXML, ">012345678<", ">0123456789<", 1)
	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	got := asValidationError(f.Validate())
	if diff := cmp.Diff((*ValidationError)(nil), got); diff != "" {
		t.Fatalf("expected no validation error (-want +got):\n%s", diff)
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
	got := asValidationError(f.Validate())
	want := &ValidationError{
		FieldErrors: map[string][]string{
			"flag": {"must be a boolean"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
	}
}

func TestForm_Validate_CornerCases(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		wantErr *ValidationError
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
			wantErr: &ValidationError{
				FieldErrors: map[string][]string{
					"s": {"length must be >= 2"},
				},
			},
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
			wantErr: &ValidationError{
				FieldErrors: map[string][]string{
					"v": {"value is not in enumeration"},
				},
			},
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
			wantErr: &ValidationError{
				FieldErrors: map[string][]string{
					"n": {"value is nil but field is not nillable"},
				},
			},
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
			wantErr: nil,
		},
		{
			name: "required_empty_short_circuits_other_checks",
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
        <data><x></x></data>
      </instance>
      <bind nodeset="x" relevant="true()" required="true" type="T"/>
    </model>
  </head>
  <body><input ref="x"><label>X</label></input></body>
</html>`,
			wantErr: &ValidationError{
				FieldErrors: map[string][]string{
					"x": {"field is required"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			got := asValidationError(f.Validate())
			if diff := cmp.Diff(tt.wantErr, got); diff != "" {
				t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidate_DecimalBounds_AndInvalidSchemaPattern(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		wantErr *ValidationError
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
			wantErr: func() *ValidationError {
				max, _ := decimal.NewFromString("10.0")
				return &ValidationError{
					FieldErrors: map[string][]string{
						"d": {"must be < " + max.String()},
					},
				}
			}(),
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
			wantErr: &ValidationError{
				FieldErrors: map[string][]string{
					"p": {"invalid schema pattern"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			got := asValidationError(f.Validate())
			if diff := cmp.Diff(tt.wantErr, got); diff != "" {
				t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
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
	got := ve.Error()
	want := "validation failed: 2 field(s) invalid (e.g. a)"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("ValidationError.Error() mismatch (-want +got):\n%s", diff)
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
	got := f.schemaDeclForInstanceField("x")
	want := &FormSchemaElementDecl{Name: "x", TypeQName: "xsd:string"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("schemaDeclForInstanceField mismatch (-want +got):\n%s", diff)
	}
}

func asValidationError(err error) *ValidationError {
	if err == nil {
		return nil
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		// Keep it explicit so mismatches are obvious in diffs.
		return &ValidationError{FieldErrors: map[string][]string{"__unexpected_error_type__": {err.Error()}}}
	}
	return ve
}
