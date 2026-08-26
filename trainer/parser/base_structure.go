package parser

import "strings"

func BaseStructureCreation(sectionList []Section) (bool, string) {
	var structure strings.Builder
	isSupported := true

	for _, section := range sectionList {
		if section.Type == "" {
			continue
		}
		if section.Type[0] == 'W' || section.Type[0] == 'E' {
			isSupported = false
		}
		structure.WriteString(section.Type)
	}

	return isSupported, structure.String()
}
