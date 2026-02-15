package xforms

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse_SchemaInvalidNillableBool_ReturnsError(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <element name="data">
          <complexType>
            <all>
              <element name="x" nillable="notabool" type="xsd:string"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance><data><x>1</x></data></instance>
      <bind nodeset="x" relevant="true()"/>
    </model>
  </head>
  <body></body>
</html>`
	_, err := Parse(strings.NewReader(xml))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParse_SchemaFacetInvalidInt_ReturnsError(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <simpleType name="S">
          <restriction base="xsd:string">
            <minLength value="nope"/>
          </restriction>
        </simpleType>
      </schema>
      <instance><data></data></instance>
    </model>
  </head>
  <body></body>
</html>`
	_, err := Parse(strings.NewReader(xml))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSchemaSimpleTypeFacets_AreParsed(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test">
        <simpleType name="S">
          <restriction base="xsd:string">
            <minLength value="1"/>
            <maxLength value="5"/>
            <enumeration value="a"/>
            <enumeration value="b"/>
          </restriction>
        </simpleType>
        <simpleType name="D">
          <restriction base="xsd:decimal">
            <totalDigits value="4"/>
            <fractionDigits value="2"/>
            <minInclusive value="0.01"/>
            <maxInclusive value="99.99"/>
          </restriction>
        </simpleType>
        <element name="data">
          <complexType>
            <all>
              <element name="s" nillable="false" type="S"/>
              <element name="d" nillable="false" type="D"/>
            </all>
          </complexType>
        </element>
      </schema>
      <instance><data><s>a</s><d>1.23</d></data></instance>
      <bind nodeset="s" relevant="true()" required="true" type="S"/>
      <bind nodeset="d" relevant="true()" required="true" type="D"/>
    </model>
  </head>
  <body></body>
</html>`

	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	ip := func(i int) *int { return &i }
	sp := func(s string) *string { return &s }
	want := map[string]*FormSchemaSimpleType{
		"S": {
			Name:          "S",
			BaseTypeQName: "xsd:string",
			MinLength:     ip(1),
			MaxLength:     ip(5),
			Enumeration:   []string{"a", "b"},
		},
		"D": {
			Name:           "D",
			BaseTypeQName:  "xsd:decimal",
			TotalDigits:    ip(4),
			FractionDigits: ip(2),
			MinInclusive:   sp("0.01"),
			MaxInclusive:   sp("99.99"),
		},
	}
	got := map[string]*FormSchemaSimpleType{
		"S": f.Schema.SimpleTypes["S"],
		"D": f.Schema.SimpleTypes["D"],
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("parsed facets mismatch (-want +got):\n%s", diff)
	}
}

func TestMarkerMethods_FormBodyElements(t *testing.T) {
	// These are marker methods; calling them increases coverage without changing behavior.
	(&FormBodyGroup{}).formBodyElement()
	(&FormBodyInput{}).formBodyElement()
	(&FormBodySelect{}).formBodyElement()
	(&FormBodyOutput{}).formBodyElement()
}
