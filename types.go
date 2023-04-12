package validator

type Comparator string
type ComparatorDescription byte

const (
	EQUALS                Comparator = "="
	NOT_EQUAL             Comparator = "!="
	LESS_THAN             Comparator = "<"
	GREATER_THAN          Comparator = ">"
	LESS_THAN_OR_EQUAL    Comparator = "<="
	GREATER_THAN_OR_EQUAL Comparator = ">="
)

const (
	NUMERICAL ComparatorDescription = 0
	TEMPORAL  ComparatorDescription = 1
)

var comparatorDescriptors = map[Comparator][]string{
	EQUALS:                {"equal", "the same as"},
	NOT_EQUAL:             {"not equal", "not the same as"},
	LESS_THAN:             {"less than", "before"},
	GREATER_THAN:          {"greater than", "after"},
	LESS_THAN_OR_EQUAL:    {"less than or equal", "at most"},
	GREATER_THAN_OR_EQUAL: {"greater than or equal", "at least"},
}

func (c Comparator) NumericDescription() string {
	return comparatorDescriptors[c][NUMERICAL]
}

func (c Comparator) TemporalDescription() string {
	return comparatorDescriptors[c][TEMPORAL]
}
