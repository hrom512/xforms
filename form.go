package xforms

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Form is a parsed XForms document (schema, instance, binds, submissions, body).
type Form struct {
	Schema      FormSchema
	Instance    FormInstance
	Binds       []*FormBind
	Submissions []*FormSubmission
	Body        FormBody
}

// FormSchema contains parsed XSD schema information used for typing and validation.
type FormSchema struct {
	TargetNamespace string

	// All maps are keyed by local name (QName prefix is ignored).
	SimpleTypes  map[string]*FormSchemaSimpleType
	ComplexTypes map[string]*FormSchemaComplexType
	Elements     map[string]*FormSchemaElementDecl
}

// FormSchemaSimpleType represents an XSD simpleType with a supported subset of facets.
type FormSchemaSimpleType struct {
	Name          string
	BaseTypeQName string // e.g. xsd:string, xsd:decimal, xsd:boolean

	Pattern        *string
	Enumeration    []string
	MinLength      *int
	MaxLength      *int
	TotalDigits    *int
	FractionDigits *int

	// Numeric facets are kept as raw strings and interpreted according to base type.
	MinInclusive *string
	MaxInclusive *string
	MinExclusive *string
	MaxExclusive *string
}

// FormSchemaComplexType represents an XSD complexType (currently: xsd:all only).
type FormSchemaComplexType struct {
	Name string

	// For initial scope, we support xsd:all.
	All []FormSchemaElementDecl
}

// FormInstance stores the instance root and its direct child fields.
type FormInstance struct {
	Root   xml.Name
	Fields map[string]FormInstanceField // keyed by local name
}

// FormBind is a parsed xforms:bind entry.
type FormBind struct {
	ID      *string
	Nodeset string

	// TypeQName is either an xsd builtin type (xsd:string/decimal/boolean)
	// or a schema type (e.g. demo:PERSONAL_ACCOUNT). Prefix is preserved.
	TypeQName *string

	ExtType *string

	Required bool
	Readonly bool
	Relevant bool

	// Raw expressions as provided in XML (useful for debugging).
	RawRequired string
	RawReadonly string
	RawRelevant string
}

// FormSubmission is a parsed xforms:submission entry.
type FormSubmission struct {
	ID     string
	Action string
	Method string
}

// FormBody contains parsed body controls as internal elements.
type FormBody struct {
	Elements []FormBodyElement
}

// FormSchemaElementDecl is an XSD element declaration.
type FormSchemaElementDecl struct {
	Name      string
	TypeQName string // may be empty if ComplexType is provided inline
	Nillable  *bool

	ComplexType *FormSchemaComplexType
}

// FormInstanceField is a direct child field under instance root.
type FormInstanceField struct {
	Name  string
	Value string

	// IsNil stores the fact that the field has xsi:nil="true" set in the instance (i.e., the value is semantically "null" according to XSD), even if the text content is empty. This differs from a "simply empty string".
	IsNil bool
}

var intertagWhitespace = regexp.MustCompile(`>\s+<`)

// Clone returns a deep copy of the instance fields map.
func (fi FormInstance) Clone() FormInstance {
	out := FormInstance{
		Root:   fi.Root,
		Fields: map[string]FormInstanceField{},
	}
	for k, v := range fi.Fields {
		out.Fields[k] = v
	}
	return out
}

// FormBodyElement is an internal parsed representation of xforms controls.
// It is later converted into user-facing []FormElement by (*Form).Elements().
type FormBodyElement interface {
	formBodyElement()
}

// FormBodyGroup represents a parsed group control (xforms:group).
type FormBodyGroup struct {
	ID       *string
	Label    string
	Elements []FormBodyElement
}

func (*FormBodyGroup) formBodyElement() {}

// FormBodyInput represents a parsed input control (xforms:input).
type FormBodyInput struct {
	ID          *string
	Ref         string
	Incremental *bool

	Label string
	Alert string
	Help  string
	Hint  string
}

func (*FormBodyInput) formBodyElement() {}

// FormBodySelect represents a parsed select/select1 control.
type FormBodySelect struct {
	ID       *string
	Ref      string
	Multiple bool // true for xforms:select, false for xforms:select1

	Label string
	Alert string
	Help  string
	Hint  string

	Items []FormBodySelectItem
}

// FormBodySelectItem represents a select item (label/value).
type FormBodySelectItem struct {
	Label string
	Value string
}

func (*FormBodySelect) formBodyElement() {}

