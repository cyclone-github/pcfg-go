package parser

import (
	"strconv"
	"unicode"
)

type Section struct {
	Value string
	Type  string
}

func isAlpha(r rune) bool {
	return unicode.IsLetter(r)
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func runeLen(s string) int {
	return len([]rune(s))
}
