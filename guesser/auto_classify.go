package guesser

import (
	"github.com/cyclone-github/pcfg-go/trainer"
	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

func newAutoParser(ig *IndexedGrammar) *trainer.PCFGParser {
	return trainer.NewPCFGParser(multiwordFromIndexed(ig))
}

// seed multiword from loaded alpha terminals (same params as pcfg_trainer)
func multiwordFromIndexed(ig *IndexedGrammar) *parser.TrieMultiWordDetector {
	mwd := parser.NewTrieMultiWordDetector(parser.DefaultThreshold, parser.DefaultMinLen, parser.DefaultMaxLen)
	if ig == nil {
		return mwd
	}
	for tid, name := range ig.name {
		if len(name) == 0 || name[0] != 'A' {
			continue
		}
		for _, e := range ig.byID[tid] {
			for _, v := range e.Values {
				mwd.Train(v, true)
			}
		}
	}
	return mwd
}

func classifyPassword(password string) (string, bool) {
	return trainer.NewPCFGParser(parser.NewTrieMultiWordDetector(parser.DefaultThreshold, parser.DefaultMinLen, parser.DefaultMaxLen)).BaseStructureOf(password)
}
