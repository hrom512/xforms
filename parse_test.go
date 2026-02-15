package xforms

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const sampleXFormsXML = `<?xml version="1.0" encoding="UTF-8"?>
<xhtml:html xmlns:demo="urn:demo-xforms" xmlns:xforms="http://www.w3.org/2002/xforms" xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <xhtml:head>
    <xforms:model>
      <xsd:schema targetNamespace="urn:demo-xforms">
        <xsd:simpleType name="PERSONAL_ACCOUNT">
          <xsd:restriction base="xsd:string">
            <xsd:pattern value="^\d{10}$"/>
          </xsd:restriction>
        </xsd:simpleType>
        <xsd:simpleType name="TERMINALID">
          <xsd:restriction base="xsd:string"/>
        </xsd:simpleType>
        <xsd:simpleType name="CHECK_TEXT">
          <xsd:restriction base="xsd:string"/>
        </xsd:simpleType>
        <xsd:complexType name="SumCheck">
          <xsd:all>
            <xsd:element name="sum" type="xsd:decimal"/>
            <xsd:element name="check" type="xsd:boolean"/>
          </xsd:all>
        </xsd:complexType>
        <xsd:element name="xmlData">
          <xsd:complexType>
            <xsd:all>
              <xsd:element name="transactionId" nillable="false" type="xsd:string"/>
              <xsd:element name="field_PERSONAL_ACCOUNT" nillable="false" type="demo:PERSONAL_ACCOUNT"/>
              <xsd:element name="field_TERMINALID" nillable="false" type="demo:TERMINALID"/>
              <xsd:element name="field_CHECK_TEXT" nillable="true" type="demo:CHECK_TEXT"/>
            </xsd:all>
          </xsd:complexType>
        </xsd:element>
      </xsd:schema>
      <xforms:instance>
        <demo:xmlData>
          <transactionId>456803</transactionId>
          <field_PERSONAL_ACCOUNT>012345678</field_PERSONAL_ACCOUNT>
          <field_TERMINALID>testTerminal01</field_TERMINALID>
          <field_CHECK_TEXT/>
        </demo:xmlData>
      </xforms:instance>
      <xforms:bind exttype="PERSONAL_ACCOUNT" nodeset="field_PERSONAL_ACCOUNT" readonly="false" relevant="true()" required="true" type="demo:PERSONAL_ACCOUNT"/>
      <xforms:bind exttype="PARAMETER" nodeset="field_TERMINALID" readonly="false" relevant="false()" required="true" type="demo:TERMINALID"/>
      <xforms:bind exttype="CLIENT" nodeset="field_CHECK_TEXT" readonly="true" relevant="false()" required="false" type="demo:CHECK_TEXT"/>
      <xforms:submission action="http://localhost/" id="submission.back" method="get"/>
      <xforms:submission action="http://localhost/" id="submission.next" method="get"/>
      <xforms:submission action="http://localhost/" id="submission.pay" method="get"/>
      <xforms:submission action="http://localhost/" id="submission.link" method="get"/>
    </xforms:model>
  </xhtml:head>
  <xhtml:body>
    <xforms:group id="10">
      <xforms:label/>
      <xforms:input id="PERSONAL_ACCOUNT" incremental="true" ref="field_PERSONAL_ACCOUNT">
        <xforms:label>Personal account number:</xforms:label>
        <xforms:alert>Incorrect personal account number format!</xforms:alert>
        <xforms:help>Example of completion: 1234567890</xforms:help>
      </xforms:input>
    </xforms:group>
    <xforms:submit id="back" submission="submission.back">
      <xforms:label>Back</xforms:label>
    </xforms:submit>
    <xforms:submit id="next" submission="submission.next">
      <xforms:label>Next</xforms:label>
    </xforms:submit>
  </xhtml:body>
</xhtml:html>
`

