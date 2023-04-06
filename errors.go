package validator

type errorSkipField struct {
	Name string
}

func (e errorSkipField) Error() string {
	return ""
}
