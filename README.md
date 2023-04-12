# go-struct-validator

Golang Struct validator.

### Features

- [x] Nested struct validation
- [x] Activation triggers: Allows selective validation and struct re-use.

### Getting Started

A guide on how to quickly get started.

1. Customize validation options (optional)
2. Add custom filter/validation functions (optional)
3. Validate away!

```go
go get github.com/SharkFourSix/go-strutct-validator
```

```go
package main
import "github.com/SharkFourSix/go-strutct-validator"

type Person {
    Id int `validator:"min(1000)" trigger:"update,delete"` // evaluate only when 'update' or 'delete' triggers have been specified
    Age int `validator:"min(18)|max(65)"` // n >= 18 and n <= 65
    Name string `validator:"max(20)" filter:"upper"` // len(s) <= 20
}

func main(){

    // 1. Customize validation options
    validator.SetupOptions(func(opts *validator.ValidationOptions){
        // override options here. See available options
    })

    // 2.Add custom validation and filter functions
    validator.AddFilter("upper", func(ctx *validator.ValidationContext) reflect.Value{
        ctx.ValueMustBeOfKind(reflect.String)

        if !ctx.IsNull {
            stringValue := strings.ToUpper(ctx.GetValue().String())
            if ctx.IsPointer {
                ctx.GetValue().Set(&stringValue)
            }else{
                ctx.GetValue().SetString(stringValue)
            }
        }

        return ctx.GetValue()
    })

    person := Person{Age: 20, Name: "Bames Jond"}

    // validate
    result := validator.Validate(&person)
    if result.IsValid() {
        fmt.Println("validation passed")
    }else{
        fmt.Println(result.Error)
        fmt.Println(result.FieldErrors)
    }
}
```

### Design Philosophy

This librabry provides validation functionality for structs only. It does not support
data binding.

Each validation rule correspods to a function, which accepts a `validator.ValidationContext`, containing the value to validate as well as any arguments that the function may require.

#### Validating required vs optional values

The contract for validating pointer values is to first inspect whether the pointer is null (`validationContext.IsNull`), returning `true` if that is the case, implying an optional value, or, continuing with the validation in case the pointer is not null.

The contract for validating literal values is to inspect the values and perform validation logic accordingly.

#### Validation functions and filters

Both validation and filter functions accept the same input parameter `validator.ValidationContext`.

Validation functions return true or false (`bool`) to indicate whether the validation test passed or failed. If the validation function wishes to provide an error message, it may do so through `validator.ValidationContext.ErrorMessage`.

The goal of filter functions is to allow transforming data into desired and suitable formats.

> **NOTE**: Because this is not a data binding library, filters may not change the data type since the type of the input value cannot be changed.

Filters return `reflect.Value`, which may be a newly allocated value or simply the same value found stored in `validator.ValidationContext.value`.

To access the input value within a filter or validator, call `ValidationContext.GetValue()`, which will return the underlying value (`reflect.Value`), resolving pointers (1 level deep) if necessary.

To check the type of the input value, you can use `ValidationContext.IsValueOfKind(...reflect.Kind)` or `ValidationContext.IsValueOfType(inteface{})`.

Sample validator

```go
func MyValidator(ctx *validator.ValidationContext) bool {
    // First always check if the value is (a pointer and) null.
    if ctx.IsNull {
        // treat this as an optional field. if the caller decides otherwise, the first validtor in the list will be "requried"
        return true
    }

    // ..check supported type (will panic)
    ctx.MustBeOfKind(reflect.String)

    // or only check for supported types only without needing to panic
    if ctx.IsValueOfKind(reflect.String) {
        myString := ctx.GetValue().String()

        // apply validation logic
        if !strings.HasSuffix(myString, ".com") {

            // provide an optional error message. The validation orchestrator will set one for you if you do not specify one
            ctx.ErrorMessage = "only .com domains are allowed"

            return false
        }
    }else{
        // panic because this is not a validation error, rather a type/low level error that needs to be fixed
        panic("unsupported type " + ctx.ValueKind.String())
    }

    return true
}
```

Sample filter