func TestParse_SampleXForms(t *testing.T) {
	got, err := Parse(strings.NewReader(sampleXFormsXML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	sp := func(s string) *string { return &s }
	bp := func(b bool) *bool { return &b }

	want := &Form{
		Schema: FormSchema{
			TargetNamespace: "urn:demo-xforms",
			SimpleTypes: map[string]*FormSchemaSimpleType{
				"PERSONAL_ACCOUNT": {
					Name:          "PERSONAL_ACCOUNT",
					BaseTypeQName: "xsd:string",
					Pattern:       sp("^\\d{10}$"),
				},
				"TERMINALID": {
					Name:          "TERMINALID",
					BaseTypeQName: "xsd:string",
				},
				"CHECK_TEXT": {
					Name:          "CHECK_TEXT",
					BaseTypeQName: "xsd:string",
				},
			},
			ComplexTypes: map[string]*FormSchemaComplexType{
				"SumCheck": {
					Name: "SumCheck",
					All: []FormSchemaElementDecl{
						{Name: "sum", TypeQName: "xsd:decimal"},
						{Name: "check", TypeQName: "xsd:boolean"},
					},
				},
			},
			Elements: map[string]*FormSchemaElementDecl{
				"xmlData": {
					Name:      "xmlData",
					TypeQName: "",
					ComplexType: &FormSchemaComplexType{
						Name: "",
						All: []FormSchemaElementDecl{
							{Name: "transactionId", TypeQName: "xsd:string", Nillable: bp(false)},
							{Name: "field_PERSONAL_ACCOUNT", TypeQName: "demo:PERSONAL_ACCOUNT", Nillable: bp(false)},
							{Name: "field_TERMINALID", TypeQName: "demo:TERMINALID", Nillable: bp(false)},
							{Name: "field_CHECK_TEXT", TypeQName: "demo:CHECK_TEXT", Nillable: bp(true)},
						},
					},
				},
			},
		},
		Instance: FormInstance{
			Root: xml.Name{Space: "urn:demo-xforms", Local: "xmlData"},
			Fields: map[string]FormInstanceField{
				"transactionId":          {Name: "transactionId", Namespace: "", Value: "456803", IsNil: false},
				"field_PERSONAL_ACCOUNT": {Name: "field_PERSONAL_ACCOUNT", Namespace: "", Value: "012345678", IsNil: false},
				"field_TERMINALID":       {Name: "field_TERMINALID", Namespace: "", Value: "testTerminal01", IsNil: false},
				"field_CHECK_TEXT":       {Name: "field_CHECK_TEXT", Namespace: "", Value: "", IsNil: false},
			},
		},
		Binds: []*FormBind{
			{
				Nodeset:     "field_PERSONAL_ACCOUNT",
				TypeQName:   sp("demo:PERSONAL_ACCOUNT"),
				ExtType:     sp("PERSONAL_ACCOUNT"),
				Required:    true,
				Readonly:    false,
				Relevant:    true,
				RawRequired: "true",
				RawReadonly: "false",
				RawRelevant: "true()",
			},
			{
				Nodeset:     "field_TERMINALID",
				TypeQName:   sp("demo:TERMINALID"),
				ExtType:     sp("PARAMETER"),
				Required:    true,
				Readonly:    false,
				Relevant:    false,
				RawRequired: "true",
				RawReadonly: "false",
				RawRelevant: "false()",
			},
			{
				Nodeset:     "field_CHECK_TEXT",
				TypeQName:   sp("demo:CHECK_TEXT"),
				ExtType:     sp("CLIENT"),
				Required:    false,
				Readonly:    true,
				Relevant:    false,
				RawRequired: "false",
				RawReadonly: "true",
				RawRelevant: "false()",
			},
		},
		Submissions: []*FormSubmission{
			{ID: "submission.back", Action: "http://localhost/", Method: "get"},
			{ID: "submission.next", Action: "http://localhost/", Method: "get"},
			{ID: "submission.pay", Action: "http://localhost/", Method: "get"},
			{ID: "submission.link", Action: "http://localhost/", Method: "get"},
		},
		Body: FormBody{
			Elements: []FormBodyElement{
				&FormBodyGroup{
					ID:    sp("10"),
					Label: "",
					Elements: []FormBodyElement{
						&FormBodyInput{
							ID:          sp("PERSONAL_ACCOUNT"),
							Ref:         "field_PERSONAL_ACCOUNT",
							Incremental: bp(true),
							Label:       "Personal account number:",
							Alert:       "Incorrect personal account number format!",
							Help:        "Example of completion: 1234567890",
							Hint:        "",
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Parse() mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name       string
		xml        string
		wantErrSub string
	}{
		{
			name: "bind_missing_nodeset",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test"></schema>
      <instance><data><x>1</x></data></instance>
      <bind required="true"/>
    </model>
  </head>
  <body></body>
</html>`,
			wantErrSub: "missing nodeset",
		},
		{
			name: "bind_unsupported_bool_expr",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test"></schema>
      <instance><data><x>1</x></data></instance>
      <bind nodeset="x" relevant="1=1"/>
    </model>
  </head>
  <body></body>
</html>`,
			wantErrSub: "unsupported boolean expression",
		},
		{
			name: "submission_missing_id",
			xml: `<?xml version="1.0"?>
<html>
  <head>
    <model>
      <schema targetNamespace="urn:test"></schema>
      <instance><data><x>1</x></data></instance>
      <submission action="http://localhost/" method="get"/>
    </model>
  </head>
  <body></body>
</html>`,
			wantErrSub: "submission missing id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.xml))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatalf("expected error to contain %q, got %q", tt.wantErrSub, err.Error())
			}
		})
	}
}
