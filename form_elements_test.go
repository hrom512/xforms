package xforms

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
					BaseInput: BaseInput{
						Name:     "a3_PERSONAL_ACCOUNT_1_1",
						Label:    "Personal account number:",
						ExtType:  sp("PERSONAL_ACCOUNT"),
						Required: true,
						Readonly: false,
						SimpleType: &FormSchemaSimpleType{
							Name:          "PERSONAL_ACCOUNT_1_1",
							BaseTypeQName: "xsd:string",
							Pattern:       sp("^\\d{10}$"),
						},
						ComplexType: nil,
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

