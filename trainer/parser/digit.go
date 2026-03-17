package parser

import "unicode"

func detectDigits(section Section) ([]Section, string) {
	runes := []rune(section.Value)
	isRun := false
	startPos := -1

	for pos, r := range runes {
		if unicode.IsDigit(r) {
			if !isRun {
				isRun = true
				startPos = pos
			}
		}

		if !unicode.IsDigit(r) || pos == len(runes)-1 {
			if isRun {
				var endPos int
				if unicode.IsDigit(r) {
					endPos = pos
				} else {
					endPos = pos - 1
				}

				foundDigit := string(runes[startPos : endPos+1])
				var parsing []Section

				if startPos != 0 {
					parsing = append(parsing, Section{Value: string(runes[0:startPos])})
				}

				parsing = append(parsing, Section{
					Value: foundDigit,
					Type:  "D" + itoa(len([]rune(foundDigit))),
				})

				if endPos != len(runes)-1 {
					parsing = append(parsing, Section{Value: string(runes[endPos+1:])})
				}
				return parsing, foundDigit
			}
		}
	}
	return []Section{section}, ""
}

func DigitDetection(sectionList []Section) ([]Section, []string) {
	var digitList []string
	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, digitStr := detectDigits(sectionList[index])
			if digitStr != "" {
				digitList = append(digitList, digitStr)
				sectionList = spliceReplace(sectionList, index, parsing)
			}
		}
		index++
	}
	return sectionList, digitList
}
