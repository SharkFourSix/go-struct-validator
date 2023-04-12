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

	// FilterTagName specifies the tag to use when looking up filter functions
	//
	// default: 'filter'
	FilterTagName string

	// ValidatorTagName specifies the tag to use when looking up validation functions
	//
	// default: 'validator'
	ValidatorTagName string

	// MessageTagName specifies the tag to use when looking up error message template
	//
	// default: 'message'
	MessageTagName string

	// LabelTagName specifies the tag to use when looking up the fields label
	//
	// default: 'label'
	LabelTagName string

	// StringAutoTrim specifies whether to automatically trim all strings
	//
	// default: false
	StringAutoTrim bool

	// StopOnFirstError specifies whether to stop validation upon encountering the first validation error
	//
	// default: false
	StopOnFirstError bool

	// ExposeValidatorNames specifies whether to expose validator function names in default error messages
	// when neither a validator nor a struct tag has specified an error message.
	//
	// Exposing validator names can provide technically meaningful error messages but may not be suitable for
	// client side presentation.
	//
	// default: false
	ExposeValidatorNames bool

	// NoPanicOnFunctionConflict specifies whether or not to panic upon encountering an existing validation or filter function
	// when adding custom validators and filters.
	//
	// default: false
	NoPanicOnFunctionConflict bool
}

var cache *fieldCache
var globalOptions ValidationOptions

func init() {
	// default parameters
	globalOptions = ValidationOptions{
		PrivateFields:        true,
		FilterTagName:        "filter",
		ValidatorTagName:     "validator",
		StringAutoTrim:       false,
		MessageTagName:       "message",
		LabelTagName:         "label",
		StopOnFirstError:     false,
		ExposeValidatorNames: false,
	}
	cache = &fieldCache{}
}

type fieldValueValidator struct {
	fn   ValidationFunction
	name string
	args []string
}

func (f fieldValueValidator) Apply(ctx *ValidationContext) interface{} {
	return f.fn(ctx)
}

type ValidationError struct {
	ErrorDelegate error
	Message       string
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e FieldError) Error() string {
	return e.Field + ": " + e.Message
}

func (e ValidationError) Error() string {
	if e.ErrorDelegate == nil {
		return e.Message
	} else {
		return e.Message + ": " + e.ErrorDelegate.Error()
	}
}

// ValidationResult contains validation status, a general error, and field errors
type ValidationResult struct {
	valid bool
	// Error the top level error summarizing what the hell happened. May not necessarily come from validating the passed struct
	//
	Error *ValidationError
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
type FilterFunction func(ctx *ValidationContext) reflect.Value

// SetupOptions SetupOptions allows you to configure the global validation options.
func SetupOptions(configCallback func(*ValidationOptions)) {
	configCallback(&globalOptions)
}

// CopyOptions CopyOptions Copies the default global options into the specified destination.
// Useful when you want to have localized validation options
func CopyOptions(opts *ValidationOptions) {
	*opts = globalOptions
}

// AddValidator adds the given validator function to the list of validators
//
// The backed storage containing the list of validators is not thread safe and so this function
// must be called once during package or application initialization.
//
// You cannot replace validator functions that have already been added to the list, so the function
// will panic if the name already exists.
func AddValidator(name string, v ValidationFunction) {
	_, exists := validatorFunctions[name]
	if exists && !globalOptions.NoPanicOnFunctionConflict {
		panic(errors.New("a validator by the name of " + name + " already exists"))
	} else {
		validatorFunctions[name] = v
	}
}

// AddFilter adds the given filter function to the list of filters
//
// The backed storage containing the list of filters is not thread safe and so this function
// must be called once during package or application initialization.
//
// You cannot replace filter functions that have already been added to the list, so the function
// will panic if the name already exists.
func AddFilter(name string, v FilterFunction) {
	_, exists := filterFunctions[name]
	if exists && !globalOptions.NoPanicOnFunctionConflict {
		panic(errors.New("a filter by the name of " + name + " already exists"))
	}
	filterFunctions[name] = v
}

// Validate validates the given struct
//
// # Parameters
//
// structPtr : Pointer to a struct
func Validate(structPtr interface{}) (res *ValidationResult) {

	t := reflect.TypeOf(structPtr)
	res = &ValidationResult{
		valid: false,
	}

	if t.Kind() != reflect.Ptr {
		res.Error = newValidationError("Invalid input type. Expected struct pointer but found " + t.Kind().String())
		return
	}

	t = t.Elem()
	structValue := reflect.ValueOf(structPtr).Elem()

	// get from cache
	fieldContexts := getStructFields(t, &globalOptions)

	for _, fc := range fieldContexts {
		errs := fc.apply(structValue, &globalOptions)
		if len(errs) > 0 {
			res.FieldErrors = append(res.FieldErrors, errs...)
		}
	}

	res.valid = res.Error == nil && len(res.FieldErrors) == 0

	return
}

func getStructFields(t reflect.Type, opts *ValidationOptions) []*fieldContext {
	fullyQualifiedStructName := t.PkgPath()
	if len(fullyQualifiedStructName) != 0 {
		fullyQualifiedStructName = fullyQualifiedStructName + "." + t.Name()
	}

	contexts, ok := cache.Get(fullyQualifiedStructName)
	if ok {
		return contexts
	}

	stack := Stack{}
	stack.Push(t)
	contexts = make([]*fieldContext, 0)

	for !stack.IsEmpty() {
		structType := stack.Pop().(reflect.Type)
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.Type.Kind() == reflect.Struct {
				stack.Push(field.Type)
			} else {
				fc := mustParseField(field, opts)
				if fc != nil {
					contexts = append(contexts, fc)
				}
			}
		}
	}

	// add to cache
	cache.Store(fullyQualifiedStructName, contexts)

	return contexts
}

func newValidationError(msg string, e ...error) *ValidationError {
	ve := ValidationError{Message: msg}
	if len(e) > 0 {
		ve.ErrorDelegate = e[0]
	}
	return &ve
}