// FormBodyOutput represents a parsed output control (xforms:output).
type FormBodyOutput struct {
	ID  *string
	Ref *string

	// For simplicity, we treat output as a message. If Ref is set, message
	// can be taken from instance during conversion.
	Label string
}

func (*FormBodyOutput) formBodyElement() {}

// UnmarshalXML implements custom decoding for XSD schema.
func (s *FormSchema) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*s = FormSchema{
		SimpleTypes:  map[string]*FormSchemaSimpleType{},
		ComplexTypes: map[string]*FormSchemaComplexType{},
		Elements:     map[string]*FormSchemaElementDecl{},
	}
	for _, a := range start.Attr {
		if a.Name.Local == "targetNamespace" {
			s.TargetNamespace = a.Value
			break
		}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "simpleType":
				var st xsdSimpleTypeXML
				if err := d.DecodeElement(&st, &t); err != nil {
					return err
				}
				conv, err := st.toForm()
				if err != nil {
					return err
				}
				if conv.Name != "" {
					s.SimpleTypes[qnameLocal(conv.Name)] = conv
				}
			case "complexType":
				var ct xsdComplexTypeXML
				if err := d.DecodeElement(&ct, &t); err != nil {
					return err
				}
				conv, err := ct.toForm()
				if err != nil {
					return err
				}
				if conv.Name != "" {
					s.ComplexTypes[qnameLocal(conv.Name)] = conv
				}
			case "element":
				var el xsdElementXML
				if err := d.DecodeElement(&el, &t); err != nil {
					return err
				}
				conv, err := el.toForm()
				if err != nil {
					return err
				}
				if conv.Name != "" {
					s.Elements[qnameLocal(conv.Name)] = conv
				}
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

// UnmarshalXML implements custom decoding for xforms:instance.
func (fi *FormInstance) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*fi = FormInstance{Fields: map[string]FormInstanceField{}}

	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// Root element of the instance (e.g. demo:xmlData)
			fi.Root = t.Name
			if err := fi.decodeInstanceRoot(d, t); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (fi *FormInstance) decodeInstanceRoot(d *xml.Decoder, root xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// Direct child field under the instance root.
			field := FormInstanceField{
				Name: t.Name.Local,
			}
			for _, a := range t.Attr {
				if a.Name.Local == "nil" && (a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" || a.Name.Space == "") {
					// Best-effort; we also accept missing namespace for practicality.
					b, err := strconv.ParseBool(a.Value)
					if err == nil {
						field.IsNil = b
					}
				}
			}
			// Use innerxml to support complexType instance values (nested elements).
			var inner struct {
				Inner string `xml:",innerxml"`
			}
			if err := d.DecodeElement(&inner, &t); err != nil {
				return err
			}
			v := inner.Inner
			// If the value contains nested XML, normalize whitespace between tags to make it stable.
			if strings.Contains(v, "<") {
				v = strings.TrimSpace(v)
				v = intertagWhitespace.ReplaceAllString(v, "><")
			}
			field.Value = v
			fi.Fields[t.Name.Local] = field
		case xml.EndElement:
			if t.Name.Local == root.Name.Local {
				return nil
			}
		}
	}
}

