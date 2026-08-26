package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type TrieMultiWordDetector struct {
	Threshold   int
	MinLen      int
	MaxLen      int
	MinCheckLen int
	Counts      map[string]int
}

func NewTrieMultiWordDetector(threshold, minLen, maxLen int) *TrieMultiWordDetector {
	return &TrieMultiWordDetector{
		Threshold:   threshold,
		MinLen:      minLen,
		MaxLen:      maxLen,
		MinCheckLen: minLen * 2,
		Counts:      make(map[string]int),
	}
}

func (d *TrieMultiWordDetector) Train(password string, setThreshold bool) {
	if isASCIIString(password) {
		d.trainASCII(password, setThreshold)
		return
	}

	lower := strings.ToLower(password)

	runeCount := utf8.RuneCountInString(lower)
	if runeCount < d.MinLen || runeCount > d.MaxLen {
		return
	}

	runStart := -1
	runLen := 0

	for bytePos, r := range lower {
		if unicode.IsLetter(r) {
			if runStart < 0 {
				runStart = bytePos
			}
			runLen++
		} else {
			if runLen >= d.MinLen {
				d.add(lower[runStart:bytePos], setThreshold)
			}
			runStart = -1
			runLen = 0
		}
	}

	if runLen >= d.MinLen {
		d.add(lower[runStart:], setThreshold)
	}
}

func (d *TrieMultiWordDetector) trainASCII(password string, setThreshold bool) {
	if len(password) < d.MinLen || len(password) > d.MaxLen {
		return
	}

	runStart := -1
	for pos := 0; pos < len(password); pos++ {
		if isASCIIAlpha(password[pos]) {
			if runStart < 0 {
				runStart = pos
			}
			continue
		}
		if runStart >= 0 && pos-runStart >= d.MinLen {
			d.add(strings.ToLower(password[runStart:pos]), setThreshold)
		}
		runStart = -1
	}
	if runStart >= 0 && len(password)-runStart >= d.MinLen {
		d.add(strings.ToLower(password[runStart:]), setThreshold)
	}
}

func (d *TrieMultiWordDetector) add(word string, setThreshold bool) {
	if count, ok := d.Counts[word]; ok {
		d.Counts[word] = count + 1
	} else if setThreshold {
		d.Counts[word] = d.Threshold
	} else {
		d.Counts[word] = 1
	}
}

// MergeFrom folds another detector's exact word counts into this one.
func (d *TrieMultiWordDetector) MergeFrom(other *TrieMultiWordDetector) {
	for word, count := range other.Counts {
		d.Counts[word] += count
	}
}

func (d *TrieMultiWordDetector) getCount(s string) int {
	return d.Counts[strings.ToLower(s)]
}

func (d *TrieMultiWordDetector) identifyMulti(alphaString string) []string {
	runes := []rune(alphaString)
	maxIndex := len(runes) - d.MinLen

	for index := maxIndex; index >= d.MinLen; index-- {
		front := string(runes[0:index])
		if d.getCount(front) >= d.Threshold {
			back := string(runes[index:])
			if d.getCount(back) >= d.Threshold {
				return []string{front, back}
			}
			results := d.identifyMulti(back)
			if results != nil {
				return append([]string{front}, results...)
			}
		}
	}
	return nil
}

func (d *TrieMultiWordDetector) identifyMultiASCII(alphaString string) []string {
	maxIndex := len(alphaString) - d.MinLen

	for index := maxIndex; index >= d.MinLen; index-- {
		front := alphaString[:index]
		if d.getCount(front) >= d.Threshold {
			back := alphaString[index:]
			if d.getCount(back) >= d.Threshold {
				return []string{front, back}
			}
			results := d.identifyMultiASCII(back)
			if results != nil {
				return append([]string{front}, results...)
			}
		}
	}
	return nil
}

func (d *TrieMultiWordDetector) identifyMultiLowerASCII(alphaString string) []string {
	maxIndex := len(alphaString) - d.MinLen

	for index := maxIndex; index >= d.MinLen; index-- {
		front := alphaString[:index]
		if d.Counts[front] >= d.Threshold {
			back := alphaString[index:]
			if d.Counts[back] >= d.Threshold {
				return []string{front, back}
			}
			results := d.identifyMultiLowerASCII(back)
			if results != nil {
				return append([]string{front}, results...)
			}
		}
	}
	return nil
}

func (d *TrieMultiWordDetector) Parse(alphaString string) (bool, []string) {
	if isASCIIString(alphaString) {
		return d.parseASCII(alphaString)
	}

	runes := []rune(alphaString)

	if len(runes) < d.MinLen {
		return false, []string{alphaString}
	}
	if len(runes) >= d.MaxLen {
		return false, []string{alphaString}
	}

	if d.getCount(alphaString) >= d.Threshold {
		return true, []string{alphaString}
	}

	if len(runes) < d.MinCheckLen {
		return false, []string{alphaString}
	}

	result := d.identifyMulti(alphaString)
	if result == nil {
		return false, []string{alphaString}
	}
	return true, result
}

func (d *TrieMultiWordDetector) parseASCII(alphaString string) (bool, []string) {
	length := len(alphaString)
	if length < d.MinLen {
		return false, []string{alphaString}
	}
	if length >= d.MaxLen {
		return false, []string{alphaString}
	}

	if d.getCount(alphaString) >= d.Threshold {
		return true, []string{alphaString}
	}
	if length < d.MinCheckLen {
		return false, []string{alphaString}
	}

	result := d.identifyMultiASCII(alphaString)
	if result == nil {
		return false, []string{alphaString}
	}
	return true, result
}

func (d *TrieMultiWordDetector) parseLowerASCII(alphaString string) (bool, []string) {
	length := len(alphaString)
	if length < d.MinLen || length >= d.MaxLen {
		return false, []string{alphaString}
	}
	if d.Counts[alphaString] >= d.Threshold {
		return true, []string{alphaString}
	}
	if length < d.MinCheckLen {
		return false, []string{alphaString}
	}

	result := d.identifyMultiLowerASCII(alphaString)
	if result == nil {
		return false, []string{alphaString}
	}
	return true, result
}
