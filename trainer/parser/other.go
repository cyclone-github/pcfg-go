package parser

func OtherDetection(sectionList []Section) ([]Section, []string) {
	var otherList []string
	for i := range sectionList {
		if sectionList[i].Type == "" {
			sectionList[i].Type = "O" + itoa(runeLen(sectionList[i].Value))
			otherList = append(otherList, sectionList[i].Value)
		}
	}
	return sectionList, otherList
}