// UnmarshalXML implements custom decoding for xforms:body.
func (b *FormBody) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*b = FormBody{}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			el, err := decodeFormBodyElement(d, t)
			if err != nil {
				return err
			}
			if el != nil {
				b.Elements = append(b.Elements, el)
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

const (
	xmlAttrID          = "id"
	xmlAttrRef         = "ref"
	xmlAttrIncremental = "incremental"

	xmlElLabel = "label"
	xmlElAlert = "alert"
	xmlElHelp  = "help"
	xmlElHint  = "hint"

	xmlElItem  = "item"
	xmlElValue = "value"

	xmlControlSelect  = "select"
	xmlControlSelect1 = "select1"
)

func decodeFormBodyElement(d *xml.Decoder, start xml.StartElement) (FormBodyElement, error) {
	switch start.Name.Local {
	case "group":
		g := &FormBodyGroup{}
		for _, a := range start.Attr {
			if a.Name.Local == xmlAttrID {
				v := a.Value
				g.ID = &v
				break
			}
		}
		for {
			tok, err := d.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case xmlElLabel:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					g.Label = txt
				default:
					child, err := decodeFormBodyElement(d, t)
					if err != nil {
						return nil, err
					}
					if child != nil {
						g.Elements = append(g.Elements, child)
					}
				}
			case xml.EndElement:
				if t.Name.Local == start.Name.Local {
					return g, nil
				}
			}
		}

	case "input":
		in := &FormBodyInput{}
		for _, a := range start.Attr {
			switch a.Name.Local {
			case xmlAttrID:
				v := a.Value
				in.ID = &v
			case xmlAttrRef:
				in.Ref = a.Value
			case xmlAttrIncremental:
				// Optional, ignore parse errors (leave nil).
				if bv, err := strconv.ParseBool(a.Value); err == nil {
					in.Incremental = &bv
				}
			}
		}
		for {
			tok, err := d.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case xmlElLabel:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					in.Label = txt
				case xmlElAlert:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					in.Alert = txt
				case xmlElHelp:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					in.Help = txt
				case xmlElHint:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					in.Hint = txt
				default:
					if err := d.Skip(); err != nil {
						return nil, err
					}
				}
			case xml.EndElement:
				if t.Name.Local == start.Name.Local {
					if strings.TrimSpace(in.Ref) == "" {
						// Ref-less inputs are not supported in the initial scope.
						return nil, nil
					}
					return in, nil
				}
			}
		}

	case xmlControlSelect1, xmlControlSelect:
		sel := &FormBodySelect{Multiple: start.Name.Local == xmlControlSelect}
		for _, a := range start.Attr {
			switch a.Name.Local {
			case xmlAttrID:
				v := a.Value
				sel.ID = &v
			case xmlAttrRef:
				sel.Ref = a.Value
			}
		}
		for {
			tok, err := d.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case xmlElLabel:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Label = txt
				case xmlElAlert:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Alert = txt
				case xmlElHelp:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Help = txt
				case xmlElHint:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Hint = txt
				case xmlElItem:
					item, err := decodeSelectItem(d, t)
					if err != nil {
						return nil, err
					}
					sel.Items = append(sel.Items, item)
				default:
					if err := d.Skip(); err != nil {
						return nil, err
					}
				}
			case xml.EndElement:
				if t.Name.Local == start.Name.Local {
					if strings.TrimSpace(sel.Ref) == "" {
						return nil, nil
					}
					return sel, nil
				}
			}
		}

	case "output":
		out := &FormBodyOutput{}
		for _, a := range start.Attr {
			switch a.Name.Local {
			case xmlAttrID:
				v := a.Value
				out.ID = &v
			case xmlAttrRef:
				v := a.Value
				out.Ref = &v
			}
		}
		for {
			tok, err := d.Token()
			if err != nil {
				return nil, err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case xmlElLabel:
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					out.Label = txt
				default:
					if err := d.Skip(); err != nil {
						return nil, err
					}
				}
			case xml.EndElement:
				if t.Name.Local == start.Name.Local {
					return out, nil
				}
			}
		}

	default:
		// Skip unsupported controls (e.g. submit) for now.
		if err := d.Skip(); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func decodeTextElement(d *xml.Decoder, start xml.StartElement) (string, error) {
	var cd struct {
		Text string `xml:",chardata"`
	}
	if err := d.DecodeElement(&cd, &start); err != nil {
		return "", err
	}
	return strings.TrimSpace(cd.Text), nil
}

func decodeSelectItem(d *xml.Decoder, start xml.StartElement) (FormBodySelectItem, error) {
	var item FormBodySelectItem
	for {
		tok, err := d.Token()
		if err != nil {
			return item, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case xmlElLabel:
				txt, err := decodeTextElement(d, t)
				if err != nil {
					return item, err
				}
				item.Label = txt
			case xmlElValue:
				txt, err := decodeTextElement(d, t)
				if err != nil {
					return item, err
				}
				item.Value = txt
			default:
				if err := d.Skip(); err != nil {
					return item, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return item, nil
			}
		}
	}
}

func qnameLocal(qname string) string {
	if i := strings.IndexByte(qname, ':'); i >= 0 {
		return qname[i+1:]
	}
	return qname
}

func parseOptionalBoolAttr(v string) (*bool, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func parseXFormsBoolExpr(expr string, defaultValue bool) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return defaultValue, nil
	}
	switch expr {
	case "true", "true()":
		return true, nil
	case "false", "false()":
		return false, nil
	default:
		// Keep the surface area small for v1.
		return false, fmt.Errorf("unsupported boolean expression %q (supported: true/false/true()/false())", expr)
	}
}

// --- Internal XML helpers for XSD parsing ---

