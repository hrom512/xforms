package xforms

import (
	"fmt"
	"strconv"
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
	tmp.Instance = f.Instance.Clone()
	tmp.applyInstanceUpdatesTo(&tmp.Instance, updates)

	if err := tmp.Validate(); err != nil {
		return err
	}

	f.Instance = tmp.Instance
	return nil
}

func (f *Form) collectFillUpdates(fieldValues []FormElement) (map[string]string, error) {
	bindByField := map[string]*FormBind{}
	for _, b := range f.Binds {
		if b == nil {
			continue
		}
		field := fieldNameFromRef(b.Nodeset)
		if field == "" {
			continue
		}
		if _, ok := bindByField[field]; !ok {
			bindByField[field] = b
		}
	}

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

		case *FieldGroup, *TextMessage:
			// ignore
		default:
			// Unknown element types are ignored for forward compatibility.
		}
	}
	return updates, nil
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
