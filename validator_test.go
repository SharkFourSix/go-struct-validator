package validator_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	validator "github.com/SharkFourSix/go-struct-validator"
)

func init() {
	validator.SetupOptions(func(opts *validator.ValidationOptions) {
		opts.ExposeEnumValues = true
		opts.ExposeValidatorNames = true
		opts.NoPanicOnFunctionConflict = true
	})
}

func TestValidator(t *testing.T) {

	type MyStruct struct {
		Name string `filter:"trim"`
		Age  int    `validator:"required|min(21)|max(35)"`
	}

	myStruct := MyStruct{Age: 21, Name: "  John Doe  "}

	res := validator.Validate(&myStruct)
	fmt.Println(res)

	assert.Equal(t, myStruct.Name, "John Doe")
	assert.True(t, res.IsValid(), "Validation failed")
}

func TestCustomFunctions(t *testing.T) {
	type MyStruct struct {
		Name string `filter:"trim|upper"`
		Age  int    `validator:"range(10,50)" filter:"square|square"`
	}

	myStruct := MyStruct{Age: 20, Name: "  John Doe  "}

	// install custom validator function
	validator.AddValidator("range", func(ctx *validator.ValidationContext) bool {
		if ctx.ArgCount() != 2 {
			// instead of panicking, we can simply return an api-level error message
			ctx.ErrorMessage = "range function requires exactly two parameters"
			return false
		}
		min := ctx.MustGetIntArg(0)
		max := ctx.MustGetIntArg(1)

		if ctx.IsNull {
			return false
		}
		value := ctx.GetValue().Int()
		if value >= min && value <= max {
			return true
		}
		ctx.ErrorMessage = "value must be between " + strconv.FormatInt(min, 10) + " and " + strconv.FormatInt(max, 10)
		return false
	})

	// this filter converts the given string to upper case
	validator.AddFilter("upper", func(ctx *validator.ValidationContext) reflect.Value {
		ctx.ValueMustBeOfKind(reflect.String)

		// the filter is aware of pointers
		if !ctx.IsNull {
			value := ctx.GetValue().String()
			return reflect.ValueOf(strings.ToUpper(value))
		}

		// simply return what's alreay there
		return ctx.GetValue()
	})

	// this filter squares the given int value
	validator.AddFilter("square", func(ctx *validator.ValidationContext) reflect.Value {
		ctx.ValueMustBeOfKind(reflect.Int)

		// the filter is aware of pointers
		if !ctx.IsNull {
			value := int(ctx.GetValue().Int())
			value = value * value
			return reflect.ValueOf(value)
		}

		// simply return what's alreay there
		return ctx.GetValue()
	})

	res := validator.Validate(&myStruct)
	fmt.Println(res)

	assert.Equal(t, myStruct.Name, "JOHN DOE")
	assert.Equal(t, myStruct.Age, 160000)
	assert.True(t, res.IsValid(), "Validation failed")
}

func TestPointer(t *testing.T) {
	age := int(42)
	name := " Bames Jond "

	type MyStruct struct {
		Name *string `filter:"trim|upper"`
		Age  *int    `validator:"min(50)" filter:"square" label:"Agent age"`
	}

	myStruct := MyStruct{Age: &age, Name: &name}

	validator.AddFilter("upper", func(ctx *validator.ValidationContext) reflect.Value {
		ctx.ValueMustBeOfKind(reflect.String)

		// the filter is aware of pointers
		if ctx.IsPointer && !ctx.IsNull {
			value := strings.ToUpper(ctx.GetValue().String())
			return reflect.ValueOf(&value)
		}

		// simply return what's alreay there
		return ctx.GetValue()
	})

	validator.AddFilter("square", func(ctx *validator.ValidationContext) reflect.Value {
		ctx.ValueMustBeOfKind(reflect.Int)

		// the filter is aware of pointers
		if ctx.IsPointer && !ctx.IsNull {
			value := int(ctx.GetValue().Int())
			value = value * value
			return reflect.ValueOf(&value)
		}

		// simply return what's alreay there
		return ctx.GetValue()
	})

	res := validator.Validate(&myStruct)
	fmt.Println(res)

	assert.Equal(t, *myStruct.Name, "BAMES JOND")
	assert.Equal(t, *myStruct.Age, 1764)
	assert.False(t, res.IsValid(), "Validation failed")
}

func TestEnum(t *testing.T) {
	type MyEnum int
	const (
		Opt1 MyEnum = iota
		Opt2 MyEnum = iota
	)

	type MyStruct struct {
		EnumValue *MyEnum `validator:"enum(0,1)"`
	}

	opt := Opt2
	myStruct := MyStruct{EnumValue: &opt}

	res := validator.Validate(&myStruct)
	fmt.Println(res)

	assert.Equal(t, *myStruct.EnumValue, Opt2)
	assert.True(t, res.IsValid(), "Validation failed")
}

func TestOptional(t *testing.T) {

	type MyStruct struct {
		Optional *string `validator:"min(10)|max(20)"`
	}

	myStruct := MyStruct{Optional: nil}

	res := validator.Validate(&myStruct)
	fmt.Println(res)

	assert.True(t, res.IsValid(), "Validation failed")
}

func TestRequired(t *testing.T) {

	type MyStruct struct {
		Optional *string `validator:"required|min(10)|max(20)"`
	}

	myStruct := MyStruct{Optional: nil}

	res := validator.Validate(&myStruct)
	fmt.Println(res)

	assert.False(t, res.IsValid(), "Validation failed")
}

func TestNested(t *testing.T) {
	type struct2 struct {
		Age int `validator:"min(18)" message:"must be at least 18 to open an account here"`
	}
	type Struct1 struct {
		struct2
		Foo int `validator:"min(100)" label:"Deposit amount"`
	}
	type MyStruct struct {
		Struct1
		Bar int `validator:"min(10)|max(20)"`
	}

	myStruct := MyStruct{Bar: 15}

	res := validator.Validate(&myStruct)
	fmt.Println(res)

	assert.False(t, res.IsValid(), "Validation failed")
}

func TestActivationTrigger(t *testing.T) {

	type Person struct {
		Id     int `validator:"min(1000)" message:"Employment ID numbers start at 1000" trigger:"update"`
		Age    int `filter:"add_five"` // implicit 'all'
		Height int `filter:"add_five|add_five" trigger:"create,update"`
	}

	validator.AddFilter("add_five", func(ctx *validator.ValidationContext) reflect.Value {
		ctx.ValueMustBeOfKind(reflect.Int)

		if !ctx.IsNull {
			value := ctx.GetValue().Int()
			value += 5
			if ctx.IsPointer {
				ctx.GetValue().Set(reflect.ValueOf(&value))
			} else {
				ctx.GetValue().SetInt(value)
			}
		}

		return ctx.GetValue()
	})

	person := Person{Id: 0, Age: 2, Height: 1}

	// create user
	res := validator.Validate(&person, "create")
	fmt.Println(res)

	assert.Equal(t, person.Id, 0)
	assert.Equal(t, person.Age, 7)
	assert.Equal(t, person.Height, 11)
	assert.True(t, res.IsValid(), "Validation failed")

	// assign id
	person.Id = 999

	// update user
	res = validator.Validate(&person, "update")
	fmt.Println(res)

	assert.Equal(t, person.Id, 999)
	assert.Equal(t, person.Age, 12)    //---\
	assert.Equal(t, person.Height, 21) // ------>  opts.StopOnFirstError = false
	assert.False(t, res.IsValid(), "Validation failed")
}
