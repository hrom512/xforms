package xforms

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type Form struct {
	Schema      FormSchema
	Instance    FormInstance
	Binds       []*FormBind
	Submissions []*FormSubmission
	Body        FormBody
}

type FormSchema struct {
	TargetNamespace string

	// All maps are keyed by local name (QName prefix is ignored).
	SimpleTypes  map[string]*FormSchemaSimpleType
	ComplexTypes map[string]*FormSchemaComplexType
	Elements     map[string]*FormSchemaElementDecl
}

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

type FormSchemaComplexType struct {
	Name string

	// For initial scope, we support xsd:all.
	All []FormSchemaElementDecl
}

type FormInstance struct {
	Root   xml.Name
	Fields map[string]FormInstanceField // keyed by local name
}

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

type FormSubmission struct {
	ID     string
	Action string
	Method string
}

type FormBody struct {
	Elements []FormBodyElement
}

type FormSchemaElementDecl struct {
	Name      string
	TypeQName string // may be empty if ComplexType is provided inline
	Nillable  *bool

	ComplexType *FormSchemaComplexType
}

type FormInstanceField struct {
	Name      string
	Namespace string
	Value     string
	IsNil     bool
}

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

type FormBodyGroup struct {
	ID       *string
	Label    string
	Elements []FormBodyElement
}

func (*FormBodyGroup) formBodyElement() {}

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

type FormBodySelectItem struct {
	Label string
	Value string
}

func (*FormBodySelect) formBodyElement() {}

type FormBodyOutput struct {
	ID  *string
	Ref *string

	// For simplicity, we treat output as a message. If Ref is set, message
	// can be taken from instance during conversion.
	Label string
}

func (*FormBodyOutput) formBodyElement() {}

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
				Name:      t.Name.Local,
				Namespace: t.Name.Space,
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
			var cd struct {
				Text string `xml:",chardata"`
			}
			if err := d.DecodeElement(&cd, &t); err != nil {
				return err
			}
			field.Value = cd.Text
			fi.Fields[t.Name.Local] = field
		case xml.EndElement:
			if t.Name.Local == root.Name.Local {
				return nil
			}
		}
	}
}

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

func decodeFormBodyElement(d *xml.Decoder, start xml.StartElement) (FormBodyElement, error) {
	switch start.Name.Local {
	case "group":
		g := &FormBodyGroup{}
		for _, a := range start.Attr {
			if a.Name.Local == "id" {
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
				case "label":
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
			case "id":
				v := a.Value
				in.ID = &v
			case "ref":
				in.Ref = a.Value
			case "incremental":
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
				case "label":
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					in.Label = txt
				case "alert":
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					in.Alert = txt
				case "help":
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					in.Help = txt
				case "hint":
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

	case "select1", "select":
		sel := &FormBodySelect{Multiple: start.Name.Local == "select"}
		for _, a := range start.Attr {
			switch a.Name.Local {
			case "id":
				v := a.Value
				sel.ID = &v
			case "ref":
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
				case "label":
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Label = txt
				case "alert":
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Alert = txt
				case "help":
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Help = txt
				case "hint":
					txt, err := decodeTextElement(d, t)
					if err != nil {
						return nil, err
					}
					sel.Hint = txt
				case "item":
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
			case "id":
				v := a.Value
				out.ID = &v
			case "ref":
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
				case "label":
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
			case "label":
				txt, err := decodeTextElement(d, t)
				if err != nil {
					return item, err
				}
				item.Label = txt
			case "value":
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
	if b, err := parseOptionalBoolAttr(el.Nillable); err != nil {
		return nil, err
	} else {
		out.Nillable = b
	}
	if el.ComplexType != nil {
		ct, err := el.ComplexType.toForm()
		if err != nil {
			return nil, err
		}
		out.ComplexType = ct
	}
	return out, nil
}
