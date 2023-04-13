package validator

// Flags used by the validation engine before calling validations and filters.
//
// Flags will alter behavior of the validator towards each field being evaluated.
type ValidationFlag string

const (
	// If a value contains zero value, allow the value to pass through by skipping
	// validation since there's nothing to validate or filter.
	AllowZero ValidationFlag = "allow_zero"
)
