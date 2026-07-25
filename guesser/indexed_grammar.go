package guesser

import (
	"strconv"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// interned grammar type name (e.g. "A4", "D2", "M")
type TypeID uint16

// compact parse-tree node (8 bytes vs string-backed pcfg.PTNode)
type packedNode struct {
	Type  TypeID
	Index int32
}

// grammar entries by interned TypeID for O(1) lookup without per-node string headers
type IndexedGrammar struct {
	byID  [][]pcfg.GrammarEntry
	name  []string
	id    map[string]TypeID
	cat   []byte // first byte of type name ('A','C','M',...)

	markovID    TypeID
	hasMarkov   bool
	markovLevel []int // pre-parsed OMEN levels for type M; index = grammar entry index
}

func newIndexedGrammar(g pcfg.Grammar, base []pcfg.BaseStructure) *IndexedGrammar {
	ig := &IndexedGrammar{
		id: make(map[string]TypeID, len(g)+64),
	}

	for k := range g {
		ig.intern(k)
	}
	for i := range base {
		for _, r := range base[i].Replacements {
			ig.intern(r)
		}
	}

	n := len(ig.name)
	ig.byID = make([][]pcfg.GrammarEntry, n)
	ig.cat = make([]byte, n)
	for tid, name := range ig.name {
		ig.byID[tid] = g[name]
		if len(name) > 0 {
			ig.cat[tid] = name[0]
		}
	}

	if mid, ok := ig.id["M"]; ok {
		ig.markovID = mid
		ig.hasMarkov = true
		entries := ig.byID[mid]
		ig.markovLevel = make([]int, len(entries))
		for i, e := range entries {
			// -1 means invalid / skip (same as old strconv.Atoi error path)
			ig.markovLevel[i] = -1
			if len(e.Values) > 0 {
				level, err := strconv.Atoi(e.Values[0])
				if err == nil {
					ig.markovLevel[i] = level
				}
			}
		}
	}

	return ig
}

func (ig *IndexedGrammar) intern(s string) TypeID {
	if id, ok := ig.id[s]; ok {
		return id
	}
	id := TypeID(len(ig.name))
	ig.id[s] = id
	ig.name = append(ig.name, s)
	return id
}

func (ig *IndexedGrammar) findProb(pt []packedNode, baseProb float64) float64 {
	prob := baseProb
	for _, node := range pt {
		entries := ig.byID[node.Type]
		idx := int(node.Index)
		if idx < len(entries) {
			prob *= entries[idx].Prob
		}
	}
	return prob
}

func (ig *IndexedGrammar) entries(t TypeID) []pcfg.GrammarEntry {
	if int(t) >= len(ig.byID) {
		return nil
	}
	return ig.byID[t]
}

func (ig *IndexedGrammar) category(t TypeID) byte {
	if int(t) >= len(ig.cat) {
		return 0
	}
	return ig.cat[t]
}

func (ig *IndexedGrammar) typeName(t TypeID) string {
	if int(t) >= len(ig.name) {
		return ""
	}
	return ig.name[t]
}

func (ig *IndexedGrammar) packPT(replacements []string) []packedNode {
	pt := make([]packedNode, len(replacements))
	for i, r := range replacements {
		pt[i] = packedNode{Type: ig.intern(r), Index: 0}
	}
	return pt
}
