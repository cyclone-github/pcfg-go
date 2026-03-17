package parser

import "strings"

type keyboardLayout struct {
	Name  string
	Row1  []rune
	SRow1 []rune
	Row2  []rune
	SRow2 []rune
	Row3  []rune
	SRow3 []rune
	Row4  []rune
	SRow4 []rune
}

type keyPos struct {
	Row int
	Pos int
}

func getUSKeyboard() keyboardLayout {
	return keyboardLayout{
		Name:  "qwerty",
		Row1:  []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '='},
		SRow1: []rune{'!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '_', '+'},
		Row2:  []rune{'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '[', ']', '\\'},
		SRow2: []rune{'Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P', '{', '}', '|'},
		Row3:  []rune{'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', ';', '\''},
		SRow3: []rune{'A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L', ':', '"'},
		Row4:  []rune{'z', 'x', 'c', 'v', 'b', 'n', 'm', ',', '.', '/'},
		SRow4: []rune{'Z', 'X', 'C', 'V', 'B', 'N', 'M', '<', '>', '?'},
	}
}

func getJCUKENKeyboard() keyboardLayout {
	return keyboardLayout{
		Name:  "jcuken",
		Row1:  []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '='},
		SRow1: []rune{'!', '"', '№', ';', '%', ':', '?', '*', '(', ')', '_', '+'},
		Row2:  []rune{'й', 'ц', 'у', 'к', 'е', 'н', 'г', 'ш', 'щ', 'з', 'х', 'ъ', '\\'},
		SRow2: []rune{'Й', 'Ц', 'У', 'К', 'Е', 'Н', 'Г', 'Ш', 'Щ', 'З', 'Х', 'Ъ', '|'},
		Row3:  []rune{'ф', 'ы', 'в', 'а', 'п', 'р', 'о', 'л', 'д', 'ж', 'э'},
		SRow3: []rune{'Ф', 'Ы', 'В', 'А', 'П', 'Р', 'О', 'Л', 'Д', 'Ж', 'Э'},
		Row4:  []rune{'я', 'ч', 'с', 'м', 'и', 'т', 'ь', 'б', 'ю', '.'},
		SRow4: []rune{'Я', 'Ч', 'С', 'М', 'И', 'Т', 'Ь', 'Б', 'Ю', ','},
	}
}

