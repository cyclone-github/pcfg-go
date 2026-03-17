package trainer

import (
	"strings"

	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

type PCFGParser struct {
	MultiwordDetector *parser.TrieMultiWordDetector

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

	sectionList, foundWalks, _ := parser.DetectKeyboardWalk(password)
	for _, walk := range foundWalks {
		p.CountKeyboard.Inc(len([]rune(walk)), walk)
	}

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

	sectionList, years := parser.YearDetection(sectionList)
	for _, y := range years {
		p.CountYears.Inc(y)
	}

	sectionList, csStrings := parser.ContextSensitiveDetection(sectionList)
	for _, cs := range csStrings {
		p.CountContext.Inc(cs)
	}

	sectionList, alphas, masks := parser.AlphaDetection(sectionList, p.MultiwordDetector)
	for _, a := range alphas {
		lowerA := strings.ToLower(a)
		p.CountAlpha.Inc(len([]rune(lowerA)), lowerA)
	}
	for _, m := range masks {
		p.CountAlphaMasks.Inc(len([]rune(m)), m)
	}

	sectionList, digits := parser.DigitDetection(sectionList)
	for _, d := range digits {
		p.CountDigits.Inc(len([]rune(d)), d)
	}

	sectionList, others := parser.OtherDetection(sectionList)
	for _, o := range others {
		p.CountOther.Inc(len([]rune(o)), o)
	}

	for _, section := range sectionList {
		if section.Type != "" {
			p.CountPrince.Inc(section.Type)
		}
	}

	isSupported, baseStruct := parser.BaseStructureCreation(sectionList)
	if isSupported {
		p.CountBaseStructs.Inc(baseStruct)
	}
	p.CountRawBaseStructs.Inc(baseStruct)
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
