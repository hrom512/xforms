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
	if _, ok := ve.FieldErrors["a3_PERSONAL_ACCOUNT_1_1"]; !ok {
		t.Fatalf("expected error for a3_PERSONAL_ACCOUNT_1_1, got: %#v", ve.FieldErrors)
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

