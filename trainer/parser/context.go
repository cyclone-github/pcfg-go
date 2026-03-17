package parser

import (
	"strings"
	"unicode/utf8"
)

var contextSensitiveReplacements = []string{
	";p", ":p", "*0*", "#1",
	"No.1", "no.1", "No.", "i<3", "I<3", "<3",
	"Mr.", "mr.", "MR.",
	"MS.", "Ms.", "ms.",
	"Mz.", "mz.", "MZ.",
	"St.", "st.",
	"Dr.", "dr.",
}

func detectContextSensitive(section Section) ([]Section, string) {
	working := section.Value

	for _, replacement := range contextSensitiveReplacements {
		startIndex := strings.Index(working, replacement)
		if startIndex == -1 {
			continue
		}

		if replacement == "#1" {
			after := startIndex + len(replacement)
			if after < len(working) {
				r, _ := utf8.DecodeRuneInString(working[after:])
				if isDigit(r) {
					continue
				}
			}
		}

		var parsing []Section
		if startIndex != 0 {
			parsing = append(parsing, Section{Value: working[0:startIndex]})
		}
		parsing = append(parsing, Section{Value: working[startIndex : startIndex+len(replacement)], Type: "X1"})
		if startIndex+len(replacement) < len(working) {
			parsing = append(parsing, Section{Value: working[startIndex+len(replacement):]})
		}
		return parsing, replacement
	}
	return []Section{section}, ""
}

func ContextSensitiveDetection(sectionList []Section) ([]Section, []string) {
	var csList []string
	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, csString := detectContextSensitive(sectionList[index])
			if csString != "" {
				csList = append(csList, csString)
				sectionList = spliceReplace(sectionList, index, parsing)
				continue
			}
		}
		index++
	}
	return sectionList, csList
}