type xsdFacetXML struct {
	Value string `xml:"value,attr"`
}

type xsdRestrictionXML struct {
	Base string `xml:"base,attr"`

	Pattern        *xsdFacetXML  `xml:"pattern"`
	Enumeration    []xsdFacetXML `xml:"enumeration"`
	MinLength      *xsdFacetXML  `xml:"minLength"`
	MaxLength      *xsdFacetXML  `xml:"maxLength"`
	TotalDigits    *xsdFacetXML  `xml:"totalDigits"`
	FractionDigits *xsdFacetXML  `xml:"fractionDigits"`

	MinInclusive *xsdFacetXML `xml:"minInclusive"`
	MaxInclusive *xsdFacetXML `xml:"maxInclusive"`
	MinExclusive *xsdFacetXML `xml:"minExclusive"`
	MaxExclusive *xsdFacetXML `xml:"maxExclusive"`
}

type xsdSimpleTypeXML struct {
	Name        string            `xml:"name,attr"`
	Restriction xsdRestrictionXML `xml:"restriction"`
}

func (st xsdSimpleTypeXML) toForm() (*FormSchemaSimpleType, error) {
	out := &FormSchemaSimpleType{
		Name:          st.Name,
		BaseTypeQName: st.Restriction.Base,
	}
	if st.Restriction.Pattern != nil {
		v := st.Restriction.Pattern.Value
		out.Pattern = &v
	}
	for _, e := range st.Restriction.Enumeration {
		out.Enumeration = append(out.Enumeration, e.Value)
	}
	if st.Restriction.MinLength != nil {
		i, err := strconv.Atoi(strings.TrimSpace(st.Restriction.MinLength.Value))
		if err != nil {
			return nil, err
		}
		out.MinLength = &i
	}
	if st.Restriction.MaxLength != nil {
		i, err := strconv.Atoi(strings.TrimSpace(st.Restriction.MaxLength.Value))
		if err != nil {
			return nil, err
		}
		out.MaxLength = &i
	}
	if st.Restriction.TotalDigits != nil {
		i, err := strconv.Atoi(strings.TrimSpace(st.Restriction.TotalDigits.Value))
		if err != nil {
			return nil, err
		}
		out.TotalDigits = &i
	}
	if st.Restriction.FractionDigits != nil {
		i, err := strconv.Atoi(strings.TrimSpace(st.Restriction.FractionDigits.Value))
		if err != nil {
			return nil, err
		}
		out.FractionDigits = &i
	}
	if st.Restriction.MinInclusive != nil {
		v := st.Restriction.MinInclusive.Value
		out.MinInclusive = &v
	}
	if st.Restriction.MaxInclusive != nil {
		v := st.Restriction.MaxInclusive.Value
		out.MaxInclusive = &v
	}
	if st.Restriction.MinExclusive != nil {
		v := st.Restriction.MinExclusive.Value
		out.MinExclusive = &v
	}
	if st.Restriction.MaxExclusive != nil {
		v := st.Restriction.MaxExclusive.Value
		out.MaxExclusive = &v
	}
	return out, nil
}

type xsdAllXML struct {
	Elements []xsdElementXML `xml:"element"`
}

type xsdComplexTypeXML struct {
	Name string `xml:"name,attr"`

	All *xsdAllXML `xml:"all"`
}

func (ct xsdComplexTypeXML) toForm() (*FormSchemaComplexType, error) {
	out := &FormSchemaComplexType{Name: ct.Name}
	if ct.All != nil {
		for _, el := range ct.All.Elements {
			conv, err := el.toForm()
			if err != nil {
				return nil, err
			}
			out.All = append(out.All, *conv)
		}
	}
	return out, nil
}

type xsdElementXML struct {
	Name     string `xml:"name,attr"`
	Type     string `xml:"type,attr"`
	Nillable string `xml:"nillable,attr"`

	ComplexType *xsdComplexTypeXML `xml:"complexType"`
}

func (el xsdElementXML) toForm() (*FormSchemaElementDecl, error) {
	out := &FormSchemaElementDecl{
		Name:      el.Name,
		TypeQName: el.Type,
	}
	b, err := parseOptionalBoolAttr(el.Nillable)
	if err != nil {
		return nil, err
	}
	out.Nillable = b
	if el.ComplexType != nil {
		ct, err := el.ComplexType.toForm()
		if err != nil {
			return nil, err
		}
		out.ComplexType = ct
	}
	return out, nil
}
