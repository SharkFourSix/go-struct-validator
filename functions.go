package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

var validatorFunctions = map[string]ValidationFunction{
	"required": IsRequired,
	"alphanum": IsAlphaNumeric,
	"uuid1":    IsUuid1,
	"uuid2":    IsUuid2,
	"uuid3":    IsUuid3,
	"uuid4":    IsUuid4,
	"min":      IsMin,
	"max":      IsMax,
	"enum":     IsEnum,
	"email":    IsEmail,
}

var emailHostNameMatcher *regexp.Regexp

func init() {
	var err error
	emailHostNameMatcher, err = regexp.Compile("^[a-zA-Z0-9][a-zA-Z0-9]+$")
	if err != nil {
		panic(errors.Join(errors.New("package init: regex error"), err))
	}
}

// IsEmail IsEmail tests if the input value matches an email format.
//
// The validation rules used here do not conform to RFC and only allow only a few latin character set values.
// Therefore this function could be considered as very strict.
func IsEmail(ctx *ValidationContext) bool {
	ctx.ValueMustBeOfKind(reflect.String)

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

// IsEnum IsEnum tests if the input value matches any of the values passed in the arguments
func IsEnum(ctx *ValidationContext) bool {
	if ctx.ArgCount() == 0 {
		panic(newValidationError("enum: At least one enum value must be specified"))
	}

	if ctx.IsValueOfKind(reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64) {
		value := strconv.FormatInt(ctx.GetValue().Int(), 10)
		return slices.Contains(ctx.Args, value)
	}

	if ctx.IsValueOfKind(reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64) {
		value := strconv.FormatUint(ctx.GetValue().Uint(), 10)
		return slices.Contains(ctx.Args, value)
	}

	panic(newValidationError("enum: unsupported type " + ctx.valueKind.String()))
}

// IsMin IsMin tests if the given input (string, integer, list) contains at least the given number of elements
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

	match := false
	propertyName := "value"
	var expected int64 = ctx.MustGetIntArg(0)

	if ctx.IsValueOfKind(reflect.String) {
		actual := len(ctx.GetValue().String())
		match := int64(actual) >= expected
		propertyName = "length"
		return match
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

// IsMax IsMax tests if the given input (string, integer, list) contains at least the given number of elements
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

// IsAlphaNumeric IsAlphaNumeric verifies that the given string is alphanumeric
func IsAlphaNumeric(ctx *ValidationContext) bool {
	ctx.ValueMustBeOfKind(reflect.String)

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

// IsRequired IsRequired check if the required field has values.
//
// For literal values, the function always returns true because the values are present.
//
// For pointer types, the function will return false if the pointer is nil or true if the pointer is non nil
func IsRequired(ctx *ValidationContext) bool {
	if ctx.IsPointer {
		if ctx.IsNull {
			ctx.ErrorMessage = "this field is requiredd"
		}
		return !ctx.IsNull
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
