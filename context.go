package validator

import "reflect"

type ValidationContext struct {
	Value     interface{}
	valueKind reflect.Kind
	IsPointer bool
	IsNull    bool
	Options   *ValidationOptions
	Args      []interface{}
}

func (vc ValidationContext) ArgCount() int {
	return len(vc.Args)
}

func (vc ValidationContext) IsValueOfKind(kind reflect.Kind) bool {
	return vc.valueKind == kind
}

func (vc ValidationContext) IsArgumentOfKind(indx int, kind reflect.Kind) bool {
	return reflect.TypeOf(vc.Args[indx]).Kind() == kind
}
