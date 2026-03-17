package parser

import (
	"strings"
	"unicode/utf8"
)

func detectEmail(section Section) ([]Section, string, string) {
	working := strings.ToLower(section.Value)

	if !strings.Contains(working, ".") || !strings.Contains(working, "@") {
		return []Section{section}, "", ""
	}

	tldList := getTLDList()

	for _, tld := range tldList {
		endIndex := strings.Index(working, tld)
		if endIndex == -1 {
			continue
		}
		endIndex += len(tld)

		markerIndex := strings.Index(working[0:endIndex], "@")
		if markerIndex == -1 {
			continue
		}

		// extract provider using rune boundaries for multi-byte safety
		providerBytes := working[markerIndex+1 : endIndex]
		if !utf8.ValidString(providerBytes) {
			continue
		}
		provider := string([]rune(providerBytes))
		found := working[0:endIndex]

		var parsing []Section
		parsing = append(parsing, Section{Value: section.Value[0:endIndex], Type: "E"})

		if endIndex != len(working) {
			parsing = append(parsing, Section{Value: section.Value[endIndex:]})
		}

		return parsing, found, provider
	}
	return []Section{section}, "", ""
}

func EmailDetection(sectionList []Section) ([]Section, []string, []string) {
	var emailList, providerList []string

	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, email, provider := detectEmail(sectionList[index])
			if email != "" {
				emailList = append(emailList, email)
				providerList = append(providerList, provider)
				sectionList = spliceReplace(sectionList, index, parsing)
			}
		}
		index++
	}
	return sectionList, emailList, providerList
}
