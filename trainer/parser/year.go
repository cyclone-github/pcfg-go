package parser

import (
	"strings"
	"unicode/utf8"
)

var yearPrefixes = []string{"19", "20"}

func detectYear(section Section) ([]Section, string) {
	working := section.Value

	for _, prefix := range yearPrefixes {
		start := 0
		for {
			startIndex := strings.Index(working[start:], prefix)
			if startIndex == -1 {
				break
			}
			startIndex += start

			if len(working) < startIndex+4 {
				break
			}

			start = startIndex + 2

			if startIndex != 0 {
				r, _ := utf8.DecodeLastRuneInString(working[:startIndex])
				if isDigit(r) {
					continue
				}
			}

			if startIndex+4 < len(working) {
				r, _ := utf8.DecodeRuneInString(working[startIndex+4:])
				if isDigit(r) {
					continue
				}
			}

			// year is 4 ASCII digits, safe to index
			if isDigit(rune(working[startIndex+2])) && isDigit(rune(working[startIndex+3])) {
				year := working[startIndex : startIndex+4]
				var parsing []Section

				if startIndex != 0 {
					parsing = append(parsing, Section{Value: working[0:startIndex]})
				}

				parsing = append(parsing, Section{Value: year, Type: "Y1"})

				if startIndex+4 < len(working) {
					parsing = append(parsing, Section{Value: working[startIndex+4:]})
				}

				return parsing, year
			}
		}
	}
	return nil, ""
}

func YearDetection(sectionList []Section) []Section {
	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, year := detectYear(sectionList[index])
			if year != "" {
				sectionList = spliceReplace(sectionList, index, parsing)
				continue
			}
		}
		index++
	}
	return sectionList
}

func spliceReplace(sl []Section, index int, replacement []Section) []Section {
	if len(replacement) == 1 {
		sl[index] = replacement[0]
		return sl
	}

	newLen := len(sl) - 1 + len(replacement)
	if newLen <= cap(sl) {
		oldLen := len(sl)
		sl = sl[:newLen]
		copy(sl[index+len(replacement):], sl[index+1:oldLen])
		copy(sl[index:], replacement)
		return sl
	}

	result := make([]Section, 0, len(sl)-1+len(replacement))
	result = append(result, sl[:index]...)
	result = append(result, replacement...)
	result = append(result, sl[index+1:]...)
	return result
}
