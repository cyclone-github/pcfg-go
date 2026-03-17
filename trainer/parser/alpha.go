package parser

import (
	"strings"
	"unicode"
)

func detectAlpha(section Section, mwd *TrieMultiWordDetector) ([]Section, []string, []string) {
	origRunes := []rune(section.Value)
	workRunes := []rune(strings.ToLower(section.Value))

	isRun := false
	startPos := -1

	for pos, r := range workRunes {
		if unicode.IsLetter(r) {
			if !isRun {
				isRun = true
				startPos = pos
			}
		}

		if !unicode.IsLetter(r) || pos == len(workRunes)-1 {
			if isRun {
				var endPos int
				if unicode.IsLetter(r) {
					endPos = pos
				} else {
					endPos = pos - 1
				}

				alphaStr := string(workRunes[startPos : endPos+1])
				_, wordList := mwd.Parse(alphaStr)

				var maskList []string
				var parsing []Section

				if startPos != 0 {
					parsing = append(parsing, Section{Value: string(origRunes[0:startPos])})
				}

				currentStart := startPos
				for _, word := range wordList {
					wordRuneLen := len([]rune(word))
					parsing = append(parsing, Section{
						Value: string(origRunes[currentStart : currentStart+wordRuneLen]),
						Type:  "A" + itoa(wordRuneLen),
					})

					var mask strings.Builder
					for _, lr := range origRunes[currentStart : currentStart+wordRuneLen] {
						if unicode.IsUpper(lr) {
							mask.WriteByte('U')
						} else {
							mask.WriteByte('L')
						}
					}
					maskList = append(maskList, mask.String())
					currentStart += wordRuneLen
				}

				if endPos != len(origRunes)-1 {
					parsing = append(parsing, Section{Value: string(origRunes[endPos+1:])})
				}

				return parsing, wordList, maskList
			}
		}
	}
	return []Section{section}, nil, nil
}

func AlphaDetection(sectionList []Section, mwd *TrieMultiWordDetector) ([]Section, []string, []string) {
	var alphaList []string
	var maskList []string

	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, alphas, masks := detectAlpha(sectionList[index], mwd)
			if alphas != nil {
				alphaList = append(alphaList, alphas...)
				maskList = append(maskList, masks...)
				sectionList = spliceReplace(sectionList, index, parsing)
			}
		}
		index++
	}
	return sectionList, alphaList, maskList
}
