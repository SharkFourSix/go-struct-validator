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
	triggers             []string
	flags                []ValidationFlag
	zeroValue            reflect.Value
}

func (fc *fieldContext) isFlagSet(flag ValidationFlag) bool {
	return slices.Contains(fc.flags, flag)
}

func (fc *fieldContext) isZero(v reflect.Value) bool {
	return fc.zeroValue.Equal(v)
}

func (fc *fieldContext) activate(trigger string) bool {
	if !slices.Contains(fc.triggers, trigger) {
		return slices.Contains(fc.triggers, "all")
	}
	return true
}

func (fc *fieldContext) apply(structValue reflect.Value, opts *ValidationOptions) []FieldError {
	field := structValue.FieldByName(fc.fieldName)
	value := field.Addr().Elem()

	ispointer := value.Kind() == reflect.Ptr
	var isnull bool = false

	var errorList []FieldError

	if ispointer {
		isnull = value.IsNil()
	}

	if fc.isFlagSet(AllowZero) {
		if ispointer {
			if value.IsZero() || fc.isZero(value.Elem()) {
				return nil
			}
		} else if fc.isZero(value) {
			return nil
		}
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
	// skip over unexported fields
	if field.Name[0] >= 'a' && field.Name[0] <= 'z' {
		return
	}

	flagTagValues, hasFlags := field.Tag.Lookup(opts.FlagTagName)
	filterTagValues, filters := field.Tag.Lookup(opts.FilterTagName)
	triggerTagValues, hasTriggers := field.Tag.Lookup(opts.TriggerTagName)
	validatorTagValues, validators := field.Tag.Lookup(opts.ValidatorTagName)
	messageTemplate, hasMsgTemplate := field.Tag.Lookup(opts.MessageTagName)
	label, hasLabel := field.Tag.Lookup(opts.LabelTagName)

	if !filters && !validators {
		return
	}

	var zeroValue reflect.Value

	if field.Type.Kind() == reflect.Ptr {
		zeroValue = reflect.Zero(field.Type.Elem())
	} else {
		zeroValue = reflect.Zero(field.Type)
	}

	fc := fieldContext{
		validators:        make([]*fieldValueValidator, 0),
		filters:           make([]*fieldValueFilter, 0),
		hasLabel:          hasLabel,
		hasMessagTemplate: hasMsgTemplate,
		fieldKind:         field.Type.Kind(),
		zeroValue:         zeroValue,
	}

	if hasTriggers {
		triggers := strings.Split(triggerTagValues, ",")
		fc.triggers = append(fc.triggers, triggers...)
	} else {
		fc.triggers = append(fc.triggers, "all")
	}

	fc.fieldName = field.Name

	// resolve actual contained type
	kinds := []reflect.Kind{reflect.Array, reflect.Map, reflect.Slice, reflect.Pointer}

	if slices.Contains(kinds, field.Type.Kind()) {
		fc.fieldKind = field.Type.Elem().Kind()
	}

	if hasLabel {
		fc.fieldLabel = label
	} else {
		fc.fieldLabel = field.Name
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

	if hasFlags {
		parts := strings.Split(flagTagValues, "|")
		if len(parts) > 0 {
			for _, flag := range parts {
				fc.flags = append(fc.flags, ValidationFlag(strings.TrimSpace(flag)))
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
