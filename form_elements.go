package xforms

import (
	"encoding/xml"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// FormElement is a convenient representation of a form element
type FormElement interface {
	FormElement()
}

type TextInput struct {
	Name  string
	Label string

	ExtType *string

	Required bool
	Readonly bool

	SimpleType *FormSchemaSimpleType

	Alert string
	Help  string
	Hint  string

	Value *string
}

func (*TextInput) FormElement() {}

type DecimalInput struct {
	Name  string
	Label string

	ExtType *string

	Required bool
	Readonly bool

	SimpleType *FormSchemaSimpleType

	Alert string
	Help  string
	Hint  string

	Value *decimal.Decimal
}

func (*DecimalInput) FormElement() {}

type CheckboxInput struct {
	Name  string
	Label string

	ExtType *string

	Required bool
	Readonly bool

	SimpleType *FormSchemaSimpleType

	Value *bool
}

func (*CheckboxInput) FormElement() {}

type SelectInput struct {
	Name  string
	Label string

	ExtType *string

	Required bool
	Readonly bool

	SimpleType *FormSchemaSimpleType

	Options []SelectOption

	Value *SelectOption
}

type SelectOption struct {
	Label string
	Value string
}

func (*SelectInput) FormElement() {}

// ComplexInput represents an input bound to an XSD complexType.
type ComplexInput struct {
	Name  string
	Label string

	ExtType *string

	Required bool
	Readonly bool

	ComplexType *FormSchemaComplexType

	Alert string
	Help  string
	Hint  string

	// RawValue is the instance field content (inner XML) as a string.
	// For values represented as nested tags, it is in stable "innerxml" form
	// (whitespace between tags is normalized).
	RawValue *string

	// Value is a best-effort map of direct nested tag values under this field.
	// Keys are local tag names; values are tag inner XML (trimmed).
	// If RawValue does not contain nested XML or parsing fails, Value is nil.
	Value map[string]string
}

func (*ComplexInput) FormElement() {}

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

		var extType *string
		required := false
		readonly := false
		if b != nil {
			extType = b.ExtType
			required = b.Required
			readonly = b.Readonly
		}
		simpleType, complexType := f.resolveSchemaType(b, name)

		// Complex type input: keep instance value as inner XML.
		if simpleType == nil && complexType != nil {
			ci := &ComplexInput{
				Name:        name,
				Label:       t.Label,
				ExtType:     extType,
				Required:    required,
				Readonly:    readonly,
				ComplexType: complexType,
				Alert:       t.Alert,
				Help:        t.Help,
				Hint:        t.Hint,
			}
			if v, ok := f.instanceFieldValue(name); ok {
				txt := strings.TrimSpace(v)
				if txt != "" {
					ci.RawValue = &txt
					ci.Value = complexValueMapFromRaw(txt)
				}
			}
			return ci
		}

		// Choose element type from base simple type.
		if st := simpleType; st != nil {
			switch qnameLocal(st.BaseTypeQName) {
			case "boolean":
				ci := &CheckboxInput{
					Name:       name,
					Label:      t.Label,
					ExtType:    extType,
					Required:   required,
					Readonly:   readonly,
					SimpleType: simpleType,
				}
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
					Name:       name,
					Label:      t.Label,
					ExtType:    extType,
					Required:   required,
					Readonly:   readonly,
					SimpleType: simpleType,
					Alert:      t.Alert,
					Help:       t.Help,
					Hint:       t.Hint,
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
			Name:       name,
			Label:      t.Label,
			ExtType:    extType,
			Required:   required,
			Readonly:   readonly,
			SimpleType: simpleType,
			Alert:      t.Alert,
			Help:       t.Help,
			Hint:       t.Hint,
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

		var extType *string
		required := false
		readonly := false
		if b != nil {
			extType = b.ExtType
			required = b.Required
			readonly = b.Readonly
		}
		simpleType, _ := f.resolveSchemaType(b, name)

		si := &SelectInput{
			Name:       name,
			Label:      t.Label,
			ExtType:    extType,
			Required:   required,
			Readonly:   readonly,
			SimpleType: simpleType,
		}
		for _, it := range t.Items {
			si.Options = append(si.Options, SelectOption(it))
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
	// Fallback to schema element declaration within the instance root element.
	if decl := f.schemaDeclForInstanceField(fieldName); decl != nil {
		if decl.ComplexType != nil {
			return nil, decl.ComplexType
		}
		if strings.TrimSpace(decl.TypeQName) != "" {
			return f.resolveSchemaTypeQName(decl.TypeQName)
		}
	}
	// Fallback to top-level schema element declaration by field name.
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

func complexValueMapFromRaw(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.Contains(raw, "<") {
		return nil
	}

	// Wrap the inner XML into a synthetic root to make it well-formed.
	wrapped := "<root>" + raw + "</root>"
	d := xml.NewDecoder(strings.NewReader(wrapped))

	out := map[string]string{}
	var depth int
	for {
		tok, err := d.Token()
		if err != nil {
			return nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			// Depth 1 is <root>, depth 2 are direct children.
			if depth == 2 {
				var inner struct {
					Inner string `xml:",innerxml"`
				}
				if err := d.DecodeElement(&inner, &t); err != nil {
					return nil
				}
				v := strings.TrimSpace(inner.Inner)
				if strings.Contains(v, "<") {
					v = intertagWhitespace.ReplaceAllString(v, "><")
				}
				out[qnameLocal(t.Name.Local)] = v
				depth-- // DecodeElement consumed the corresponding EndElement.
			}
		case xml.EndElement:
			depth--
			if depth == 0 {
				if len(out) == 0 {
					return nil
				}
				return out
			}
		}
	}
}
