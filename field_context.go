package validator

import (
	"reflect"
	"strings"
)

type fieldContext struct {
	required             bool
	filters              []*fieldValueFilter
	validators           []*fieldValueValidator
	fieldName            string
	fieldKind            reflect.Kind
	fieldLabel           string
	fieldMessageTemplate string
	hasLabel             bool
	hasMessagTemplate    bool
}

func (fc fieldContext) hasFilters() bool {
	return len(fc.filters) > 0
}

func (fc *fieldContext) apply(val interface{}, opts *ValidationOptions) {

}

func mustParse(field reflect.StructField, opts *ValidationOptions) (ctx *fieldContext) {
	// 1. Get validator tags
	// 2. Get filter tags
	// 3. Get message template tag
	// 4. Get label template tag
	// 5.
	filterTagValues, filters := field.Tag.Lookup(opts.FilterTagName)
	validatorTagValues, validators := field.Tag.Lookup(opts.ValidatorTagName)
	if !filters && !validators {
		return
	}

	if validators {
		// split by "|"
		// `validate:"required|uuidv4"`
		parts := strings.Split(validatorTagValues, "|")
		if len(parts) > 0 {

		}
	}
}
