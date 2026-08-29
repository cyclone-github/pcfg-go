package trainer

import (
	"testing"

	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

func TestBaseStructureOfMatchesParse(t *testing.T) {
	mwd := parser.NewTrieMultiWordDetector(parser.DefaultThreshold, parser.DefaultMinLen, parser.DefaultMaxLen)
	mwd.Train("hello", true)
	mwd.Train("world", true)
	p := NewPCFGParser(mwd)
	p.Parse("helloworld")
	p.Parse("hello1234")
	p.Parse("foo bar")

	want := map[string]int{"A5A5": 1, "A5D4": 1, "A3O1A3": 1}
	for key, n := range want {
		if p.CountBaseStructs.M[key] != n {
			t.Fatalf("%s count=%d want %d map=%v", key, p.CountBaseStructs.M[key], n, p.CountBaseStructs.M)
		}
	}

	cases := []struct {
		pw, key string
	}{
		{"helloworld", "A5A5"},
		{"hello1234", "A5D4"},
		{"foo bar", "A3O1A3"},
	}
	for _, tc := range cases {
		got, ok := p.BaseStructureOf(tc.pw)
		if !ok || got != tc.key {
			t.Fatalf("%q -> %q ok=%v, want %s", tc.pw, got, ok, tc.key)
		}
	}
}
