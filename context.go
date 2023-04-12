package validator

import (
	"reflect"
	"strconv"
)

type ValidationContext struct {
	// The input value to be validated
	value reflect.Value

	// The resolved kind of the input value
	valueKind reflect.Kind

	// The resolved type of the input value
	ValueType reflect.Type

	// If the input value is a pointer
	IsPointer bool

	// If the input value is a pointer and the point is null
	IsNull bool

	// Validation options
	Options *ValidationOptions

	// Arguments passed to the validation or filter function
	Args []string

	// Containst the validation error message
	ErrorMessage string

	// An error that may have occurred during validation
	AdditionalError error
}

// GetValue GetValue Returns the underlying value, resolving pointers if necessary
func (vc ValidationContext) GetValue() reflect.Value {
	if vc.IsPointer {
		return vc.value.Elem()
	} else {
		return vc.value
	}
}

func (vc ValidationContext) ArgCount() int {
	return len(vc.Args)
}

func (vc ValidationContext) IsValueOfKind(kind ...reflect.Kind) bool {
	_len := len(kind)
	if _len == 0 {
		return false
	} else {
		for _, k := range kind {
			if vc.valueKind == k {
				return true
			}
		}
		return false
	}
}

// ValueMustBeOfKind ValueMustBeOfKind tests if the resolved kind of the input value matches any of the given kinds.
//
// If there is no match, the function panics.
func (vc *ValidationContext) ValueMustBeOfKind(kind ...reflect.Kind) {
	for _, k := range kind {
		if vc.valueKind == k {
			return
		}
	}
	panic(newValidationError("unexpected type found: " + vc.valueKind.String()))
}

func (vc *ValidationContext) MustGetIntArg(position int) int64 {
	value := vc.Args[position]
	intv, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(newValidationError("error getting integer parmeter value", err))
	}
	return intv
}

func (vc *ValidationContext) MustGetUintArg(position int) uint64 {
	value := vc.Args[position]
	intv, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		panic(newValidationError("error getting unsigned integer parmeter value", err))
	}
	return intv
}

func (vc *ValidationContext) IsValueOfType(i interface{}) bool {
	return vc.ValueType.AssignableTo(reflect.TypeOf(i))
}
