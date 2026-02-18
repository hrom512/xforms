# XForms

Go library for working with **XForms** (a subset of the spec: `https://www.w3.org/TR/xforms20/`):

- parse XForms XML into a `Form` structure
- build a convenient field representation via `Form.Elements()`
- fill/update instance values via `Form.Fill()`
- validate instance values via `Form.Validate()` and `Form.ValidateAndFill()`

## Supported features (v1)

- **XForms**: `model/schema/instance/bind/submission` + `body` with `group`, `input`, `select1`/`select`, `output`
- **Bindings**: `ref`/`nodeset` are treated as a **direct child element name** under the instance root (full XPath is not supported)
- **Bind expressions**: only `true/false/true()/false()` for `relevant/required/readonly`
- **XSD facets (partial)**: `pattern`, `enumeration`, `minLength`, `maxLength`, plus numeric bounds for `xsd:decimal` (`min/max Inclusive/Exclusive` when present)

## Quick example

Below is a minimal (simplified) form example. It includes:

- `xsd:schema` with `simpleType` and `pattern`
- `xforms:instance` with values
- `xforms:bind` for required/readonly/relevant and type
- `xforms:input` and `xforms:select1`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xhtml:html
  xmlns:xhtml="http://www.w3.org/1999/xhtml"
  xmlns:xforms="http://www.w3.org/2002/xforms"
  xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xhtml:head>
    <xforms:model>
      <xsd:schema targetNamespace="urn:demo">
        <xsd:simpleType name="CODE">
          <xsd:restriction base="xsd:string">
            <xsd:pattern value="^[A-Z]{2}\\d{4}$"/>
          </xsd:restriction>
        </xsd:simpleType>

        <xsd:element name="data">
          <xsd:complexType>
            <xsd:all>
              <xsd:element name="code" nillable="false" type="CODE"/>
              <xsd:element name="plan" nillable="false" type="xsd:string"/>
              <xsd:element name="message" nillable="false" type="xsd:string"/>
            </xsd:all>
          </xsd:complexType>
        </xsd:element>
      </xsd:schema>

      <xforms:instance>
        <data>
          <code>AB1234</code>
          <plan>basic</plan>
          <message>Hello</message>
        </data>
      </xforms:instance>

      <xforms:bind nodeset="code" required="true" relevant="true()" readonly="false" type="CODE"/>
      <xforms:bind nodeset="plan" required="true" relevant="true()" readonly="false" type="xsd:string"/>
      <xforms:bind nodeset="message" required="false" relevant="true()" readonly="true" type="xsd:string"/>
    </xforms:model>
  </xhtml:head>

  <xhtml:body>
    <xforms:group>
      <xforms:label>Demo</xforms:label>

      <xforms:input ref="code">
        <xforms:label>Code</xforms:label>
        <xforms:alert>Code must match pattern</xforms:alert>
      </xforms:input>

      <xforms:select1 ref="plan">
        <xforms:label>Plan</xforms:label>
        <xforms:item><xforms:label>Basic</xforms:label><xforms:value>basic</xforms:value></xforms:item>
        <xforms:item><xforms:label>Pro</xforms:label><xforms:value>pro</xforms:value></xforms:item>
      </xforms:select1>

      <xforms:output ref="message">
        <xforms:label>Message</xforms:label>
      </xforms:output>
    </xforms:group>
  </xhtml:body>
</xhtml:html>
```

### 1) Parse + Elements

```go
package main

import (
	"fmt"
	"strings"

	"github.com/hrom512/xforms"
)

const formXML = `...XML from above...`

func main() {
	f, err := xforms.Parse(strings.NewReader(formXML))
	if err != nil {
		panic(err)
	}

	for _, el := range f.Elements() {
		switch v := el.(type) {
		case *xforms.FieldGroup:
			fmt.Println("Group:", v.Label)
			for _, child := range v.Elements {
				fmt.Printf("  - %T\n", child)
			}
		case *xforms.TextInput:
			fmt.Println("Text:", v.Name, "label=", v.Label, "required=", v.Required, "readonly=", v.Readonly, "value=", deref(v.Value))
		case *xforms.SelectInput:
			fmt.Println("Select:", v.Name, "label=", v.Label, "value=", selectValue(v.Value))
		case *xforms.TextMessage:
			fmt.Println("Output:", v.Message)
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func selectValue(o *xforms.SelectOption) string {
	if o == nil {
		return ""
	}
	return o.Value
}
```

### 2) Validate

```go
if err := f.Validate(); err != nil {
	if ve, ok := err.(*xforms.ValidationError); ok {
		for field, msgs := range ve.FieldErrors {
			fmt.Println(field, msgs)
		}
	} else {
		fmt.Println("validate error:", err)
	}
}
```

### 3) Fill / ValidateAndFill

`Fill` updates only the provided values (all other instance fields stay unchanged).

```go
newCode := "ZZ9999"
newPlan := "pro"

err := f.Fill([]xforms.FormElement{
	&xforms.TextInput{Name: "code", Value: &newCode},
	&xforms.SelectInput{Name: "plan", Value: &xforms.SelectOption{Value: newPlan}},
})
if err != nil {
	panic(err)
}
```

`ValidateAndFill` is atomic: it validates “as if applied” first, and only commits to `Instance` on success.

```go
badCode := "not-a-code"
if err := f.ValidateAndFill([]xforms.FormElement{
	&xforms.TextInput{Name: "code", Value: &badCode},
}); err != nil {
	// instance stays unchanged
	fmt.Println("validation failed:", err)
}
```

## Limitations

- There is no full XPath/XForms expression evaluator (only `true/false/true()/false()` are supported).
- `ref/nodeset` must point to a direct child field name under the instance root.
- Only a subset of controls and XSD facets is supported (see above).
