package parser

import "unicode"

func CheckValid(password string) bool {
	if len(password) == 0 {
		return false
	}

	for _, r := range password {
		if r == '\t' {
			return false
		}

		if r < 0x20 {
			return false
		}

		if r == '\u2028' {
			return false
		}

		if r == '\u0085' {
			return false
		}
	}

	return true
}

func IsControlChar(r rune) bool {
	return unicode.IsControl(r) && r != '\t'
}