func getQWERTZKeyboard() keyboardLayout {
	return keyboardLayout{
		Name:  "qwertz",
		Row1:  []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', 'ß', '´'},
		SRow1: []rune{'!', '"', '§', '$', '%', '&', '/', '(', ')', '=', '?', '`'},
		Row2:  []rune{'q', 'w', 'e', 'r', 't', 'z', 'u', 'i', 'o', 'p', 'ü', '+', '#'},
		SRow2: []rune{'Q', 'W', 'E', 'R', 'T', 'Z', 'U', 'I', 'O', 'P', 'Ü', '*', '\''},
		Row3:  []rune{'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'ö', 'ä'},
		SRow3: []rune{'A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L', 'Ö', 'Ä'},
		Row4:  []rune{'y', 'x', 'c', 'v', 'b', 'n', 'm', ',', '.', '-'},
		SRow4: []rune{'Y', 'X', 'C', 'V', 'B', 'N', 'M', ';', ':', '_'},
	}
}

func getAZERTYKeyboard() keyboardLayout {
	return keyboardLayout{
		Name:  "azerty",
		Row1:  []rune{'&', 'é', '"', '\'', '(', '-', 'è', '_', 'ç', 'à', ')', '='},
		SRow1: []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '°', '+'},
		Row2:  []rune{'a', 'z', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '^', '$'},
		SRow2: []rune{'A', 'Z', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P', '¨', '£'},
		Row3:  []rune{'q', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'ù'},
		SRow3: []rune{'Q', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L', 'M', '%'},
		Row4:  []rune{'w', 'x', 'c', 'v', 'b', 'n', ',', ';', ':', '!'},
		SRow4: []rune{'W', 'X', 'C', 'V', 'B', 'N', '?', '.', '/', '§'},
	}
}

func getDVORAKKeyboard() keyboardLayout {
	return keyboardLayout{
		Name:  "dvorak",
		Row1:  []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '[', ']'},
		SRow1: []rune{'!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '{', '}'},
		Row2:  []rune{'\'', ',', '.', 'p', 'y', 'f', 'g', 'c', 'r', 'l', '/', '=', '\\'},
		SRow2: []rune{'"', '<', '>', 'P', 'Y', 'F', 'G', 'C', 'R', 'L', '?', '+', '|'},
		Row3:  []rune{'a', 'o', 'e', 'u', 'i', 'd', 'h', 't', 'n', 's', '-'},
		SRow3: []rune{'A', 'O', 'E', 'U', 'I', 'D', 'H', 'T', 'N', 'S', '_'},
		Row4:  []rune{';', 'q', 'j', 'k', 'x', 'b', 'm', 'w', 'v', 'z'},
		SRow4: []rune{':', 'Q', 'J', 'K', 'X', 'B', 'M', 'W', 'V', 'Z'},
	}
}

var charPosLookup map[rune]map[string]keyPos

func init() {
	kbs := []keyboardLayout{
		getUSKeyboard(),
		getJCUKENKeyboard(),
		getQWERTZKeyboard(),
		getAZERTYKeyboard(),
		getDVORAKKeyboard(),
	}

	charPosLookup = make(map[rune]map[string]keyPos)

	for _, kb := range kbs {
		rows := []struct {
			row   int
			chars []rune
		}{
			{1, kb.Row1}, {1, kb.SRow1},
			{2, kb.Row2}, {2, kb.SRow2},
			{3, kb.Row3}, {3, kb.SRow3},
			{4, kb.Row4}, {4, kb.SRow4},
		}

		for _, r := range rows {
			for i, c := range r.chars {
				if _, exists := charPosLookup[c]; !exists {
					charPosLookup[c] = make(map[string]keyPos)
				}
				if _, exists := charPosLookup[c][kb.Name]; !exists {
					charPosLookup[c][kb.Name] = keyPos{Row: r.row, Pos: i}
				}
			}
		}
	}
}

func findKeyboardRowColumn(ch rune) map[string]keyPos {
	return charPosLookup[ch]
}

type runInfo struct {
	PastRow, PastPos int
	CurRow, CurPos   int
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func isNextOnKeyboard(past, current map[string]keyPos) map[string]runInfo {
	runs := make(map[string]runInfo)

	if past == nil || current == nil {
		return runs
	}

	for pastName, pastData := range past {
		curData, ok := current[pastName]
		if !ok {
			continue
		}

		if curData.Row == pastData.Row && curData.Pos == pastData.Pos {
			continue
		}

		if abs(curData.Row-pastData.Row) <= 1 && abs(curData.Pos-pastData.Pos) <= 1 {
			runs[pastName] = runInfo{pastData.Row, pastData.Pos, curData.Row, curData.Pos}
		}
	}

	return runs
}

var falsePositiveWords = []string{
	"drew", "kiki", "fred", "were", "pop",
	"123;", "234;", "й123",
}

func interestingKeyboard(combo []rune) bool {
	if len(combo) < 4 {
		return false
	}

	if combo[0] == 'e' {
		return false
	}
	if combo[1] == 'e' && combo[2] == 'r' {
		return false
	}
	if combo[0] == 't' && combo[1] == 'y' {
		return false
	}
	if len(combo) >= 3 && combo[0] == 't' && combo[1] == 't' && combo[2] == 'y' {
		return false
	}
	if combo[0] == 'y' {
		return false
	}
	if combo[0] == '1' && combo[1] == '2' && combo[2] == '3' {
		return false
	}

	n := len(combo)
	if n >= 4 && combo[n-1] == '3' && combo[n-2] == '2' && combo[n-3] == '1' && combo[n-4] != 'q' && combo[n-4] != 'Q' {
		return false
	}

	fullLower := strings.ToLower(string(combo))
	for _, fp := range falsePositiveWords {
		if strings.Contains(fullLower, fp) {
			return false
		}
	}

	alpha, digit, special := 0, 0, 0
	for _, r := range combo {
		if isAlpha(r) {
			alpha = 1
		} else if isDigit(r) {
			digit = 1
		} else {
			special = 1
		}
	}

	return (alpha + digit + special) >= 2
}

func DetectKeyboardWalk(password string) ([]Section, []string, []string) {
	return detectKeyboardWalkImpl([]rune(password), 4)
}

func detectKeyboardWalkImpl(runes []rune, minRun int) ([]Section, []string, []string) {
	pastPosList := map[string]keyPos{}
	curCombo := []rune{}
	keyboardRunList := map[string]runInfo{}
	foundList := []string{}
	sectionList := []Section{}
	detectedKeyboards := []string{}

	for index, ch := range runes {
		posList := findKeyboardRowColumn(ch)

		if index == 0 {
			for board := range posList {
				detectedKeyboards = append(detectedKeyboards, board)
			}
		} else {
			filtered := detectedKeyboards[:0]
			for _, k := range detectedKeyboards {
				if _, ok := posList[k]; ok {
					filtered = append(filtered, k)
				}
			}
			detectedKeyboards = filtered
		}

		currentRuns := isNextOnKeyboard(pastPosList, posList)
		pastPosList = copyPosMap(posList)

		if len(keyboardRunList) == 0 {
			keyboardRunList = copyRunMap(currentRuns)
		} else {
			for key := range keyboardRunList {
				if _, ok := currentRuns[key]; !ok {
					delete(keyboardRunList, key)
				}
			}
		}

		if len(keyboardRunList) > 0 {
			curCombo = append(curCombo, ch)
		} else {
			if len(curCombo) >= minRun {
				if interestingKeyboard(curCombo) {
					comboStr := string(curCombo)
					foundList = append(foundList, comboStr)

					if len(curCombo) != index {
						sectionList = append(sectionList, Section{
							Value: string(runes[0 : index-len(curCombo)]),
						})
					}

					sectionList = append(sectionList, Section{
						Value: comboStr,
						Type:  keyboardType(len(curCombo)),
					})

					if index != len(runes) {
						tempSections, tempFound, tempDetected := detectKeyboardWalkImpl(runes[index:], minRun)
						sectionList = append(sectionList, tempSections...)
						foundList = append(foundList, tempFound...)

						newDetected := []string{}
						for _, k := range tempDetected {
							for _, d := range detectedKeyboards {
								if k == d {
									newDetected = append(newDetected, k)
									break
								}
							}
						}
						detectedKeyboards = newDetected

						return sectionList, foundList, detectedKeyboards
					}
				}
			}
			curCombo = []rune{ch}
		}
	}

	if len(curCombo) >= minRun {
		if interestingKeyboard(curCombo) {
			comboStr := string(curCombo)
			foundList = append(foundList, comboStr)

			if len(curCombo) != len(runes) {
				sectionList = append(sectionList, Section{
					Value: string(runes[0 : len(runes)-len(curCombo)]),
				})
			}

			sectionList = append(sectionList, Section{
				Value: comboStr,
				Type:  keyboardType(len(curCombo)),
			})
		} else {
			sectionList = append(sectionList, Section{Value: string(runes)})
		}
	} else {
		sectionList = append(sectionList, Section{Value: string(runes)})
	}

	return sectionList, foundList, detectedKeyboards
}

func keyboardType(length int) string {
	return "K" + itoa(length)
}

func copyPosMap(m map[string]keyPos) map[string]keyPos {
	r := make(map[string]keyPos, len(m))
	for k, v := range m {
		r[k] = v
	}
	return r
}

func copyRunMap(m map[string]runInfo) map[string]runInfo {
	r := make(map[string]runInfo, len(m))
	for k, v := range m {
		r[k] = v
	}
	return r
}
