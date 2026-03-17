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
	return []Section{section}, ""
}

func YearDetection(sectionList []Section) ([]Section, []string) {
	var yearList []string
	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, year := detectYear(sectionList[index])
			if year != "" {
				yearList = append(yearList, year)
				sectionList = spliceReplace(sectionList, index, parsing)
				continue
			}
		}
		index++
	}
	return sectionList, yearList
}

func spliceReplace(sl []Section, index int, replacement []Section) []Section {
	result := make([]Section, 0, len(sl)-1+len(replacement))
	result = append(result, sl[:index]...)
	result = append(result, replacement...)
	result = append(result, sl[index+1:]...)
	return result
}
