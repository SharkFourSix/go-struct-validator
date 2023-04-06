package validator

import "github.com/SharkFourSix/go-struct-validator/validators"

var validatorFunctions = map[string]ValidationFunction{
	"required": validators.Required,
}

var filterFunctions = map[string]*FilterFunction{}
