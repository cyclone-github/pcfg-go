package parser

import (
	"strings"
	"unicode"
)

type MultiWordDetector struct {
	Threshold   int
	MinLen      int
	MaxLen      int
	MinCheckLen int
	Lookup      map[rune]interface{}
}

func NewMultiWordDetector(threshold, minLen, maxLen int) *MultiWordDetector {
	return &MultiWordDetector{
		Threshold:   threshold,
		MinLen:      minLen,
		MaxLen:      maxLen,
		MinCheckLen: minLen * 2,
		Lookup:      make(map[rune]interface{}),
	}
}

type trieNode struct {
	Children map[rune]*trieNode
	Count    int
	HasCount bool
}

func newTrieNode() *trieNode {
	return &trieNode{Children: make(map[rune]*trieNode)}
}

type TrieMultiWordDetector struct {
	Threshold   int
	MinLen      int
	MaxLen      int
	MinCheckLen int
	Root        *trieNode
}

func NewTrieMultiWordDetector(threshold, minLen, maxLen int) *TrieMultiWordDetector {
	return &TrieMultiWordDetector{
		Threshold:   threshold,
		MinLen:      minLen,
		MaxLen:      maxLen,
		MinCheckLen: minLen * 2,
		Root:        newTrieNode(),
	}
}

func (d *TrieMultiWordDetector) Train(password string, setThreshold bool) {
	runes := []rune(strings.ToLower(password))

	if len(runes) < d.MinLen || len(runes) > d.MaxLen {
		return
	}

	node := d.Root
	runLen := 0

	for _, r := range runes {
		if unicode.IsLetter(r) {
			runLen++
			if _, ok := node.Children[r]; !ok {
				node.Children[r] = newTrieNode()
			}
			node = node.Children[r]
		} else {
			if runLen >= d.MinLen {
				if !node.HasCount {
					if setThreshold {
						node.Count = d.Threshold
					} else {
						node.Count = 1
					}
					node.HasCount = true
				} else {
					node.Count++
				}
			}
			runLen = 0
			node = d.Root
		}
	}

	if runLen >= d.MinLen {
		if !node.HasCount {
			if setThreshold {
				node.Count = d.Threshold
			} else {
				node.Count = 1
			}
			node.HasCount = true
		} else {
			node.Count++
		}
	}
}

func (d *TrieMultiWordDetector) getCount(s string) int {
	node := d.Root
	for _, r := range strings.ToLower(s) {
		child, ok := node.Children[r]
		if !ok {
			return 0
		}
		node = child
	}
	if node.HasCount {
		return node.Count
	}
	return 0
}

func (d *TrieMultiWordDetector) identifyMulti(alphaString string) []string {
	runes := []rune(alphaString)
	maxIndex := len(runes) - d.MinLen

	for index := maxIndex; index >= d.MinLen; index-- {
		front := string(runes[0:index])
		if d.getCount(front) >= d.Threshold {
			back := string(runes[index:])
			if d.getCount(back) >= d.Threshold {
				return []string{front, back}
			}
			results := d.identifyMulti(back)
			if results != nil {
				return append([]string{front}, results...)
			}
		}
	}
	return nil
}

func (d *TrieMultiWordDetector) Parse(alphaString string) (bool, []string) {
	runes := []rune(alphaString)

	if len(runes) < d.MinLen {
		return false, []string{alphaString}
	}
	if len(runes) >= d.MaxLen {
		return false, []string{alphaString}
	}

	if d.getCount(alphaString) >= d.Threshold {
		return true, []string{alphaString}
	}

	if len(runes) < d.MinCheckLen {
		return false, []string{alphaString}
	}

	result := d.identifyMulti(alphaString)
	if result == nil {
		return false, []string{alphaString}
	}
	return true, result
}
