package set

type StrSet map[string]bool

func (s StrSet) Count() int {
	return len(s)
}

func (s StrSet) Add(str string) {
	s[str] = true
}

func (s StrSet) Has(str string) bool {
	a, b := s[str]
	return a && b
}
