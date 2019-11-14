package shifters

type Replacer struct {
	From string
	To   string
}

type Replacers []Replacer

type Shifter interface {
	Shift(r Replacers) error
}
