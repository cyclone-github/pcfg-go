package trainer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

const cachedCapitalizationMaskLength = 128

var (
	lowerMasks [cachedCapitalizationMaskLength + 1]string
	upperMasks [cachedCapitalizationMaskLength + 1]string
	titleMasks [cachedCapitalizationMaskLength + 1]string
)

func init() {
	for length := 1; length <= cachedCapitalizationMaskLength; length++ {
		lowerMasks[length] = strings.Repeat("L", length)
		upperMasks[length] = strings.Repeat("U", length)
		titleMasks[length] = "U" + lowerMasks[length-1]
	}
}

func capitalizationMask(value string, length int) string {
	if length <= cachedCapitalizationMaskLength {
		var mask [cachedCapitalizationMaskLength]byte
		upperCount := 0
		index := 0
		for _, r := range value {
			if unicode.IsUpper(r) {
				mask[index] = 'U'
				upperCount++
			} else {
				mask[index] = 'L'
			}
			index++
		}
		switch {
		case upperCount == 0:
			return lowerMasks[length]
		case upperCount == length:
			return upperMasks[length]
		case upperCount == 1 && mask[0] == 'U':
			return titleMasks[length]
		default:
			return string(mask[:length])
		}
	}

	var mask strings.Builder
	mask.Grow(length)
	for _, r := range value {
		if unicode.IsUpper(r) {
			mask.WriteByte('U')
		} else {
			mask.WriteByte('L')
		}
	}
	return mask.String()
}

type PCFGParser struct {
	MultiwordDetector  *parser.TrieMultiWordDetector
	sectionScratch     []parser.Section
	replacementScratch []parser.Section

	CountKeyboard       *LenIndexedCounters
	CountEmails         *Counter
	CountEmailProv      *Counter
	CountWebsiteURLs    *Counter
	CountWebsiteHosts   *Counter
	CountWebsitePfx     *Counter
	CountYears          *Counter
	CountContext        *Counter
	CountAlpha          *LenIndexedCounters
	CountAlphaMasks     *LenIndexedCounters
	CountDigits         *LenIndexedCounters
	CountOther          *LenIndexedCounters
	CountBaseStructs    *Counter
	CountRawBaseStructs *Counter
	CountPrince         *Counter
}

func NewPCFGParser(mwd *parser.TrieMultiWordDetector) *PCFGParser {
	return &PCFGParser{
		MultiwordDetector:   mwd,
		CountKeyboard:       NewLenIndexedCounters(),
		CountEmails:         NewCounter(),
		CountEmailProv:      NewCounter(),
		CountWebsiteURLs:    NewCounter(),
		CountWebsiteHosts:   NewCounter(),
		CountWebsitePfx:     NewCounter(),
		CountYears:          NewCounter(),
		CountContext:        NewCounter(),
		CountAlpha:          NewLenIndexedCounters(),
		CountAlphaMasks:     NewLenIndexedCounters(),
		CountDigits:         NewLenIndexedCounters(),
		CountOther:          NewLenIndexedCounters(),
		CountBaseStructs:    NewCounter(),
		CountRawBaseStructs: NewCounter(),
		CountPrince:         NewCounter(),
	}
}

func (p *PCFGParser) Parse(password string) {
	sectionList := parser.DetectKeyboardWalk(password, p.sectionScratch)

	sectionList, emails, providers := parser.EmailDetection(sectionList)
	for _, e := range emails {
		p.CountEmails.Inc(e)
	}
	for _, pr := range providers {
		p.CountEmailProv.Inc(pr)
	}

	sectionList, urls, hosts, prefixes := parser.WebsiteDetection(sectionList)
	for _, u := range urls {
		p.CountWebsiteURLs.Inc(u)
	}
	for _, h := range hosts {
		p.CountWebsiteHosts.Inc(h)
	}
	for _, pf := range prefixes {
		if pf != "" {
			p.CountWebsitePfx.Inc(pf)
		}
	}

	sectionList = parser.YearDetection(sectionList)
	sectionList = parser.ContextSensitiveDetection(sectionList)
	sectionList, p.replacementScratch = parser.AlphaDetection(
		sectionList,
		p.MultiwordDetector,
		p.replacementScratch,
	)
	sectionList, p.replacementScratch = parser.DigitDetection(sectionList, p.replacementScratch)
	sectionList = parser.OtherDetection(sectionList)

	for _, section := range sectionList {
		sectionType := section.Type
		if sectionType == "" {
			continue
		}

		p.CountPrince.Inc(sectionType)
		switch sectionType[0] {
		case 'K':
			p.CountKeyboard.Inc(utf8.RuneCountInString(section.Value), section.Value)
		case 'Y':
			p.CountYears.Inc(section.Value)
		case 'X':
			p.CountContext.Inc(section.Value)
		case 'A':
			lower := strings.ToLower(section.Value)
			length := utf8.RuneCountInString(lower)
			p.CountAlpha.Inc(length, lower)
			p.CountAlphaMasks.Inc(length, capitalizationMask(section.Value, length))
		case 'D':
			p.CountDigits.Inc(utf8.RuneCountInString(section.Value), section.Value)
		case 'O':
			p.CountOther.Inc(utf8.RuneCountInString(section.Value), section.Value)
		}
	}

	isSupported, baseStruct := parser.BaseStructureCreation(sectionList)
	if isSupported {
		p.CountBaseStructs.Inc(baseStruct)
	}
	p.CountRawBaseStructs.Inc(baseStruct)
	p.sectionScratch = sectionList
}

func (p *PCFGParser) MergeFrom(other *PCFGParser) {
	p.CountKeyboard.MergeFrom(other.CountKeyboard)
	p.CountEmails.MergeFrom(other.CountEmails)
	p.CountEmailProv.MergeFrom(other.CountEmailProv)
	p.CountWebsiteURLs.MergeFrom(other.CountWebsiteURLs)
	p.CountWebsiteHosts.MergeFrom(other.CountWebsiteHosts)
	p.CountWebsitePfx.MergeFrom(other.CountWebsitePfx)
	p.CountYears.MergeFrom(other.CountYears)
	p.CountContext.MergeFrom(other.CountContext)
	p.CountAlpha.MergeFrom(other.CountAlpha)
	p.CountAlphaMasks.MergeFrom(other.CountAlphaMasks)
	p.CountDigits.MergeFrom(other.CountDigits)
	p.CountOther.MergeFrom(other.CountOther)
	p.CountBaseStructs.MergeFrom(other.CountBaseStructs)
	p.CountRawBaseStructs.MergeFrom(other.CountRawBaseStructs)
	p.CountPrince.MergeFrom(other.CountPrince)
}
