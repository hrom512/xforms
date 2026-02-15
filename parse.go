package xforms

import (
	"encoding/xml"
	"fmt"
	"io"
)

func Parse(input io.Reader) (*Form, error) {
	dec := xml.NewDecoder(input)

	var doc xformsHTMLXML
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}

	f := &Form{
		Schema:   doc.Head.Model.Schema,
		Instance: doc.Head.Model.Instance,
		Body:     doc.Body,
	}

	// Ensure maps are non-nil for caller convenience.
	if f.Schema.SimpleTypes == nil {
		f.Schema.SimpleTypes = map[string]*FormSchemaSimpleType{}
	}
	if f.Schema.ComplexTypes == nil {
		f.Schema.ComplexTypes = map[string]*FormSchemaComplexType{}
	}
	if f.Schema.Elements == nil {
		f.Schema.Elements = map[string]*FormSchemaElementDecl{}
	}
	if f.Instance.Fields == nil {
		f.Instance.Fields = map[string]FormInstanceField{}
	}

	for _, b := range doc.Head.Model.Binds {
		if b.Nodeset == "" {
			return nil, fmt.Errorf("xforms:bind missing nodeset")
		}

		required, err := parseXFormsBoolExpr(b.Required, false)
		if err != nil {
			return nil, fmt.Errorf("xforms:bind nodeset=%q required=%q: %w", b.Nodeset, b.Required, err)
		}
		readonly, err := parseXFormsBoolExpr(b.Readonly, false)
		if err != nil {
			return nil, fmt.Errorf("xforms:bind nodeset=%q readonly=%q: %w", b.Nodeset, b.Readonly, err)
		}
		relevant, err := parseXFormsBoolExpr(b.Relevant, true)
		if err != nil {
			return nil, fmt.Errorf("xforms:bind nodeset=%q relevant=%q: %w", b.Nodeset, b.Relevant, err)
		}

		var idPtr *string
		if b.ID != "" {
			v := b.ID
			idPtr = &v
		}
		var typePtr *string
		if b.Type != "" {
			v := b.Type
			typePtr = &v
		}
		var extPtr *string
		if b.ExtType != "" {
			v := b.ExtType
			extPtr = &v
		}

		f.Binds = append(f.Binds, &FormBind{
			ID:          idPtr,
			Nodeset:     b.Nodeset,
			TypeQName:   typePtr,
			ExtType:     extPtr,
			Required:    required,
			Readonly:    readonly,
			Relevant:    relevant,
			RawRequired: b.Required,
			RawReadonly: b.Readonly,
			RawRelevant: b.Relevant,
		})
	}

	for _, s := range doc.Head.Model.Submissions {
		if s.ID == "" {
			return nil, fmt.Errorf("xforms:submission missing id")
		}
		f.Submissions = append(f.Submissions, &FormSubmission{
			ID:     s.ID,
			Action: s.Action,
			Method: s.Method,
		})
	}

	return f, nil
}

// --- Internal XML model for Parse() ---

type xformsHTMLXML struct {
	XMLName xml.Name      `xml:"html"`
	Head    xformsHeadXML `xml:"head"`
	Body    FormBody      `xml:"body"`
}

type xformsHeadXML struct {
	Model xformsModelXML `xml:"model"`
}

type xformsModelXML struct {
	Schema      FormSchema            `xml:"schema"`
	Instance    FormInstance          `xml:"instance"`
	Binds       []xformsBindXML       `xml:"bind"`
	Submissions []xformsSubmissionXML `xml:"submission"`
}

type xformsBindXML struct {
	ID       string `xml:"id,attr"`
	Nodeset  string `xml:"nodeset,attr"`
	Type     string `xml:"type,attr"`
	ExtType  string `xml:"exttype,attr"`
	Required string `xml:"required,attr"`
	Readonly string `xml:"readonly,attr"`
	Relevant string `xml:"relevant,attr"`
}

type xformsSubmissionXML struct {
	ID     string `xml:"id,attr"`
	Action string `xml:"action,attr"`
	Method string `xml:"method,attr"`
}
