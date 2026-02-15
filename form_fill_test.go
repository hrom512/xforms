package xforms

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func TestForm_Fill_UpdatesOnlyProvidedFields(t *testing.T) {
	f, err := Parse(strings.NewReader(sampleXFormsXML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	before := f.Instance.Clone()

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

	want := before.Clone()
	v := want.Fields["field_PERSONAL_ACCOUNT"]
	v.Value = newVal
	want.Fields["field_PERSONAL_ACCOUNT"] = v

	if diff := cmp.Diff(want, f.Instance); diff != "" {
		t.Fatalf("Instance mismatch after Fill() (-want +got):\n%s", diff)
	}
}

func TestForm_ValidateAndFill_IsAtomic(t *testing.T) {
	xml := strings.Replace(sampleXFormsXML, ">012345678<", ">0123456789<", 1)
	f, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	before := f.Instance.Clone()
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
	if diff := cmp.Diff(before, f.Instance); diff != "" {
		t.Fatalf("expected instance unchanged on failed ValidateAndFill() (-before +after):\n%s", diff)
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
	before := f.Instance.Clone()
	if err := f.ValidateAndFill([]FormElement{&TextInput{Name: "field_PERSONAL_ACCOUNT", Value: &newVal}}); err != nil {
		t.Fatalf("ValidateAndFill() error: %v", err)
	}

	want := before.Clone()
	v := want.Fields["field_PERSONAL_ACCOUNT"]
	v.Value = newVal
	want.Fields["field_PERSONAL_ACCOUNT"] = v

	if diff := cmp.Diff(want, f.Instance); diff != "" {
		t.Fatalf("Instance mismatch after successful ValidateAndFill() (-want +got):\n%s", diff)
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

	want := FormInstance{
		Fields: map[string]FormInstanceField{
			"ro": {Name: "ro", Value: "old"},
			"d":  {Name: "d", Value: "10.50"},
			"b":  {Name: "b", Value: "true"},
			"s":  {Name: "s", Value: "x"},
		},
	}
	if diff := cmp.Diff(want, f.Instance); diff != "" {
		t.Fatalf("Instance mismatch after Fill types (-want +got):\n%s", diff)
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

	want := FormInstance{
		Fields: map[string]FormInstanceField{
			"t": {Name: "t", Value: ""},
			"d": {Name: "d", Value: ""},
			"b": {Name: "b", Value: ""},
			"s": {Name: "s", Value: ""},
		},
	}
	if diff := cmp.Diff(want, f.Instance); diff != "" {
		t.Fatalf("Instance mismatch after clearing (-want +got):\n%s", diff)
	}
}

func TestForm_Fill_ComplexInput_RawValueWinsOverMap(t *testing.T) {
	raw := "<a>1</a><sum>2.00</sum>"
	f := &Form{
		Instance: FormInstance{
			Fields: map[string]FormInstanceField{
				"obj": {Name: "obj", Value: "old"},
			},
		},
		Binds: []*FormBind{
			{Nodeset: "obj", Readonly: false, Relevant: true},
		},
	}
	err := f.Fill([]FormElement{
		&ComplexInput{
			Name:     "obj",
			RawValue: &raw,
			Value: map[string]string{
				"a": "ignored",
			},
		},
	})
	if err != nil {
		t.Fatalf("Fill() error: %v", err)
	}
	want := FormInstance{
		Fields: map[string]FormInstanceField{
			"obj": {Name: "obj", Value: raw},
		},
	}
	if diff := cmp.Diff(want, f.Instance); diff != "" {
		t.Fatalf("Instance mismatch (-want +got):\n%s", diff)
	}
}

func TestForm_Fill_ComplexInput_MapIsSerialized_OrderedAndEscaped(t *testing.T) {
	ct := &FormSchemaComplexType{
		Name: "C",
		All: []FormSchemaElementDecl{
			{Name: "a"},
			{Name: "sum"},
			{Name: "check"},
		},
	}
	f := &Form{
		Instance: FormInstance{Fields: map[string]FormInstanceField{}},
		Binds:    []*FormBind{{Nodeset: "obj", Readonly: false, Relevant: true}},
	}

	err := f.Fill([]FormElement{
		&ComplexInput{
			Name:        "obj",
			ComplexType: ct,
			Value: map[string]string{
				"a":     "x & y",
				"sum":   "1.00",
				"check": "true",
				"extra": "<z/>",
			},
		},
	})
	if err != nil {
		t.Fatalf("Fill() error: %v", err)
	}

	want := FormInstance{
		Fields: map[string]FormInstanceField{
			"obj": {Name: "obj", Value: "<a>x &amp; y</a><sum>1.00</sum><check>true</check><extra><z/></extra>"},
		},
	}
	if diff := cmp.Diff(want, f.Instance); diff != "" {
		t.Fatalf("Instance mismatch (-want +got):\n%s", diff)
	}
}

func TestForm_Fill_CreatesFieldsMapWhenNil(t *testing.T) {
	f := &Form{
		Instance: FormInstance{}, // Fields is nil
		Binds:    []*FormBind{{Nodeset: "t", Readonly: false, Relevant: true}},
	}
	v := "x"
	if err := f.Fill([]FormElement{&TextInput{Name: "t", Value: &v}}); err != nil {
		t.Fatalf("Fill() error: %v", err)
	}
	want := FormInstance{
		Fields: map[string]FormInstanceField{
			"t": {Name: "t", Value: "x"},
		},
	}
	if diff := cmp.Diff(want, f.Instance); diff != "" {
		t.Fatalf("Instance mismatch (-want +got):\n%s", diff)
	}
}

type unknownFormElement struct{}

func (*unknownFormElement) FormElement() {}

func TestForm_Fill_UnknownElementIsIgnored(t *testing.T) {
	f := &Form{
		Instance: FormInstance{
			Fields: map[string]FormInstanceField{
				"t": {Name: "t", Value: "x"},
			},
		},
	}
	before := f.Instance.Clone()
	if err := f.Fill([]FormElement{&unknownFormElement{}}); err != nil {
		t.Fatalf("Fill() error: %v", err)
	}
	if diff := cmp.Diff(before, f.Instance); diff != "" {
		t.Fatalf("expected instance unchanged when unknown element passed (-before +after):\n%s", diff)
	}
}
