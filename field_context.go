package validator

import (
	"reflect"
	"strings"

	"golang.org/x/exp/slices"
)

type fieldContext struct {
	filters              []*fieldValueFilter
	validators           []*fieldValueValidator
	fieldName            string
	fieldKind            reflect.Kind
	fieldLabel           string
	fieldMessageTemplate string
	hasLabel             bool
	hasMessagTemplate    bool
}

func (fc *fieldContext) apply(structValue reflect.Value, opts *ValidationOptions) []FieldError {
	field := structValue.FieldByName(fc.fieldName)
	value := field.Addr().Elem()
	//value := field.Addr().Interface()
	ispointer := value.Kind() == reflect.Ptr
	var isnull bool = false

	var errorList []FieldError

	if ispointer {
		isnull = value.IsNil()
	}

	for _, validator := range fc.validators {
		ctx := ValidationContext{
			IsPointer: ispointer,
			IsNull:    isnull,
			Options:   opts,
			Args:      validator.args,
			value:     value,
			valueKind: fc.fieldKind,
		}

		if !validator.fn(&ctx) {
			fe := FieldError{Field: fc.fieldLabel}
			if fc.hasMessagTemplate {
				fe.Message = fc.fieldMessageTemplate
			} else {
				if len(ctx.ErrorMessage) > 0 {
					fe.Message = ctx.ErrorMessage
				} else {
					fe.Message = fc.fieldLabel + ": field validation failed"
					if opts.ExposeValidatorNames {
						fe.Message += " using function " + validator.name
					}
				}
			}
			errorList = append(errorList, fe)
			if opts.StopOnFirstError {
				return errorList
			}
		}
	}

	for _, filter := range fc.filters {
		ctx := ValidationContext{
			IsPointer: ispointer,
			IsNull:    isnull,
			Options:   opts,
			Args:      filter.args,
			value:     value,
			valueKind: fc.fieldKind,
		}
		newValue := filter.fn(&ctx)
		value.Set(newValue)
	}

	return errorList
}

func mustParseField(field reflect.StructField, opts *ValidationOptions) (ctx *fieldContext) {
	// 1. Get validator tags
	// 2. Get filter tags
	// 3. Get message template tag
	// 4. Get label template tag
	// 5.

	if field.Name[0] >= 'a' || field.Name[0] <= 'z' {
		if !opts.PrivateFields {
			return
		}
	}

	filterTagValues, filters := field.Tag.Lookup(opts.FilterTagName)
	validatorTagValues, validators := field.Tag.Lookup(opts.ValidatorTagName)
	messageTemplate, hasMsgTemplate := field.Tag.Lookup(opts.MessageTagName)
	label, hasLabel := field.Tag.Lookup(opts.LabelTagName)
	if !filters && !validators {
		return
	}

	fc := fieldContext{
		validators:        make([]*fieldValueValidator, 0),
		filters:           make([]*fieldValueFilter, 0),
		hasLabel:          hasLabel,
		hasMessagTemplate: hasMsgTemplate,
		fieldKind:         field.Type.Kind(),
	}

	if len(field.PkgPath) == 0 {
		fc.fieldName = field.Name
	} else {
		fc.fieldName = field.PkgPath + "." + field.Name
	}

	// resolve actual contained type
	kinds := []reflect.Kind{reflect.Array, reflect.Map, reflect.Slice, reflect.Pointer}

	if slices.Contains(kinds, field.Type.Kind()) {
		fc.fieldKind = field.Type.Elem().Kind()
	}

	if hasLabel {
		fc.fieldLabel = label
	} else {
		if opts.UseFullyQualifiedFieldNames {
			fc.fieldLabel = fc.fieldName
		} else {
			fc.fieldLabel = field.Name
		}
	}

	if hasMsgTemplate {
		fc.fieldMessageTemplate = messageTemplate
	}

	if validators {
		// split by "|"
		// `validate:"required|uuidv4|v1(arg1,arg2)"`
		parts := strings.Split(validatorTagValues, "|")
		if len(parts) > 0 {
			for _, function := range parts {
				// extract
				name, args := extractFunctionInformation(function)

				v, ok := validatorFunctions[name]
				if !ok {
					panic(newValidationError("validator `" + name + "` referenced by field " + field.Name + " not found"))
				}

				fc.validators = append(fc.validators, &fieldValueValidator{name: name, fn: v, args: args})
			}
		}
	}

	if filters {
		parts := strings.Split(filterTagValues, "|")
		if len(parts) > 0 {
			for _, function := range parts {
				// extract
				name, args := extractFunctionInformation(function)

				v, ok := filterFunctions[name]
				if !ok {
					panic(newValidationError("filter " + name + " referenced by field " + field.Name + " not found"))
				}

				fc.filters = append(fc.filters, &fieldValueFilter{name: name, fn: v, args: args})
			}
		}
	}

	ctx = &fc
	return
}

func extractFunctionInformation(funcDefinition string) (name string, args []string) {
	if strings.HasSuffix(funcDefinition, "()") {
		name = strings.Trim(funcDefinition, "()")
		args = []string{}
	} else if strings.ContainsAny(funcDefinition, "()") {
		openParenthesisPosition := strings.Index(funcDefinition, "(")
		closeParenthesisPosition := strings.LastIndex(funcDefinition, ")")
		name = funcDefinition[0:openParenthesisPosition]
		args = strings.Split(funcDefinition[openParenthesisPosition+1:closeParenthesisPosition], ",")
	} else {
		name = funcDefinition
		args = []string{}
	}
	return
}
