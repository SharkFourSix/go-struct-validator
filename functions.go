package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

var validatorFunctions = map[string]ValidationFunction{
	"required":       IsRequired,
	"alphanum":       IsAlphaNumeric,
	"uuid1":          IsUuid1,
	"uuid2":          IsUuid2,
	"uuid3":          IsUuid3,
	"uuid4":          IsUuid4,
	"min":            IsMin,
	"max":            IsMax,
	"enum":           IsEnum,
	"email":          IsEmail,
	"at_least_today": IsOrBeforeToday,
	"at_most_today":  IsOrAfterToday,
	"today":          IsToday,
	"before_today":   IsBeforeToday,
	"after_today":    IsAfterToday,
}

var emailHostNameMatcher *regexp.Regexp

func init() {
	var err error
	emailHostNameMatcher, err = regexp.Compile("^[a-zA-Z0-9][a-zA-Z0-9]+$")
	if err != nil {
		panic(errors.Join(errors.New("package init: regex error"), err))
	}
}

func timeValidator(ctx *ValidationContext, comparator Comparator) bool {
	var err error
	var then time.Time
	today := time.Now()
	layout := "2006-01-02"

	if ctx.IsPointer && ctx.IsNull {
		return true
	}

	if ctx.ArgCount() == 1 {
		layout = ctx.Args[0]
	}

	if ctx.IsValueOfKind(reflect.String) {
		then, err = time.Parse(layout, ctx.GetValue().String())
		if err != nil {
			ctx.AdditionalError = err
			ctx.ErrorMessage = "invalid date format. expected format is " + layout
			return false
		}
	} else if ctx.IsValueOfType(&then) {
		then = ctx.GetValue().Interface().(time.Time)
	} else {
		panic(newValidationError("only time.Time and string and their pointer types are supported"))
	}

	match := false
	switch comparator {
	case GREATER_THAN:
		match = then.After(today)
	case GREATER_THAN_OR_EQUAL:
		match = then.After(today) || then.Equal(today)
	case LESS_THAN:
		match = then.Before(today)
	case LESS_THAN_OR_EQUAL:
		match = then.Before(today) || then.Equal(today)
	case NOT_EQUAL:
		match = !then.Equal(today)
	}

	if !match {
		ctx.ErrorMessage = fmt.Sprintf(
			"%s must be %s %s",
			then.Format(layout),
			comparator.TemporalDescription(),
			today.Format(layout),
		)
	}

	return match
}

// IsBeforeToday tests whether the given date is today or before today.
//
// If the time layout is not specified, '2006-01-02' will be used
func IsOrBeforeToday(ctx *ValidationContext) bool {
	return timeValidator(ctx, LESS_THAN_OR_EQUAL)
}

// IsOrAfterToday tests whether the given date is today or after today.
//
// If the time layout is not specified, '2006-01-02' will be used
func IsOrAfterToday(ctx *ValidationContext) bool {
	return timeValidator(ctx, GREATER_THAN_OR_EQUAL)
}

// IsBeforeToday tests whether the given date is before today.
//
// If the time layout is not specified, '2006-01-02' will be used
func IsBeforeToday(ctx *ValidationContext) bool {
	return timeValidator(ctx, LESS_THAN)
}

// IsAfterToday tests whether the given date is after today.
//
// If the time layout is not specified, '2006-01-02' will be used
func IsAfterToday(ctx *ValidationContext) bool {
	return timeValidator(ctx, GREATER_THAN)
}

// IsToday tests whether the given date is today.
//
// If the time layout is not specified, '2006-01-02' will be used
func IsToday(ctx *ValidationContext) bool {
	return timeValidator(ctx, EQUALS)
}

// IsNotToday tests whether the given date is not today.
//
// If the time layout is not specified, '2006-01-02' will be used
func IsNotToday(ctx *ValidationContext) bool {
	return timeValidator(ctx, NOT_EQUAL)
}

// IsEmail tests if the input value matches an email format.
//
// The validation rules used here do not conform to RFC and only allow only a few latin character set values.
// Therefore this function could be considered as very strict.
func IsEmail(ctx *ValidationContext) bool {
	ctx.ValueMustBeOfKind(reflect.String)

	if ctx.IsNull {
		return true
	}

	email := ctx.GetValue().String()
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	user := parts[0]
	host := parts[1]

	// last.first@sub.main.tld
	usernamePattern := "^[a-zA-Z0-9][a-zA-Z0-9_.]+$"
	m, err := regexp.MatchString(usernamePattern, user)
	if err != nil {
		panic(newValidationError("email: user part regex error", err))
	}

	if !m {
		return false
	}

	parts = strings.Split(host, ".")
	for _, domain := range parts {
		m := emailHostNameMatcher.MatchString(domain)
		if err != nil {
			panic(newValidationError("email: host part regex error", err))
		}
		if !m {
			return false
		}
	}
	return true
}

