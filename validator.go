package validator

import (
	"errors"
	"reflect"
)

type ValidationOptions struct {
	//
	// PrivateFields Whether to validate private fields or not
	//
	PrivateFields bool
	//
	// UseFullyQualifiedFieldNames Whether to use nested fields fully qualified names
	//
	//   type Foo struct {
	//	   FieldA int
	//   }
	//
	//   type Bar struct {
	//     Foo
	//     FieldB int
	//   }
	//
	//   foo := Foo{FieldA:3}
	//   bar := Bar{Foo: foo, FieldB: 4}
	//
	// In the case above, by default bar.Foo.FieldA will resolve to bar.FieldA or simply 'FieldA'.
	//
	// If this option is set to true then the field will be reported as bar.Foo.FieldA.
	//
	// default: false
	UseFullyQualifiedFieldNames bool

	// FilterTagName FilterTagName specifies the tag to use when looking up filter functions
	//
	// default: 'filter'
	FilterTagName string

	// ValidatorTagName ValidatorTagName specifies the tag to use when looking up validation functions
	//
	// default: 'validator'
	ValidatorTagName string

	// StringAutoTrim StringAutoTrim specifies whether to automatically trim all strings
	//
	// default: false
	StringAutoTrim bool
}

var cache *fieldCache
var globalOptions ValidationOptions

func init() {
	// default parameters
	globalOptions = ValidationOptions{
		PrivateFields:    true,
		FilterTagName:    "filter",
		ValidatorTagName: "validator",
		StringAutoTrim:   false,
	}
	cache = &fieldCache{}
}

type fieldValueValidator struct {
	fn   ValidationFunction
	name string
}

func (f fieldValueValidator) Apply(ctx *ValidationContext) interface{} {
	return f.fn(ctx)
}

type ValidationError struct {
	DelegatedError error
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e FieldError) Error() string {
	return e.Field + ": " + e.Message
}

func (e ValidationError) Error() string {
	return e.DelegatedError.Error()
}

type ValidationResult struct {
	valid bool
	// Error the top level error summarizing what the hell happened. May not necessarily come from validating the passed struct
	//
	Error ValidationError
	//
	// FieldErrors FieldErrors struct field validation errors
	FieldErrors []FieldError
}

func (r ValidationResult) IsValid() bool {
	return r.valid
}

// ValidationFunction ValidationFunction is used to validate input.
// Validator functions return a boolean indicating whether the input is valid or not.
type ValidationFunction func(ctx *ValidationContext) bool

// FilterFunction FilterFunction is used to manipulate input values.
// This function may manipulate the value in place or return a completely new value.
//
// However, the contract is that they must always return a value depending on the input value and logic contained therein.
type FilterFunction func(ctx *ValidationContext) interface{}

// SetupOptions SetupOptions allows you to configure the global validation options.
func SetupOptions(configCallback func(*ValidationOptions)) {
	configCallback(&globalOptions)
}

// CopyOptions CopyOptions Copies the default global options into the specified destination.
// Useful when you want to have localized validation options
func CopyOptions(opts *ValidationOptions) {
	*opts = globalOptions
}

// AddValidator AddValidator adds the given validator function to the list of validators
//
// The backed storage containing the list of validators is not thread safe and so this function
// must be called once during package or application initialization.
//
// You cannot replace validator functions that have already been added to the list, so the function
// will panic if the name already exists.
func AddValidator(name string, v *ValidationFunction) {
	_, exists := validatorFunctions[name]
	if exists {
		panic(errors.New("a validator by the name of " + name + " already exists"))
	}
	validatorFunctions[name] = v
}

// Validate Validate validates the given struct
//
// # Parameters
//
// structPtr : Pointer to a struct
//
// localOptions: Optinal validation options, which will override the default global validation options
func Validate(structPtr interface{}, localOptions ...*ValidationOptions) (res *ValidationResult) {

	t := reflect.TypeOf(structPtr)
	res = &ValidationResult{
		valid: false,
	}

	if t.Kind() != reflect.Struct {
		res.Error = newValidationError("Invalid input type. Expected struct")
		return
	}

	// get from cache
	ctx := ValidationContext{}

	return
}

func getFields(t reflect.Type) {
	walkStruct := func(_type reflect.Type) bool {
		for i := 0; i < _type.NumField(); i++ {
			field := _type.Field(i)
			if field.Type.Kind() == reflect.Struct {

			}
		}
	}

}

func newValidationError(msg string) ValidationError {
	return ValidationError{DelegatedError: errors.New(msg)}
}
