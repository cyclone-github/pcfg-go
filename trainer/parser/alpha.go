package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func detectAlpha(section Section, mwd *TrieMultiWordDetector, replacement []Section) ([]Section, bool) {
	if isASCIIString(section.Value) {
		return detectAlphaASCII(section, mwd, replacement)
	}
	return detectAlphaUnicode(section, mwd, replacement)
}

func detectAlphaASCII(section Section, mwd *TrieMultiWordDetector, replacement []Section) ([]Section, bool) {
	value := section.Value
	start := -1

	for pos := 0; pos < len(value); pos++ {
		if isASCIIAlpha(value[pos]) {
			if start < 0 {
				start = pos
			}
			continue
		}
		if start >= 0 {
			return buildASCIIAlphaSections(section, start, pos, mwd, replacement), true
		}
	}

	if start >= 0 {
		return buildASCIIAlphaSections(section, start, len(value), mwd, replacement), true
	}
	return nil, false
}

func buildASCIIAlphaSections(section Section, start, end int, mwd *TrieMultiWordDetector, parsing []Section) []Section {
	alpha := strings.ToLower(section.Value[start:end])
	_, words := mwd.parseLowerASCII(alpha)
	parsing = parsing[:0]

	if start != 0 {
		parsing = append(parsing, Section{Value: section.Value[:start]})
	}

	current := start
	for _, word := range words {
		wordLen := len(word)
		parsing = append(parsing, Section{
			Value: section.Value[current : current+wordLen],
			Type:  lengthType('A', wordLen),
		})
		current += wordLen
	}

	if end != len(section.Value) {
		parsing = append(parsing, Section{Value: section.Value[end:]})
	}
	return parsing
}

func detectAlphaUnicode(section Section, mwd *TrieMultiWordDetector, replacement []Section) ([]Section, bool) {
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

				parsing := replacement[:0]

				if startPos != 0 {
					parsing = append(parsing, Section{Value: string(origRunes[0:startPos])})
				}

				currentStart := startPos
				for _, word := range wordList {
					wordRuneLen := utf8.RuneCountInString(word)
					parsing = append(parsing, Section{
						Value: string(origRunes[currentStart : currentStart+wordRuneLen]),
						Type:  lengthType('A', wordRuneLen),
					})
					currentStart += wordRuneLen
				}

				if endPos != len(origRunes)-1 {
					parsing = append(parsing, Section{Value: string(origRunes[endPos+1:])})
				}

				return parsing, true
			}
		}
	}
	return nil, false
}

func AlphaDetection(sectionList []Section, mwd *TrieMultiWordDetector, replacement []Section) ([]Section, []Section) {
	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, found := detectAlpha(sectionList[index], mwd, replacement)
			if found {
				sectionList = spliceReplace(sectionList, index, parsing)
				replacement = parsing
			}
		}
		index++
	}
	return sectionList, replacement
}
