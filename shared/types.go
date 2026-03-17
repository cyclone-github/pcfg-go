package pcfg

// section represents a parsed segment of a password
// value is the raw string content, Type is the classification (e.g. "A4", "D2", "K5", "Y1", nil)
type Section struct {
	Value string
	Type  string // empty string means unclassified
}

// groups terminal values sharing the same probability
type GrammarEntry struct {
	Values []string
	Prob   float64
}

// represents a base structure with its probability and replacement types
type BaseStructure struct {
	Prob         float64
	Replacements []string
}

// parse tree item used in the priority queue
type PTItem struct {
	Prob     float64
	PT       []PTNode
	BaseProb float64
}

// single node in a parse tree: a type (e.g. "A4") and an index into the grammar
type PTNode struct {
	Type  string
	Index int
}

// maps transition type names to ordered slices of GrammarEntry
type Grammar map[string][]GrammarEntry

// TransitionID constants
const (
	Alpha    = "A"
	Digit    = "D"
	Other    = "O"
	Keyboard = "K"
	Context  = "X"
	Year     = "Y"
	Markov   = "M"
	Cap      = "C"
	Email    = "E"
	Website  = "W"
)
