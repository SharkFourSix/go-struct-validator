package validator

import "reflect"

type fieldValueFilter struct {
	fn   FilterFunction
	name string
	args []string
}

func (f fieldValueFilter) Apply(ctx *ValidationContext) reflect.Value {
	return f.fn(ctx)
}
