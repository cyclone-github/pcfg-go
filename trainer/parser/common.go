package parser

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

type Section struct {
	Value string
	Type  string
}

const cachedSectionTypeLength = 256

var (
	alphaSectionTypes    [cachedSectionTypeLength + 1]string
	digitSectionTypes    [cachedSectionTypeLength + 1]string
	otherSectionTypes    [cachedSectionTypeLength + 1]string
	keyboardSectionTypes [cachedSectionTypeLength + 1]string
)

func init() {
	for length := 1; length <= cachedSectionTypeLength; length++ {
		suffix := strconv.Itoa(length)
		alphaSectionTypes[length] = "A" + suffix
		digitSectionTypes[length] = "D" + suffix
		otherSectionTypes[length] = "O" + suffix
		keyboardSectionTypes[length] = "K" + suffix
	}
}

func isAlpha(r rune) bool {
	return unicode.IsLetter(r)
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

func lengthType(kind byte, length int) string {
	if length > 0 && length <= cachedSectionTypeLength {
		switch kind {
		case 'A':
			return alphaSectionTypes[length]
		case 'D':
			return digitSectionTypes[length]
		case 'O':
			return otherSectionTypes[length]
		case 'K':
			return keyboardSectionTypes[length]
		}
	}
	return string(kind) + strconv.Itoa(length)
}

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func isASCIIString(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func isASCIIAlpha(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