// IsEnum tests if the input value matches any of the values passed in the arguments
func IsEnum(ctx *ValidationContext) bool {
	if ctx.IsNull {
		return true
	}

	match := false

	if ctx.ArgCount() == 0 {
		panic(newValidationError("enum: At least one enum value must be specified"))
	}

	if ctx.IsValueOfKind(reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64) {
		value := strconv.FormatInt(ctx.GetValue().Int(), 10)
		match = slices.Contains(ctx.Args, value)
	} else if ctx.IsValueOfKind(reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64) {
		value := strconv.FormatUint(ctx.GetValue().Uint(), 10)
		match = slices.Contains(ctx.Args, value)
	} else if ctx.IsValueOfKind(reflect.String) {
		match = slices.Contains(ctx.Args, ctx.GetValue().String())
	} else {
		panic(newValidationError("enum: unsupported type " + ctx.valueKind.String()))
	}

	if !match {
		ctx.ErrorMessage = "invalid value specified"
		if ctx.Options.ExposeEnumValues {
			ctx.ErrorMessage += ". expected any of " + strings.Join(ctx.Args, ",")
		}
	}

	return match
}

// IsMin tests if the given input (string, integer, list) contains at least the given number of elements
func IsMin(ctx *ValidationContext) bool {
	ctx.ValueMustBeOfKind(
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.String,
	)

	if ctx.ArgCount() == 0 {
		panic(newValidationError("min: expected length or size parameter"))
	}

	if ctx.IsNull {
		return true
	}

	match := false
	propertyName := "value"
	var expected int64 = ctx.MustGetIntArg(0)

	if ctx.IsValueOfKind(reflect.String) {
		actual := len(ctx.GetValue().String())
		match = int64(actual) >= expected
		propertyName = "length"
	} else if ctx.IsValueOfKind(reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64) {
		actual := ctx.GetValue().Int()
		match = actual >= expected
	} else if ctx.IsValueOfKind(reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64) {
		expected := ctx.MustGetUintArg(0)
		actual := ctx.GetValue().Uint()
		match = actual >= expected
	}

	if !match {
		ctx.ErrorMessage = fmt.Sprintf("%s (%v) must be at least %v", propertyName, ctx.GetValue(), ctx.Args[0])
	}

	return match
}

// IsMax tests if the given input (string, integer, list) contains at least the given number of elements
func IsMax(ctx *ValidationContext) bool {
	ctx.ValueMustBeOfKind(
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.String,
	)

	if ctx.ArgCount() == 0 {
		panic(newValidationError("max: expected length or size parameter"))
	}

	if ctx.IsNull {
		return true
	}

	match := false
	propertyName := "value"
	var expected int64 = ctx.MustGetIntArg(0)

	if ctx.IsValueOfKind(reflect.String) {
		actual := len(ctx.GetValue().String())
		match = int64(actual) <= expected
		propertyName = "length"
	} else if ctx.IsValueOfKind(reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64) {
		actual := ctx.GetValue().Int()
		match = actual <= expected
	} else if ctx.IsValueOfKind(reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64) {
		expected := ctx.MustGetUintArg(0)
		actual := ctx.GetValue().Uint()
		match = actual <= expected
	}

	if !match {
		ctx.ErrorMessage = fmt.Sprintf("%s (%v) must not exceed %v", propertyName, ctx.GetValue(), ctx.Args[0])
	}

	return match
}

// IsAlphaNumeric verifies that the given string is alphanumeric
func IsAlphaNumeric(ctx *ValidationContext) bool {
	ctx.ValueMustBeOfKind(reflect.String)

	if ctx.IsNull {
		return true
	}

	alphaNumPattern := "^[a-z0-9]+$"
	m, err := regexp.MatchString(alphaNumPattern, ctx.GetValue().String())
	if err != nil {
		panic(newValidationError("regex error when validating input", err))
	}
	if !m {
		ctx.ErrorMessage = "must be alphanumeric"
	}
	return m
}

// IsRequired check if the required field has values.
//
// For literal values, the function always returns true because the values are present and can subsequnetly
// be validated appropriately.
//
// For pointer types, the function will return false if the pointer is null or true if the pointer is not null
func IsRequired(ctx *ValidationContext) bool {
	if ctx.IsNull {
		ctx.ErrorMessage = "this field is requiredd"
		return false
	}
	return true
}

func uuidFn(ctx *ValidationContext, version int) bool {
	ctx.ValueMustBeOfKind(reflect.String)

	if ctx.IsNull {
		return false
	}

	id, err := uuid.Parse(ctx.GetValue().String())
	if err != nil {
		ctx.ErrorMessage = "invalid uuid format"
		return false
	}
	match := id.Version() == uuid.Version(version)
	if !match {
		ctx.ErrorMessage = fmt.Sprintf("expectedd UUIDv%d but found UUIDv%d", version, int(id.Version()))
	}
	return match
}

func IsUuid1(ctx *ValidationContext) bool {
	return uuidFn(ctx, 1)
}

func IsUuid2(ctx *ValidationContext) bool {
	return uuidFn(ctx, 2)
}

func IsUuid3(ctx *ValidationContext) bool {
	return uuidFn(ctx, 3)
}

func IsUuid4(ctx *ValidationContext) bool {
	return uuidFn(ctx, 4)
}

var filterFunctions = map[string]FilterFunction{
	"trim": Trim,
}

func Trim(ctx *ValidationContext) reflect.Value {
	ctx.ValueMustBeOfKind(reflect.String)

	if ctx.IsPointer && !ctx.IsNull {
		value := ctx.GetValue().String()
		trimmed := strings.TrimSpace(value)
		return reflect.ValueOf(&trimmed)
	} else {
		return reflect.ValueOf(strings.TrimSpace(ctx.GetValue().String()))
	}
}
