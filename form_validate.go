package xforms

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// ValidationError represents form validation failures grouped by field.
type ValidationError struct {
	FieldErrors map[string][]string
}

// Error returns a compact, deterministic description of validation failures.
func (e *ValidationError) Error() string {
	if e == nil || len(e.FieldErrors) == 0 {
		return "validation failed"
	}
	// Deterministic, compact message (field count + first field).
	firstField := ""
	for k := range e.FieldErrors {
		if firstField == "" || k < firstField {
			firstField = k
		}
	}
	return fmt.Sprintf("validation failed: %d field(s) invalid (e.g. %s)", len(e.FieldErrors), firstField)
}

func (e *ValidationError) add(field, msg string) {
	if e.FieldErrors == nil {
		e.FieldErrors = map[string][]string{}
	}
	e.FieldErrors[field] = append(e.FieldErrors[field], msg)
}

// Validate checks the correctness of form field values in FormInstance
func (f *Form) Validate() error {
	var verr ValidationError

	for _, b := range f.Binds {
		if b == nil {
			continue
		}
		if !b.Relevant {
			continue
		}

		field := fieldNameFromRef(b.Nodeset)
		if field == "" {
			continue
		}

		raw, present := f.instanceFieldValue(field)
		val := strings.TrimSpace(raw)

		decl := f.schemaDeclForInstanceField(field)
		if decl != nil && decl.Nillable != nil && !*decl.Nillable && present {
			if inst, ok := f.Instance.Fields[field]; ok && inst.IsNil {
				verr.add(field, "value is nil but field is not nillable")
				// Don't return early; collect all errors.
			}
		}

		if b.Required && val == "" {
			verr.add(field, "field is required")
			// Still continue to collect other errors.
			continue
		}

		// If empty and not required, skip type/facet checks.
		if val == "" {
			continue
		}

		st, _ := f.resolveSchemaType(b, field)
		baseQName := "xsd:string"
		if st != nil && strings.TrimSpace(st.BaseTypeQName) != "" {
			baseQName = st.BaseTypeQName
		} else if b.TypeQName != nil && strings.TrimSpace(*b.TypeQName) != "" {
			// For builtins, resolveSchemaType() returns a synthetic type; but if it didn't,
			// we still want to attempt base-type parsing for xsd:*.
			baseQName = *b.TypeQName
		}
		switch qnameLocal(baseQName) {
		case xsdLocalBoolean:
			if _, err := strconv.ParseBool(val); err != nil {
				verr.add(field, "must be a boolean")
			}
		case xsdLocalDecimal:
			if _, err := decimal.NewFromString(val); err != nil {
				verr.add(field, "must be a decimal number")
			}
		default:
			// Treat all other bases as string for v1.
		}

		// Facets (only for simple types).
		if st == nil {
			continue
		}

		if st.MinLength != nil {
			if runeLen(val) < *st.MinLength {
				verr.add(field, fmt.Sprintf("length must be >= %d", *st.MinLength))
			}
		}
		if st.MaxLength != nil {
			if runeLen(val) > *st.MaxLength {
				verr.add(field, fmt.Sprintf("length must be <= %d", *st.MaxLength))
			}
		}
		if st.Pattern != nil && strings.TrimSpace(*st.Pattern) != "" {
			re, err := regexp.Compile(*st.Pattern)
			if err != nil {
				verr.add(field, "invalid schema pattern")
			} else if !re.MatchString(val) {
				verr.add(field, "does not match required pattern")
			}
		}
		if len(st.Enumeration) > 0 {
			ok := false
			for _, ev := range st.Enumeration {
				if val == ev {
					ok = true
					break
				}
			}
			if !ok {
				verr.add(field, "value is not in enumeration")
			}
		}

		// Numeric bounds (decimal only, for now).
		if qnameLocal(baseQName) == xsdLocalDecimal {
			cur, err := decimal.NewFromString(val)
			if err == nil {
				if st.MinInclusive != nil {
					if minVal, err := decimal.NewFromString(strings.TrimSpace(*st.MinInclusive)); err == nil {
						if cur.Cmp(minVal) < 0 {
							verr.add(field, fmt.Sprintf("must be >= %s", minVal.String()))
						}
					}
				}
				if st.MaxInclusive != nil {
					if maxVal, err := decimal.NewFromString(strings.TrimSpace(*st.MaxInclusive)); err == nil {
						if cur.Cmp(maxVal) > 0 {
							verr.add(field, fmt.Sprintf("must be <= %s", maxVal.String()))
						}
					}
				}
				if st.MinExclusive != nil {
					if minVal, err := decimal.NewFromString(strings.TrimSpace(*st.MinExclusive)); err == nil {
						if cur.Cmp(minVal) <= 0 {
							verr.add(field, fmt.Sprintf("must be > %s", minVal.String()))
						}
					}
				}
				if st.MaxExclusive != nil {
					if maxVal, err := decimal.NewFromString(strings.TrimSpace(*st.MaxExclusive)); err == nil {
						if cur.Cmp(maxVal) >= 0 {
							verr.add(field, fmt.Sprintf("must be < %s", maxVal.String()))
						}
					}
				}
			}
		}
	}

	if len(verr.FieldErrors) > 0 {
		return &verr
	}
	return nil
}

func (f *Form) schemaDeclForInstanceField(fieldName string) *FormSchemaElementDecl {
	// Prefer declaration under the instance root element.
	rootLocal := qnameLocal(f.Instance.Root.Local)
	if rootLocal != "" && f.Schema.Elements != nil {
		if rootDecl, ok := f.Schema.Elements[rootLocal]; ok && rootDecl != nil && rootDecl.ComplexType != nil {
			for i := range rootDecl.ComplexType.All {
				if rootDecl.ComplexType.All[i].Name == fieldName {
					return &rootDecl.ComplexType.All[i]
				}
			}
		}
	}
	// Fallback to top-level element declarations.
	if f.Schema.Elements != nil {
		if decl, ok := f.Schema.Elements[fieldName]; ok {
			return decl
		}
	}
	return nil
}

func runeLen(s string) int {
	// XSD length is in characters, not bytes.
	return len([]rune(s))
}
