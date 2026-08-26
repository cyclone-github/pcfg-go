package parser

import "unicode"

func detectDigits(section Section, replacement []Section) ([]Section, string) {
	if isASCIIString(section.Value) {
		return detectDigitsASCII(section, replacement)
	}

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
				parsing := replacement[:0]

				if startPos != 0 {
					parsing = append(parsing, Section{Value: string(runes[0:startPos])})
				}

				parsing = append(parsing, Section{
					Value: foundDigit,
					Type:  lengthType('D', len([]rune(foundDigit))),
				})

				if endPos != len(runes)-1 {
					parsing = append(parsing, Section{Value: string(runes[endPos+1:])})
				}
				return parsing, foundDigit
			}
		}
	}
	return nil, ""
}

func detectDigitsASCII(section Section, replacement []Section) ([]Section, string) {
	value := section.Value
	start := -1

	for pos := 0; pos < len(value); pos++ {
		if isASCIIDigit(value[pos]) {
			if start < 0 {
				start = pos
			}
			continue
		}
		if start >= 0 {
			return buildASCIIDigitSections(section, start, pos, replacement)
		}
	}

	if start >= 0 {
		return buildASCIIDigitSections(section, start, len(value), replacement)
	}
	return nil, ""
}

func buildASCIIDigitSections(section Section, start, end int, parsing []Section) ([]Section, string) {
	found := section.Value[start:end]
	parsing = parsing[:0]
	if start != 0 {
		parsing = append(parsing, Section{Value: section.Value[:start]})
	}
	parsing = append(parsing, Section{Value: found, Type: lengthType('D', end-start)})
	if end != len(section.Value) {
		parsing = append(parsing, Section{Value: section.Value[end:]})
	}
	return parsing, found
}

func DigitDetection(sectionList []Section, replacement []Section) ([]Section, []Section) {
	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, digitStr := detectDigits(sectionList[index], replacement)
			if digitStr != "" {
				sectionList = spliceReplace(sectionList, index, parsing)
				replacement = parsing
			}
		}
		index++
	}
	return sectionList, replacement
}
