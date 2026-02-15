package xforms

import (
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// FormElement is a convenient representation of a form element
type FormElement interface {
	FormElement()
}

type BaseInput struct {
	Name  string
	Label string

	ExtType *string

	Required bool
	Readonly bool

	// One of SimpleType or ComplexType is filled
	SimpleType  *FormSchemaSimpleType
	ComplexType *FormSchemaComplexType
}

type TextInput struct {
	BaseInput

	Alert string
	Help  string
	Hint  string

	Value *string
}

func (*TextInput) FormElement() {}

type DecimalInput struct {
	BaseInput

	Alert string
	Help  string
	Hint  string

	Value *decimal.Decimal
}

func (*DecimalInput) FormElement() {}

type CheckboxInput struct {
	BaseInput

	Value *bool
}

func (*CheckboxInput) FormElement() {}

type SelectInput struct {
	BaseInput

	Options []SelectOption

	Value *SelectOption
}

type SelectOption struct {
	Label string
	Value string
}

func (*SelectInput) FormElement() {}

type TextMessage struct {
	Message string
	ID      *string
}

func (*TextMessage) FormElement() {}

type FieldGroup struct {
	Label    string
	Elements []FormElement
}

func (*FieldGroup) FormElement() {}

// Elements parses form and returns a convenient list of form elements
func (f *Form) Elements() []FormElement {
	bindByField := map[string]*FormBind{}
	for _, b := range f.Binds {
		if b == nil {
			continue
		}
		field := fieldNameFromRef(b.Nodeset)
		if field == "" {
			continue
		}
		// First bind wins (stable behavior).
		if _, ok := bindByField[field]; !ok {
			bindByField[field] = b
		}
	}

	var out []FormElement
	for _, el := range f.Body.Elements {
		if conv := f.convertBodyElement(el, bindByField); conv != nil {
			out = append(out, conv)
		}
	}
	return out
}

func (f *Form) convertBodyElement(el FormBodyElement, bindByField map[string]*FormBind) FormElement {
	switch t := el.(type) {
	case *FormBodyGroup:
		g := &FieldGroup{Label: t.Label}
		for _, child := range t.Elements {
			if c := f.convertBodyElement(child, bindByField); c != nil {
				g.Elements = append(g.Elements, c)
			}
		}
		// Keep empty groups; caller can decide whether to render them.
		return g

	case *FormBodyInput:
		name := fieldNameFromRef(t.Ref)
		if name == "" {
			return nil
		}

		b := bindByField[name]
		if b != nil && !b.Relevant {
			return nil
		}

		base := BaseInput{
			Name:     name,
			Label:    t.Label,
			ExtType:  nil,
			Required: false,
			Readonly: false,
		}
		if b != nil {
			base.ExtType = b.ExtType
			base.Required = b.Required
			base.Readonly = b.Readonly
		}
		base.SimpleType, base.ComplexType = f.resolveSchemaType(b, name)

		// Choose element type from base simple type.
		if st := base.SimpleType; st != nil {
			switch qnameLocal(st.BaseTypeQName) {
			case "boolean":
				ci := &CheckboxInput{BaseInput: base}
				if v, ok := f.instanceFieldValue(name); ok {
					txt := strings.TrimSpace(v)
					if txt != "" {
						if bv, err := strconv.ParseBool(txt); err == nil {
							ci.Value = &bv
						}
					}
				}
				return ci
			case "decimal":
				di := &DecimalInput{
					BaseInput: base,
					Alert:     t.Alert,
					Help:      t.Help,
					Hint:      t.Hint,
				}
				if v, ok := f.instanceFieldValue(name); ok {
					txt := strings.TrimSpace(v)
					if txt != "" {
						if dv, err := decimal.NewFromString(txt); err == nil {
							di.Value = &dv
						}
					}
				}
				return di
			}
		}

		ti := &TextInput{
			BaseInput: base,
			Alert:     t.Alert,
			Help:      t.Help,
			Hint:      t.Hint,
		}
		if v, ok := f.instanceFieldValue(name); ok {
			txt := strings.TrimSpace(v)
			ti.Value = &txt
		}
		return ti

	case *FormBodySelect:
		name := fieldNameFromRef(t.Ref)
		if name == "" {
			return nil
		}

		b := bindByField[name]
		if b != nil && !b.Relevant {
			return nil
		}

		base := BaseInput{
			Name:     name,
			Label:    t.Label,
			ExtType:  nil,
			Required: false,
			Readonly: false,
		}
		if b != nil {
			base.ExtType = b.ExtType
			base.Required = b.Required
			base.Readonly = b.Readonly
		}
		base.SimpleType, base.ComplexType = f.resolveSchemaType(b, name)

		si := &SelectInput{
			BaseInput: base,
		}
		for _, it := range t.Items {
			si.Options = append(si.Options, SelectOption{Label: it.Label, Value: it.Value})
		}
		if v, ok := f.instanceFieldValue(name); ok {
			txt := strings.TrimSpace(v)
			for i := range si.Options {
				if si.Options[i].Value == txt {
					si.Value = &si.Options[i]
					break
				}
			}
		}
		return si

	case *FormBodyOutput:
		msg := strings.TrimSpace(t.Label)
		if t.Ref != nil {
			name := fieldNameFromRef(*t.Ref)
			if v, ok := f.instanceFieldValue(name); ok {
				msg = strings.TrimSpace(v)
			}
		}
		return &TextMessage{Message: msg, ID: t.ID}

	default:
		return nil
	}
}

func (f *Form) instanceFieldValue(fieldName string) (string, bool) {
	if f.Instance.Fields == nil {
		return "", false
	}
	v, ok := f.Instance.Fields[fieldName]
	if !ok {
		return "", false
	}
	return v.Value, true
}

func (f *Form) resolveSchemaType(b *FormBind, fieldName string) (*FormSchemaSimpleType, *FormSchemaComplexType) {
	// Try explicit bind type first.
	if b != nil && b.TypeQName != nil && strings.TrimSpace(*b.TypeQName) != "" {
		return f.resolveSchemaTypeQName(*b.TypeQName)
	}
	// Fallback to schema element declaration by field name.
	if f.Schema.Elements != nil {
		if ed, ok := f.Schema.Elements[fieldName]; ok && ed != nil {
			if ed.ComplexType != nil {
				return nil, ed.ComplexType
			}
			if strings.TrimSpace(ed.TypeQName) != "" {
				return f.resolveSchemaTypeQName(ed.TypeQName)
			}
		}
	}
	return nil, nil
}

func (f *Form) resolveSchemaTypeQName(typeQName string) (*FormSchemaSimpleType, *FormSchemaComplexType) {
	local := qnameLocal(strings.TrimSpace(typeQName))
	if local == "" {
		return nil, nil
	}

	// Built-in XSD types are represented as a synthetic simple type.
	switch local {
	case "string", "decimal", "boolean":
		return &FormSchemaSimpleType{
			Name:          local,
			BaseTypeQName: "xsd:" + local,
		}, nil
	}

	if f.Schema.SimpleTypes != nil {
		if st, ok := f.Schema.SimpleTypes[local]; ok {
			return st, nil
		}
	}
	if f.Schema.ComplexTypes != nil {
		if ct, ok := f.Schema.ComplexTypes[local]; ok {
			return nil, ct
		}
	}
	return nil, nil
}

func fieldNameFromRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	// Scope is "direct child element names", but allow basic path forms.
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	ref = strings.TrimSpace(ref)
	return qnameLocal(ref)
}
