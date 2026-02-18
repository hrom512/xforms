package componenttest

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hrom512/xforms"
	"github.com/shopspring/decimal"
)

func TestFullXFormsExample_Invalid_ValidateAndElements(t *testing.T) {
	xml := readFixture(t, "full_invalid.xforms.xml")

	f, err := xforms.Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	type snap struct {
		ElementNames        []string
		ValidationFieldKeys []string
	}
	var got snap
	got.ElementNames = flattenInputNames(f.Elements())

	vErr := f.Validate()
	ve, ok := vErr.(*xforms.ValidationError)
	if ok {
		got.ValidationFieldKeys = sortedKeys(ve.FieldErrors)
	}

	want := snap{
		// Elements() should omit relevant=false fields.
		ElementNames:        []string{"code", "username", "plan", "amount", "agree", "requiredEmpty", "noNil", "serverMessage", "obj"},
		ValidationFieldKeys: []string{"agree", "amount", "code", "noNil", "plan", "requiredEmpty", "username"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("invalid example snapshot mismatch (-want +got):\n%s", diff)
	}
}

func TestFullXFormsExample_Valid_EndToEnd(t *testing.T) {
	xml := readFixture(t, "full_valid.xforms.xml")

	f, err := xforms.Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if err := f.Validate(); err != nil {
		t.Fatalf("expected Validate() to succeed, got %T: %v", err, err)
	}

	type elementSnap struct {
		Kind     string
		Name     string
		Label    string
		Required bool
		Readonly bool

		TextValue    *string
		DecimalValue *string
		BoolValue    *bool
		SelectValue  *string
		Options      []xforms.SelectOption

		ComplexTypeName *string
		ComplexRaw      *string
		ComplexMap      map[string]string

		Message *string
	}
	type snapshot struct {
		GroupLabel string
		Elements   []elementSnap
	}

	els := f.Elements()
	group, ok := firstGroup(els)
	if !ok {
		t.Fatalf("expected top-level FieldGroup, got %T", first(els))
	}

	var got snapshot
	got.GroupLabel = strings.TrimSpace(group.Label)
	for _, el := range group.Elements {
		switch v := el.(type) {
		case *xforms.TextInput:
			var tv *string
			if v.Value != nil {
				s := strings.TrimSpace(*v.Value)
				tv = &s
			}
			got.Elements = append(got.Elements, elementSnap{
				Kind:      "text",
				Name:      v.Name,
				Label:     v.Label,
				Required:  v.Required,
				Readonly:  v.Readonly,
				TextValue: tv,
			})
		case *xforms.DecimalInput:
			var dv *string
			if v.Value != nil {
				s := v.Value.String()
				dv = &s
			}
			got.Elements = append(got.Elements, elementSnap{
				Kind:         "decimal",
				Name:         v.Name,
				Label:        v.Label,
				Required:     v.Required,
				Readonly:     v.Readonly,
				DecimalValue: dv,
			})
		case *xforms.CheckboxInput:
			got.Elements = append(got.Elements, elementSnap{
				Kind:     "checkbox",
				Name:     v.Name,
				Label:    v.Label,
				Required: v.Required,
				Readonly: v.Readonly,
				BoolValue: func() *bool {
					if v.Value == nil {
						return nil
					}
					b := *v.Value
					return &b
				}(),
			})
		case *xforms.SelectInput:
			var sv *string
			if v.Value != nil {
				s := v.Value.Value
				sv = &s
			}
			got.Elements = append(got.Elements, elementSnap{
				Kind:        "select",
				Name:        v.Name,
				Label:       v.Label,
				Required:    v.Required,
				Readonly:    v.Readonly,
				SelectValue: sv,
				Options:     v.Options,
			})
		case *xforms.ComplexInput:
			var ctn *string
			if v.ComplexType != nil {
				s := v.ComplexType.Name
				ctn = &s
			}
			got.Elements = append(got.Elements, elementSnap{
				Kind:            "complex",
				Name:            v.Name,
				Label:           v.Label,
				Required:        v.Required,
				Readonly:        v.Readonly,
				ComplexTypeName: ctn,
				ComplexRaw:      v.RawValue,
				ComplexMap:      v.Value,
			})
		case *xforms.TextMessage:
			m := v.Message
			got.Elements = append(got.Elements, elementSnap{
				Kind:    "output",
				Message: &m,
			})
		}
	}

	want := snapshot{
		GroupLabel: "Full demo",
		Elements: []elementSnap{
			{
				Kind:      "text",
				Name:      "code",
				Label:     "Code",
				Required:  true,
				Readonly:  false,
				TextValue: strPtr("AB1234"),
			},
			{
				Kind:      "text",
				Name:      "username",
				Label:     "Username",
				Required:  true,
				Readonly:  false,
				TextValue: strPtr("John"),
			},
			{
				Kind:        "select",
				Name:        "plan",
				Label:       "Plan",
				Required:    true,
				Readonly:    false,
				SelectValue: strPtr("basic"),
				Options: []xforms.SelectOption{
					{Label: "Basic", Value: "basic"},
					{Label: "Pro", Value: "pro"},
				},
			},
			{
				Kind:         "decimal",
				Name:         "amount",
				Label:        "Amount",
				Required:     true,
				Readonly:     false,
				DecimalValue: strPtr("1.23"),
			},
			{
				Kind:      "checkbox",
				Name:      "agree",
				Label:     "Agree",
				Required:  true,
				Readonly:  false,
				BoolValue: boolPtr(true)},
			{
				Kind:      "text",
				Name:      "requiredEmpty",
				Label:     "Required (non-empty)",
				Required:  true,
				Readonly:  false,
				TextValue: strPtr("x"),
			},
			{
				Kind:      "text",
				Name:      "noNil",
				Label:     "Not nillable",
				Required:  false,
				Readonly:  false,
				TextValue: strPtr("ok"),
			},
			{
				Kind:      "text",
				Name:      "serverMessage",
				Label:     "Server message",
				Required:  false,
				Readonly:  true,
				TextValue: strPtr("From server"),
			},
			{
				Kind:            "complex",
				Name:            "obj",
				Label:           "Complex object",
				Required:        false,
				Readonly:        false,
				ComplexTypeName: strPtr("ObjType"),
				ComplexRaw:      strPtr("<a>demo</a><sum>1.00</sum><check>true</check>"),
				ComplexMap: map[string]string{"a": "demo",
					"sum":   "1.00",
					"check": "true"},
			},
			{
				Kind:    "output",
				Message: strPtr("From server"),
			},
			{
				Kind:    "output",
				Message: strPtr("Static output message"),
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("valid example Elements() snapshot mismatch (-want +got):\n%s", diff)
	}

	// Check instance on start
	inst := instanceValuesSnapshot(f.Instance.Fields)
	wantInst := map[string]string{
		"code":          "AB1234",
		"username":      "John",
		"plan":          "basic",
		"amount":        "1.23",
		"agree":         "true",
		"requiredEmpty": "x",
		"noNil":         "ok",
		"serverMessage": "From server",
		"hiddenBad":     "XX0000",
		"obj":           "<a>demo</a><sum>1.00</sum><check>true</check>",
	}
	if diff := cmp.Diff(wantInst, inst); diff != "" {
		t.Fatalf("Instance snapshot mismatch (-want +got):\n%s", diff)
	}

	// Validate original form
	if err := f.Validate(); err != nil {
		t.Fatalf("Validate() with err: %v", err)
	}

	// Fill: update a couple of fields (including decimal).
	newCode := "ZZ9999"
	newPlan := "pro"
	newAmountStr := "2.50"
	newAmount := mustDecimal(newAmountStr)
	if err := f.Fill([]xforms.FormElement{
		&xforms.TextInput{Name: "code", Value: &newCode},
		&xforms.SelectInput{Name: "plan", Value: &xforms.SelectOption{Value: newPlan}},
		&xforms.DecimalInput{Name: "amount", Value: &newAmount},
	}); err != nil {
		t.Fatalf("Fill() error: %v", err)
	}

	// Fill complex type with raw value
	newObj := "<a>demo1</a><sum>1.00</sum><check>true</check>"
	if err := f.Fill([]xforms.FormElement{
		&xforms.ComplexInput{Name: "obj", RawValue: &newObj},
	}); err != nil {
		t.Fatalf("Fill() with obj error: %v", err)
	}

	// Fill complex type with parsed value
	newObj2Str := "<a>demo2</a><check>false</check><sum>2.00</sum>"
	newObj2 := map[string]string{
		"a":     "demo2",
		"check": "false",
		"sum":   "2.00",
	}
	if err := f.Fill([]xforms.FormElement{
		&xforms.ComplexInput{Name: "obj", Value: newObj2},
	}); err != nil {
		t.Fatalf("Fill() with obj map error: %v", err)
	}

	// Filling a readonly field must fail.
	newMsg := "client overwrite"
	if err := f.Fill([]xforms.FormElement{&xforms.TextInput{Name: "serverMessage", Value: &newMsg}}); err == nil {
		t.Fatalf("expected readonly fill error, got nil")
	}

	// ValidateAndFill must be atomic.
	badCode := "bad"
	before := f.Instance.Clone()
	if err := f.ValidateAndFill([]xforms.FormElement{&xforms.TextInput{Name: "code", Value: &badCode}}); err == nil {
		t.Fatalf("expected ValidateAndFill() to fail, got nil")
	}
	if diff := cmp.Diff(before, f.Instance); diff != "" {
		t.Fatalf("expected instance unchanged on failed ValidateAndFill (-before +after):\n%s", diff)
	}

	// Try ValidateAndFill SelectInput with undefined option
	newInvalidPlan := "invalid"
	if err := f.ValidateAndFill([]xforms.FormElement{
		&xforms.SelectInput{Name: "plan", Value: &xforms.SelectOption{Value: newInvalidPlan}},
	}); err == nil {
		t.Fatalf("expected readonly fill error, got nil")
	}

	// Check result instance
	inst = instanceValuesSnapshot(f.Instance.Fields)
	wantInst = map[string]string{
		"code":          newCode,
		"username":      "John",
		"plan":          newPlan,
		"amount":        newAmountStr,
		"agree":         "true",
		"requiredEmpty": "x",
		"noNil":         "ok",
		"serverMessage": "From server",
		"hiddenBad":     "XX0000",
		"obj":           newObj2Str,
	}
	if diff := cmp.Diff(wantInst, inst); diff != "" {
		t.Fatalf("instance snapshot mismatch after ValidateAndFill (-want +got):\n%s", diff)
	}

	// Validate result form
	if err := f.Validate(); err != nil {
		t.Fatalf("Validate() with err: %v", err)
	}
}

// --- helpers ---

func readFixture(t *testing.T, name string) string {
	t.Helper()
	if filepath.Base(name) != name {
		t.Fatalf("invalid fixture name %q", name)
	}
	//nolint:gosec // fixture name is validated and read from testdata
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}
	return string(b)
}

func flattenInputNames(els []xforms.FormElement) []string {
	var out []string
	var walk func(e xforms.FormElement)
	walk = func(e xforms.FormElement) {
		switch v := e.(type) {
		case *xforms.FieldGroup:
			for _, c := range v.Elements {
				walk(c)
			}
		case *xforms.TextInput:
			out = append(out, v.Name)
		case *xforms.DecimalInput:
			out = append(out, v.Name)
		case *xforms.CheckboxInput:
			out = append(out, v.Name)
		case *xforms.SelectInput:
			out = append(out, v.Name)
		case *xforms.ComplexInput:
			out = append(out, v.Name)
		}
	}
	for _, e := range els {
		walk(e)
	}
	return out
}

func sortedKeys(m map[string][]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func firstGroup(els []xforms.FormElement) (*xforms.FieldGroup, bool) {
	if len(els) == 0 {
		return nil, false
	}
	g, ok := els[0].(*xforms.FieldGroup)
	return g, ok
}

func first[T any](s []T) T {
	var zero T
	if len(s) == 0 {
		return zero
	}
	return s[0]
}

func mustDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

func instanceValuesSnapshot(fields map[string]xforms.FormInstanceField) map[string]string {
	out := map[string]string{}
	for k, v := range fields {
		out[k] = strings.TrimSpace(v.Value)
	}
	return out
}
