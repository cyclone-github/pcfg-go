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

// numKeyboards is the number of supported keyboard layouts. Each layout is
// assigned a stable bit position (see keyboardNames / the init order below) so
// per-character position data and "active run" sets can be tracked with fixed
// arrays and uint8 bitmasks instead of per-character map allocations.
const numKeyboards = 5

var keyboardNames = [numKeyboards]string{"qwerty", "jcuken", "qwertz", "azerty", "dvorak"}

// charInfo holds, for a single rune, which keyboards contain it (present) and
// the (row, pos) of its first occurrence on each of those keyboards.
type charInfo struct {
	present uint8
	pos     [numKeyboards]keyPos
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

var (
	charPosLookup          map[rune]*charInfo
	asciiKeyboardAdjacency [128][128]uint8
)

func init() {
	kbs := []keyboardLayout{
		getUSKeyboard(),
		getJCUKENKeyboard(),
		getQWERTZKeyboard(),
		getAZERTYKeyboard(),
		getDVORAKKeyboard(),
	}

	charPosLookup = make(map[rune]*charInfo)

	for id, kb := range kbs {
		_ = keyboardNames[id]
		bit := uint8(1) << uint(id)
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
				ci := charPosLookup[c]
				if ci == nil {
					ci = &charInfo{}
					charPosLookup[c] = ci
				}
				// first occurrence of this rune on this keyboard wins
				if ci.present&bit == 0 {
					ci.present |= bit
					ci.pos[id] = keyPos{Row: r.row, Pos: i}
				}
			}
		}
	}

	for past := 0; past < len(asciiKeyboardAdjacency); past++ {
		for current := 0; current < len(asciiKeyboardAdjacency[past]); current++ {
			asciiKeyboardAdjacency[past][current] = isNextOnKeyboard(
				charPosLookup[rune(past)],
				charPosLookup[rune(current)],
			)
		}
	}
}

func findKeyboardRowColumn(ch rune) *charInfo {
	return charPosLookup[ch]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// isNextOnKeyboard returns the set (bitmask) of keyboards on which the current
// rune is adjacent to (within one row and one column of) the previous rune,
// mirroring the original map-based logic but without allocations.
func isNextOnKeyboard(past, current *charInfo) uint8 {
	if past == nil || current == nil {
		return 0
	}

	both := past.present & current.present
	if both == 0 {
		return 0
	}

	var runs uint8
	for id := 0; id < numKeyboards; id++ {
		bit := uint8(1) << uint(id)
		if both&bit == 0 {
			continue
		}
		pd := past.pos[id]
		cd := current.pos[id]
		if cd.Row == pd.Row && cd.Pos == pd.Pos {
			continue
		}
		if abs(cd.Row-pd.Row) <= 1 && abs(cd.Pos-pd.Pos) <= 1 {
			runs |= bit
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

func DetectKeyboardWalk(password string, sections []Section) []Section {
	sections = sections[:0]
	if isASCIIString(password) {
		return detectKeyboardWalkASCII(password, 4, sections)
	}
	return detectKeyboardWalkImpl([]rune(password), 4, sections)
}

func detectKeyboardWalkASCII(password string, minRun int, sectionList []Section) []Section {
	var keyboardRunList uint8
	comboStart := 0
	past := -1

	for index := 0; index < len(password); index++ {
		current := int(password[index])
		var currentRuns uint8
		if past >= 0 {
			currentRuns = asciiKeyboardAdjacency[past][current]
		}
		past = current

		if keyboardRunList == 0 {
			keyboardRunList = currentRuns
		} else {
			keyboardRunList &= currentRuns
		}
		if keyboardRunList != 0 {
			continue
		}

		if index-comboStart >= minRun {
			combo := password[comboStart:index]
			if interestingKeyboardASCII(combo) {
				if comboStart != 0 {
					sectionList = append(sectionList, Section{Value: password[:comboStart]})
				}
				sectionList = append(sectionList, Section{
					Value: combo,
					Type:  keyboardType(len(combo)),
				})
				return detectKeyboardWalkASCII(password[index:], minRun, sectionList)
			}
		}
		comboStart = index
	}

	combo := password[comboStart:]
	if len(combo) >= minRun && interestingKeyboardASCII(combo) {
		if comboStart != 0 {
			sectionList = append(sectionList, Section{Value: password[:comboStart]})
		}
		sectionList = append(sectionList, Section{
			Value: combo,
			Type:  keyboardType(len(combo)),
		})
	} else {
		sectionList = append(sectionList, Section{Value: password})
	}
	return sectionList
}

func interestingKeyboardASCII(combo string) bool {
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
	if combo[n-1] == '3' && combo[n-2] == '2' && combo[n-3] == '1' &&
		combo[n-4] != 'q' && combo[n-4] != 'Q' {
		return false
	}

	fullLower := strings.ToLower(combo)
	for _, fp := range falsePositiveWords {
		if strings.Contains(fullLower, fp) {
			return false
		}
	}

	alpha, digit, special := false, false, false
	for i := 0; i < len(combo); i++ {
		switch {
		case isASCIIAlpha(combo[i]):
			alpha = true
		case isASCIIDigit(combo[i]):
			digit = true
		default:
			special = true
		}
	}
	return boolCount(alpha, digit, special) >= 2
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func detectKeyboardWalkImpl(runes []rune, minRun int, sectionList []Section) []Section {
	var pastPos *charInfo
	var curCombo []rune
	var keyboardRunList uint8

	for index, ch := range runes {
		posList := findKeyboardRowColumn(ch)

		currentRuns := isNextOnKeyboard(pastPos, posList)
		pastPos = posList

		// mask == 0 mirrors the original "len(keyboardRunList) == 0" reset path
		if keyboardRunList == 0 {
			keyboardRunList = currentRuns
		} else {
			keyboardRunList &= currentRuns
		}

		if keyboardRunList != 0 {
			curCombo = append(curCombo, ch)
		} else {
			if len(curCombo) >= minRun {
				if interestingKeyboard(curCombo) {
					comboStr := string(curCombo)

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
						return detectKeyboardWalkImpl(runes[index:], minRun, sectionList)
					}
				}
			}
			curCombo = []rune{ch}
		}
	}

	if len(curCombo) >= minRun {
		if interestingKeyboard(curCombo) {
			comboStr := string(curCombo)

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

	return sectionList
}

func keyboardType(length int) string {
	return lengthType('K', length)
}
