package gen

type Stringish struct {
	S string
	L int
}

const maxSize = (32 * 64) //64 bits per uint64
const maxGuesses = 3

func NewStringish(s string) *Stringish {
	return &Stringish{s, len(s)}
}
