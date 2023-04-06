package validator

type fieldValueFilter struct {
	fn   FilterFunction
	name string
}

func (f fieldValueFilter) Apply(ctx *ValidationContext) interface{} {
	return f.fn(ctx)
}
