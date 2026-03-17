package parser

import "strings"

func BaseStructureCreation(sectionList []Section) (bool, string) {
	var parts []string
	isSupported := true

	for _, section := range sectionList {
		if section.Type == "" {

			continue
		}
		if len(section.Type) > 0 && (section.Type[0] == 'W' || section.Type[0] == 'E') {
			isSupported = false
		}
		parts = append(parts, section.Type)
	}

	return isSupported, strings.Join(parts, "")
}
