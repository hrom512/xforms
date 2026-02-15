package xforms

import (
	"strings"
	"testing"
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
			BaseInput: BaseInput{Name: "a3_PERSONAL_ACCOUNT_1_1"},
			Value:     &newVal,
		},
	})
	if err != nil {
		t.Fatalf("Fill() error: %v", err)
	}

	if got := strings.TrimSpace(f.Instance.Fields["a3_PERSONAL_ACCOUNT_1_1"].Value); got != "0123456789" {
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

	orig := f.Instance.Fields["a3_PERSONAL_ACCOUNT_1_1"].Value
	bad := "1"
	err = f.ValidateAndFill([]FormElement{
		&TextInput{
			BaseInput: BaseInput{Name: "a3_PERSONAL_ACCOUNT_1_1"},
			Value:     &bad,
		},
	})
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	// Must not update on error.
	if got := f.Instance.Fields["a3_PERSONAL_ACCOUNT_1_1"].Value; got != orig {
		t.Fatalf("expected instance unchanged; before=%q after=%q", orig, got)
	}
}
