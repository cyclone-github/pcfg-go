package parser

func OtherDetection(sectionList []Section) []Section {
	for i := range sectionList {
		if sectionList[i].Type == "" {
			sectionList[i].Type = lengthType('O', runeLen(sectionList[i].Value))
		}
	}
	return sectionList
}
