package validator

type Stack []any

func (s *Stack) mustNotBeEmpty() {
	if s.IsEmpty() {
		panic("empty stack")
	}
}

func (s *Stack) Push(item any) {
	*s = append(*s, item)
}

func (s *Stack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *Stack) Peek() any {
	s.mustNotBeEmpty()
	return (*s)[0]
}

func (s *Stack) Pop() any {
	s.mustNotBeEmpty()
	index := len(*s) - 1
	item := (*s)[index]
	*s = (*s)[:index]
	return item
}
