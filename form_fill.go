package xforms

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Fill adds the passed field values to the contents of FormInstance
func (f *Form) Fill(fieldValues []FormElement) error {
	updates, err := f.collectFillUpdates(fieldValues)
	if err != nil {
		return err
	}
	f.applyInstanceUpdates(updates)
	return nil
}

// ValidateAndFill validates the update (as if applied) and only commits it if valid.
func (f *Form) ValidateAndFill(fieldValues []FormElement) error {
	updates, err := f.collectFillUpdates(fieldValues)
	if err != nil {
		return err
	}

	tmp := *f
	tmp.Instance = f.Instance.clone()
	tmp.applyInstanceUpdatesTo(&tmp.Instance, updates)

	if err := tmp.Validate(); err != nil {
		return err
	}

	f.Instance = tmp.Instance
	return nil
}

func (f *Form) collectFillUpdates(fieldValues []FormElement) (map[string]string, error) {
	bindByField := f.indexBindsByField()

	updates := map[string]string{}
	for _, el := range fieldValues {
		switch v := el.(type) {
		case *TextInput:
			name := v.Name
			if name == "" {
				continue
			}
			if b := bindByField[name]; b != nil && b.Readonly {
				return nil, fmt.Errorf("field %q is readonly", name)
			}
			if v.Value == nil {
				updates[name] = ""
			} else {
				updates[name] = *v.Value
			}

		case *DecimalInput:
			name := v.Name
			if name == "" {
				continue
			}
			if b := bindByField[name]; b != nil && b.Readonly {
				return nil, fmt.Errorf("field %q is readonly", name)
			}
			if v.Value == nil {
				updates[name] = ""
			} else {
				// Preserve the scale from the Decimal value.
				// shopspring/decimal's String() trims trailing fractional zeros, while
				// StringFixed(-Exponent()) keeps them (when Exponent() <= 0).
				exp := v.Value.Exponent()
				if exp <= 0 {
					updates[name] = v.Value.StringFixed(-exp)
				} else {
					updates[name] = v.Value.String()
				}
			}

		case *CheckboxInput:
			name := v.Name
			if name == "" {
				continue
			}
			if b := bindByField[name]; b != nil && b.Readonly {
				return nil, fmt.Errorf("field %q is readonly", name)
			}
			if v.Value == nil {
				updates[name] = ""
			} else {
				updates[name] = strconv.FormatBool(*v.Value)
			}

		case *SelectInput:
			name := v.Name
			if name == "" {
				continue
			}
			if b := bindByField[name]; b != nil && b.Readonly {
				return nil, fmt.Errorf("field %q is readonly", name)
			}
			if v.Value == nil {
				updates[name] = ""
			} else {
				updates[name] = v.Value.Value
			}

		case *ComplexInput:
			name := v.Name
			if name == "" {
				continue
			}
			if b := bindByField[name]; b != nil && b.Readonly {
				return nil, fmt.Errorf("field %q is readonly", name)
			}
			if v.RawValue != nil {
				updates[name] = *v.RawValue
				continue
			}
			if v.Value == nil {
				updates[name] = ""
				continue
			}
			updates[name] = complexRawValueFromMap(v.Value, v.ComplexType)

		case *FieldGroup, *TextMessage:
			// ignore
		default:
			// Unknown element types are ignored for forward compatibility.
		}
	}
	return updates, nil
}

func complexRawValueFromMap(m map[string]string, ct *FormSchemaComplexType) string {
	if len(m) == 0 {
		return ""
	}

	used := map[string]struct{}{}
	var ordered []string
	if ct != nil {
		for _, decl := range ct.All {
			k := qnameLocal(strings.TrimSpace(decl.Name))
			if k == "" {
				continue
			}
			if _, ok := m[k]; ok {
				ordered = append(ordered, k)
				used[k] = struct{}{}
			}
		}
	}

	var rest []string
	for k := range m {
		if _, ok := used[k]; ok {
			continue
		}
		rest = append(rest, k)
	}
	sort.Strings(rest)
	ordered = append(ordered, rest...)

	var buf bytes.Buffer
	for _, k := range ordered {
		v := m[k]
		buf.WriteString("<")
		buf.WriteString(k)
		buf.WriteString(">")
		if strings.Contains(v, "<") {
			// Treat as inner XML.
			buf.WriteString(v)
		} else {
			// Treat as text.
			_ = xml.EscapeText(&buf, []byte(v))
		}
		buf.WriteString("</")
		buf.WriteString(k)
		buf.WriteString(">")
	}
	return buf.String()
}

func (f *Form) applyInstanceUpdates(updates map[string]string) {
	f.applyInstanceUpdatesTo(&f.Instance, updates)
}

func (f *Form) applyInstanceUpdatesTo(inst *FormInstance, updates map[string]string) {
	if inst.Fields == nil {
		inst.Fields = map[string]FormInstanceField{}
	}
	for name, value := range updates {
		cur := inst.Fields[name]
		cur.Name = name
		cur.Value = value
		cur.IsNil = false
		inst.Fields[name] = cur
	}
}
