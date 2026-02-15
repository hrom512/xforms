package xforms

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestForm_Fill_UpdatesOnlyProvidedFields(t *testing.T) {
	f, err := Parse(strings.NewReader(sampleXFormsXML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	beforeTx := f.Instance.Fields["transactionId"].Value

	newVal := "0123456789"
	err = f.Fill([]FormElement{
		&TextInput{
			Name:  "field_PERSONAL_ACCOUNT",
			Value: &newVal,
		},
	})
	if err != nil {
		t.Fatalf("Fill() error: %v", err)
	}

	if got := strings.TrimSpace(f.Instance.Fields["field_PERSONAL_ACCOUNT"].Value); got != "0123456789" {
		t.Fatalf("unexpected updated value: %q", got)
	}
	if got := f.Instance.Fields["transactionId"].Value; got != beforeTx {
		t.Fatalf("expected transactionId unchanged; before=%q after=%q", beforeTx, got)
	}
}

func TestForm_ValidateAndFill_IsAtomic(t *testing.T) {
	xml := strings.Replace(sampleXFormsXML, ">012345678<", ">0123456789<", 1)
	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	orig := f.Instance.Fields["field_PERSONAL_ACCOUNT"].Value
	bad := "1"
	err = f.ValidateAndFill([]FormElement{
		&TextInput{
			Name:  "field_PERSONAL_ACCOUNT",
			Value: &bad,
		},
	})
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	// Must not update on error.
	if got := f.Instance.Fields["field_PERSONAL_ACCOUNT"].Value; got != orig {
		t.Fatalf("expected instance unchanged; before=%q after=%q", orig, got)
	}
}

func TestValidateAndFill_SuccessPath(t *testing.T) {
	// Fix sample so it validates, then update a required field with valid data.
	xml := strings.Replace(sampleXFormsXML, ">012345678<", ">0123456789<", 1)
	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	newVal := "1111111111"
	if err := f.ValidateAndFill([]FormElement{&TextInput{Name: "field_PERSONAL_ACCOUNT", Value: &newVal}}); err != nil {
		t.Fatalf("ValidateAndFill() error: %v", err)
	}
	if got := strings.TrimSpace(f.Instance.Fields["field_PERSONAL_ACCOUNT"].Value); got != "1111111111" {
		t.Fatalf("unexpected instance value: %q", got)
	}
}

func TestForm_Fill_ReadonlyAndTypes(t *testing.T) {
	dec, err := decimal.NewFromString("10.50")
	if err != nil {
		t.Fatalf("decimal.NewFromString: %v", err)
	}
	truth := true

	f := &Form{
		Schema: FormSchema{},
		Instance: FormInstance{
			Fields: map[string]FormInstanceField{
				"ro": {Name: "ro", Value: "old"},
			},
		},
		Binds: []*FormBind{
			{Nodeset: "ro", Readonly: true, Relevant: true},
			{Nodeset: "d", Readonly: false, Relevant: true},
			{Nodeset: "b", Readonly: false, Relevant: true},
			{Nodeset: "s", Readonly: false, Relevant: true},
		},
	}

	newVal := "new"
	if err := f.Fill([]FormElement{&TextInput{Name: "ro", Value: &newVal}}); err == nil {
		t.Fatalf("expected readonly error, got nil")
	}

	err = f.Fill([]FormElement{
		&DecimalInput{Name: "d", Value: &dec},
		&CheckboxInput{Name: "b", Value: &truth},
		&SelectInput{Name: "s", Value: &SelectOption{Label: "X", Value: "x"}},
		&FieldGroup{Label: "ignored"},
		&TextMessage{Message: "ignored"},
	})
	if err != nil {
		t.Fatalf("Fill() error: %v", err)
	}
	if got := f.Instance.Fields["d"].Value; got != "10.50" {
		t.Fatalf("unexpected decimal value: %q", got)
	}
	if got := f.Instance.Fields["b"].Value; got != "true" {
		t.Fatalf("unexpected boolean value: %q", got)
	}
	if got := f.Instance.Fields["s"].Value; got != "x" {
		t.Fatalf("unexpected select value: %q", got)
	}
}

func TestFill_NilValuesClearFields(t *testing.T) {
	f := &Form{
		Instance: FormInstance{
			Fields: map[string]FormInstanceField{
				"t": {Name: "t", Value: "x"},
				"d": {Name: "d", Value: "1.0"},
				"b": {Name: "b", Value: "true"},
				"s": {Name: "s", Value: "a"},
			},
		},
	}
	if err := f.Fill([]FormElement{
		&TextInput{Name: "t", Value: nil},
		&DecimalInput{Name: "d", Value: nil},
		&CheckboxInput{Name: "b", Value: nil},
		&SelectInput{Name: "s", Value: nil},
	}); err != nil {
		t.Fatalf("Fill() error: %v", err)
	}
	if f.Instance.Fields["t"].Value != "" || f.Instance.Fields["d"].Value != "" || f.Instance.Fields["b"].Value != "" || f.Instance.Fields["s"].Value != "" {
		t.Fatalf("expected cleared values, got: %#v", f.Instance.Fields)
	}
}
