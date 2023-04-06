// Contains all built-in validators
package validators

import validator "github.com/SharkFourSix/go-struct-validator"

// Required Required check if the required field has values.
//
// For literal values, the function always returns true.
//
// For pointer types, the function will return false if the pointer is nil or true if the pointer is non nil
func Required(ctx *validator.ValidationContext) bool {
	if ctx.isPointer {
		return !ctx.isNull
	}
	return true
}
