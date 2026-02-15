package xforms

import "github.com/shopspring/decimal"

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
	// TODO
	return nil
}
