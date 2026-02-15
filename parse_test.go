package xforms

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const sampleXFormsXML = `<?xml version="1.0" encoding="UTF-8"?>
<xhtml:html xmlns:a-3="http://www.a-3.ru/xforms/schema" xmlns:xforms="http://www.w3.org/2002/xforms" xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchemainstance" xsi:schemaLocation="http://www.w3.org/2002/xforms http://www.w3.org/MarkUp/Forms/2002/XForms-Schema.xsd">
  <xhtml:head>
    <xforms:model>
      <xsd:schema targetNamespace="http://www.a-3.ru/xforms/schema">
        <xsd:simpleType name="PERSONAL_ACCOUNT_1_1">
          <xsd:restriction base="xsd:string">
            <xsd:pattern value="^\d{10}$"/>
          </xsd:restriction>
        </xsd:simpleType>
        <xsd:simpleType name="TERMINALID_2_1">
          <xsd:restriction base="xsd:string"/>
        </xsd:simpleType>
        <xsd:simpleType name="_CHECK_TEXT__3_1">
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
              <xsd:element name="a3_PERSONAL_ACCOUNT_1_1" nillable="false" type="a-3:PERSONAL_ACCOUNT_1_1"/>
              <xsd:element name="a3_TERMINALID_2_1" nillable="false" type="a-3:TERMINALID_2_1"/>
              <xsd:element name="a3__CHECK_TEXT__3_1" nillable="true" type="a-3:_CHECK_TEXT__2_1"/>
            </xsd:all>
          </xsd:complexType>
        </xsd:element>
      </xsd:schema>
      <xforms:instance>
        <a-3:xmlData>
          <transactionId>456803</transactionId>
          <a3_PERSONAL_ACCOUNT_1_1>012345678</a3_PERSONAL_ACCOUNT_1_1>
          <a3_TERMINALID_2_1>testTerminal01</a3_TERMINALID_2_1>
          <a3__CHECK_TEXT__3_1/>
        </a-3:xmlData>
      </xforms:instance>
      <xforms:bind exttype="PERSONAL_ACCOUNT" nodeset="a3_PERSONAL_ACCOUNT_1_1" readonly="false" relevant="true()" required="true" type="a-3:PERSONAL_ACCOUNT_1_1"/>
      <xforms:bind exttype="PARAMETER" nodeset="a3_TERMINALID_2_1" readonly="false" relevant="false()" required="true" type="a-3:TERMINALID_2_1"/>
      <xforms:bind exttype="CLIENT" nodeset="a3__CHECK_TEXT__3_1" readonly="true" relevant="false()" required="false" type="a-3:_CHECK_TEXT__3_1"/>
      <xforms:submission action="http://localhost/" id="a-3.submission.back" method="get"/>
      <xforms:submission action="http://localhost/" id="a-3.submission.next" method="get"/>
      <xforms:submission action="http://localhost/" id="a-3.submission.pay" method="get"/>
      <xforms:submission action="http://localhost/" id="a-3.submission.link" method="get"/>
    </xforms:model>
  </xhtml:head>
  <xhtml:body>
    <xforms:group id="10">
      <xforms:label/>
      <xforms:input id="PERSONAL_ACCOUNT_1_1" incremental="true" ref="a3_PERSONAL_ACCOUNT_1_1">
        <xforms:label>Personal account number:</xforms:label>
        <xforms:alert>Incorrect personal account number format!</xforms:alert>
        <xforms:help>Example of completion: 1234567890</xforms:help>
      </xforms:input>
    </xforms:group>
    <xforms:submit id="a-3.back" submission="a-3.submission.back">
      <xforms:label>Back</xforms:label>
    </xforms:submit>
    <xforms:submit id="a-3.next" submission="a-3.submission.next">
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
			TargetNamespace: "http://www.a-3.ru/xforms/schema",
			SimpleTypes: map[string]*FormSchemaSimpleType{
				"PERSONAL_ACCOUNT_1_1": {
					Name:          "PERSONAL_ACCOUNT_1_1",
					BaseTypeQName: "xsd:string",
					Pattern:       sp("^\\d{10}$"),
				},
				"TERMINALID_2_1": {
					Name:          "TERMINALID_2_1",
					BaseTypeQName: "xsd:string",
				},
				"_CHECK_TEXT__3_1": {
					Name:          "_CHECK_TEXT__3_1",
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
							{Name: "a3_PERSONAL_ACCOUNT_1_1", TypeQName: "a-3:PERSONAL_ACCOUNT_1_1", Nillable: bp(false)},
							{Name: "a3_TERMINALID_2_1", TypeQName: "a-3:TERMINALID_2_1", Nillable: bp(false)},
							{Name: "a3__CHECK_TEXT__3_1", TypeQName: "a-3:_CHECK_TEXT__2_1", Nillable: bp(true)},
						},
					},
				},
			},
		},
		Instance: FormInstance{
			Root: xml.Name{Space: "http://www.a-3.ru/xforms/schema", Local: "xmlData"},
			Fields: map[string]FormInstanceField{
				"transactionId":           {Name: "transactionId", Namespace: "", Value: "456803", IsNil: false},
				"a3_PERSONAL_ACCOUNT_1_1": {Name: "a3_PERSONAL_ACCOUNT_1_1", Namespace: "", Value: "012345678", IsNil: false},
				"a3_TERMINALID_2_1":       {Name: "a3_TERMINALID_2_1", Namespace: "", Value: "testTerminal01", IsNil: false},
				"a3__CHECK_TEXT__3_1":     {Name: "a3__CHECK_TEXT__3_1", Namespace: "", Value: "", IsNil: false},
			},
		},
		Binds: []*FormBind{
			{
				Nodeset:     "a3_PERSONAL_ACCOUNT_1_1",
				TypeQName:   sp("a-3:PERSONAL_ACCOUNT_1_1"),
				ExtType:     sp("PERSONAL_ACCOUNT"),
				Required:    true,
				Readonly:    false,
				Relevant:    true,
				RawRequired: "true",
				RawReadonly: "false",
				RawRelevant: "true()",
			},
			{
				Nodeset:     "a3_TERMINALID_2_1",
				TypeQName:   sp("a-3:TERMINALID_2_1"),
				ExtType:     sp("PARAMETER"),
				Required:    true,
				Readonly:    false,
				Relevant:    false,
				RawRequired: "true",
				RawReadonly: "false",
				RawRelevant: "false()",
			},
			{
				Nodeset:     "a3__CHECK_TEXT__3_1",
				TypeQName:   sp("a-3:_CHECK_TEXT__3_1"),
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
			{ID: "a-3.submission.back", Action: "http://localhost/", Method: "get"},
			{ID: "a-3.submission.next", Action: "http://localhost/", Method: "get"},
			{ID: "a-3.submission.pay", Action: "http://localhost/", Method: "get"},
			{ID: "a-3.submission.link", Action: "http://localhost/", Method: "get"},
		},
		Body: FormBody{
			Elements: []FormBodyElement{
				&FormBodyGroup{
					ID:    sp("10"),
					Label: "",
					Elements: []FormBodyElement{
						&FormBodyInput{
							ID:          sp("PERSONAL_ACCOUNT_1_1"),
							Ref:         "a3_PERSONAL_ACCOUNT_1_1",
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