```go
func InPlaceMutationFilter(ctx *validator.ValidationContext) reflect.Value {

    // Check supported type (will panic)
    ctx.MustBeOfKind(reflect.Int)

    // always check if the value is (a pointer and) null.
    if !ctx.IsNull {
        myNumber := ctx.GetValue().Int()
        myNumber = myNumber * myNumber

        // update the value in place
        if ctx.IsPointer {
            ctx.GetValue().Set(&myNumber)
        }else{
            ctx.GetValue().Set(myNumber)
        }
    }

    return ctx.GetValue()
}
```

```go
func NewValueFilter(ctx *validator.ValidationContext) reflect.Value {

    // Check supported type (will panic)
    ctx.MustBeOfKind(reflect.Int)

    value := ctx.GetValue()

    // always check if the value is (a pointer and) null.
    if !ctx.IsNull {
        myNumber := value.Int()
        myNumber = myNumber * myNumber

        // return a new value
        if ctx.IsPointer {
            value = reflect.ValueOf(&myNumber)
        }else{
            value = reflect.ValueOf(myNumber)
        }
    }

    return value
}
```

#### Execution order and activation

**Selective Validation**

Sometimes you may wish to use the same struct but only work with specific fields in specific cases. Instead of creating a struct for each use case, you can use activation triggers to selectively evaluate those specific fields.

To specify an activation trigger, include the name of the trigger in the `trigger` tag.

> **NOTE** Trigger names can be anything.
>
> A special activation trigger **'all'** exists which causes a field to be evaluated in all use cases. Omitting the `trigger` tag is equivalent to explicitly specifying ONLY this special value.

```go
type ResourceRequest struct {
    ResourceId string `validator:"uuidv4" trigger:"update,delete"`
    ResourceName string `validator:"min(3)|max(50)"`
}

myResource := ResourceRequest{}

// get from some http request
httpRequestDataBinder.BindData(&myResource)

// making the following call validates .ResourceName
validator.Validate(&myResource, "create")

// ... later on when updating the resource name, both .ResourceId and .ResourceName
// will get evaludated
validator.Validate(&myResource, "update")
```

**Execution Order**

Validators are evaluated first and filters last.

#### Accessing validation errors

`validator.ValidationResults.IsValid()` indicates whether validation succeeded or not. If validation did not exceed, you are guaranteed to have at least one validation error in `validator.ValidationResults.FieldErrors`.

Each field error contains the label take from the field name or `label` tag, and error message returned by the failing validation function, or taken from the `message` tag or default generic error message if none of the former options were specified.

```go
func main(){
    type Person {
        Age int `validator:"min(18)|max(65)" message:"You're too young or too old for this"`
        Name string `validator:"min(20)" filter:"upper" message:"Your name is too long" label:"Candidate name"`
    }

    person := Person{Age: 16, Name: "Bames Jond"}

    // validate
    result := validator.Validate(&person)
    if result.IsValid() {
        fmt.Println("validation passed")
    }else{
        fmt.Println(result.Error)
        fmt.Println(result.FieldErrors)
    }
}
```

### Packaged validators

| Name           | Function        | Parameters                |
| -------------- | --------------- | ------------------------- |
| required       | IsRequired      |
| alphanum       | IsAlphaNumeric  |
| uuid1          | IsUuid1         |
| uuid2          | IsUuid2         |
| uuid3          | IsUuid3         |
| uuid4          | IsUuid4         |
| min            | IsMin           | (number)                  |
| max            | IsMax           | (number)                  |
| enum           | IsEnum          | (...string)               |
| email          | IsEmail         |
| at_least_today | IsOrBeforeToday | (dateLayout) - _optional_ |
| at_most_today  | IsOrAfterToday  | (dateLayout) - _optional_ |
| today          | IsToday         | (dateLayout) - _optional_ |
| before_today   | IsBeforeToday   | (dateLayout) - _optional_ |
| after_today    | IsAfterToday    | (dateLayout) - _optional_ |

### Packaged filters

| Name | Function | Parameters | Description       |
| ---- | -------- | ---------- | ----------------- |
| trim | Trim     |            | Trim string space |

### Validation options

Refer to `validator.ValidationOptions` to see list of options in [validator.go](validator.go)

### Documentation

https://pkg.go.dev/github.com/SharkFourSix/go-struct-validator#section-documentation

### Contribution

Contributions are welcome


Inspiration taken from https://github.com/gookit/validate

### TODO

- [ ] Add static analysis workflow file