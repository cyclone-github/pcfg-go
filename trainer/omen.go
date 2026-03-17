package trainer

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

type AlphabetGenerator struct {
	MaxSize int
	NGram   int
	Counts  map[rune]int
}

func NewAlphabetGenerator(maxSize, ngram int) *AlphabetGenerator {
	return &AlphabetGenerator{
		MaxSize: maxSize,
		NGram:   ngram,
		Counts:  make(map[rune]int),
	}
}

func (ag *AlphabetGenerator) ProcessPassword(password string) {
	for _, r := range password {
		ag.Counts[r]++
	}
}

func (ag *AlphabetGenerator) GetAlphabet() []rune {
	type rc struct {
		R rune
		C int
	}
	pairs := make([]rc, 0, len(ag.Counts))
	for r, c := range ag.Counts {
		pairs = append(pairs, rc{r, c})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].C != pairs[j].C {
			return pairs[i].C > pairs[j].C
		}
		return pairs[i].R < pairs[j].R
	})
	size := ag.MaxSize
	if size > len(pairs) {
		size = len(pairs)
	}
	result := make([]rune, size)
	for i := 0; i < size; i++ {
		result[i] = pairs[i].R
	}
	return result
}

type cpData struct {
	Count int
	Level int
}

type omenContext struct {
	IPCount int
	IPLevel int

	EPCount int
	EPLevel int

	CPCount    int
	NextLetter map[rune]*cpData
	ksCache    map[int]map[int]int64
}

type smoothedLNEntry struct {
	FinalLevel int

	PreNormLevel int

	OriginalCount int
}

type OmenTrainer struct {
	Alphabet []rune
	alphaSet map[rune]bool
	NGram    int
	MaxLen   int
	MinLen   int
	Grammar  map[string]*omenContext
	LNLookup []int

	LNCounter int
	IPCounter int
	EPCounter int

	SmoothedLN []smoothedLNEntry
}

func NewOmenTrainer(alphabet []rune, ngram, maxLen int) *OmenTrainer {
	alphaSet := make(map[rune]bool, len(alphabet))
	for _, r := range alphabet {
		alphaSet[r] = true
	}
	minLen := ngram
	if minLen < 1 {
		minLen = 1
	}
	return &OmenTrainer{
		Alphabet: alphabet,
		alphaSet: alphaSet,
		NGram:    ngram,
		MaxLen:   maxLen,
		MinLen:   minLen,
		Grammar:  make(map[string]*omenContext),
		LNLookup: make([]int, maxLen),
	}
}

func (ot *OmenTrainer) isInAlphabet(s string) bool {
	for _, r := range s {
		if !ot.alphaSet[r] {
			return false
		}
	}
	return true
}

func (ot *OmenTrainer) Parse(password string) {
	runes := []rune(password)
	pwLen := len(runes)

	if pwLen < ot.MinLen || pwLen > ot.MaxLen {
		return
	}

	ot.LNLookup[pwLen-1]++
	ot.LNCounter++

	prefixLen := ot.NGram - 1

	for i := 0; i <= pwLen-ot.NGram+1; i++ {
		prefix := string(runes[i : i+prefixLen])

		ctx, inGrammar := ot.Grammar[prefix]
		if !inGrammar {
			if ot.isInAlphabet(prefix) {
				ctx = &omenContext{NextLetter: make(map[rune]*cpData)}
				ot.Grammar[prefix] = ctx
			} else {
				continue
			}
		}

		if i == 0 {
			ctx.IPCount++
			ot.IPCounter++
		}

		if i != pwLen-prefixLen {
			endChar := runes[i+prefixLen]
			if cp, exists := ctx.NextLetter[endChar]; exists {
				cp.Count++
				ctx.CPCount++
			} else if ot.alphaSet[endChar] {
				ctx.NextLetter[endChar] = &cpData{Count: 1}
				ctx.CPCount++
			}
		} else {
			ctx.EPCount++
			ot.EPCounter++
		}
	}
}

func (ot *OmenTrainer) ApplySmoothing() {
	for _, ctx := range ot.Grammar {
		ctx.IPLevel = calcLevel(ctx.IPCount, ot.IPCounter, 250)
		ctx.EPLevel = calcLevel(ctx.EPCount, ot.EPCounter, 250)
		for _, cp := range ctx.NextLetter {
			cp.Level = calcLevel(cp.Count, ctx.CPCount, 2)
		}
	}
	ot.smoothLength()
}

func (ot *OmenTrainer) smoothLength() {
	maxLevel := 10
	minLevel := maxLevel

	type firstPassEntry struct {
		level int
		count int
	}
	firstPass := make([]firstPassEntry, len(ot.LNLookup))

	for i, count := range ot.LNLookup {
		if ot.LNCounter == 0 {
			firstPass[i] = firstPassEntry{level: maxLevel, count: 0}
			continue
		}
		level := calcLevel(count, ot.LNCounter, 1)
		level += i

		if level < minLevel {
			minLevel = level
		}
		firstPass[i] = firstPassEntry{level: level, count: count}
	}

	ot.SmoothedLN = make([]smoothedLNEntry, len(ot.LNLookup))
	for i, entry := range firstPass {
		finalLevel := entry.level - minLevel
		if finalLevel > maxLevel {
			finalLevel = maxLevel
		}
		ot.SmoothedLN[i] = smoothedLNEntry{
			FinalLevel:    finalLevel,
			PreNormLevel:  entry.level,
			OriginalCount: entry.count,
		}
	}
}

func calcLevel(baseCount, totalCount int, adjustFactor float64) int {
	if totalCount == 0 {
		return 10
	}
	probi := float64(baseCount)/float64(totalCount)*adjustFactor + 1e-11
	level := int(math.Floor(-1 * math.Log(probi)))
	if level > 10 {
		level = 10
	}
	if level < 0 {
		level = 0
	}
	return level
}

func CalcOmenKeyspace(ot *OmenTrainer) map[int]int64 {
	maxLevel := 18
	maxKeyspace := int64(10000000000)
	keyspace := make(map[int]int64)

	ipKeys := make([]string, 0, len(ot.Grammar))
	for ip := range ot.Grammar {
		ipKeys = append(ipKeys, ip)
	}
	sort.Strings(ipKeys)

	for level := 1; level <= maxLevel; level++ {
		for _, ip := range ipKeys {
			ipInfo := ot.Grammar[ip]
			levelMinusIP := level - ipInfo.IPLevel
			if levelMinusIP <= 0 {
				continue
			}
			for idx, lnEntry := range ot.SmoothedLN {
				length := idx + 1
				if length <= ot.NGram {
					continue
				}
				if lnEntry.FinalLevel <= levelMinusIP {
					keyspace[level] += recCalcKeyspace(
						ot,
						levelMinusIP-lnEntry.FinalLevel,
						length-ot.NGram+1,
						ip,
					)
					if keyspace[level] > maxKeyspace {
						return keyspace
					}
				}
			}
		}
		fmt.Printf("OMEN Keyspace for Level : %d : %d\n", level, keyspace[level])
	}

	return keyspace
}

func recCalcKeyspace(ot *OmenTrainer, level, length int, ip string) int64 {
	ctx, ok := ot.Grammar[ip]
	if !ok {
		return 0
	}

	if ctx.ksCache == nil {
		ctx.ksCache = make(map[int]map[int]int64)
	}
	if ctx.ksCache[length] == nil {
		ctx.ksCache[length] = make(map[int]int64)
	}
	if v, found := ctx.ksCache[length][level]; found {
		return v
	}

	ctx.ksCache[length][level] = 0

	if length == 1 {
		for _, cp := range ctx.NextLetter {
			if cp.Level == level {
				ctx.ksCache[length][level]++
			}
		}
	} else {
		for ch, cp := range ctx.NextLetter {
			if cp.Level <= level {
				ipRunes := []rune(ip)
				nextIP := string(ipRunes[1:]) + string(ch)
				ctx.ksCache[length][level] += recCalcKeyspace(
					ot, level-cp.Level, length-1, nextIP,
				)
			}
		}
	}

	return ctx.ksCache[length][level]
}

func (ot *OmenTrainer) MergeFrom(other *OmenTrainer) {
	for prefix, oCtx := range other.Grammar {
		ctx, exists := ot.Grammar[prefix]
		if !exists {
			ctx = &omenContext{NextLetter: make(map[rune]*cpData)}
			ot.Grammar[prefix] = ctx
		}
		ctx.IPCount += oCtx.IPCount
		ctx.EPCount += oCtx.EPCount
		ctx.CPCount += oCtx.CPCount
		for ch, oCp := range oCtx.NextLetter {
			cp, ok := ctx.NextLetter[ch]
			if !ok {
				ctx.NextLetter[ch] = &cpData{Count: oCp.Count}
			} else {
				cp.Count += oCp.Count
			}
		}
	}
	for i := 0; i < len(ot.LNLookup) && i < len(other.LNLookup); i++ {
		ot.LNLookup[i] += other.LNLookup[i]
	}
	ot.LNCounter += other.LNCounter
	ot.IPCounter += other.IPCounter
	ot.EPCounter += other.EPCounter
}

func FindOmenLevel(ot *OmenTrainer, password string) int {
	runes := []rune(password)
	pwLen := len(runes)

	if pwLen < ot.MinLen || pwLen > ot.MaxLen {
		return -1
	}

	ngram := ot.NGram
	lnLevel := ot.SmoothedLN[pwLen-1].FinalLevel

	chunk := string(runes[0 : ngram-1])
	ctx, ok := ot.Grammar[chunk]
	if !ok {
		return -1
	}
	chainLevel := ctx.IPLevel

	for endPos := ngram; endPos <= pwLen; endPos++ {
		prefixRunes := runes[endPos-ngram : endPos-1]
		lastChar := runes[endPos-1]

		pCtx, pOK := ot.Grammar[string(prefixRunes)]
		if !pOK {
			return -1
		}
		cp, cpOK := pCtx.NextLetter[lastChar]
		if !cpOK {
			return -1
		}
		chainLevel += cp.Level
	}

	return lnLevel + chainLevel
}

func SaveOmenRules(baseDir string, ot *OmenTrainer, omenKeyspace map[int]int64, omenLevels map[int]int, numValid int, info *ProgramInfo) error {
	omenDir := filepath.Join(baseDir, "Omen")

	if err := writeIPLevel(omenDir, ot); err != nil {
		return err
	}
	if err := writeEPLevel(omenDir, ot); err != nil {
		return err
	}
	if err := writeCPLevel(omenDir, ot); err != nil {
		return err
	}
	if err := writeLNLevel(omenDir, ot); err != nil {
		return err
	}
	if err := writeOmenConfig(omenDir, info); err != nil {
		return err
	}
	if err := writeAlphabet(omenDir, ot.Alphabet); err != nil {
		return err
	}
	if err := writeOmenKeyspace(omenDir, omenKeyspace); err != nil {
		return err
	}
	if err := writeOmenPWsPerLevel(omenDir, omenLevels); err != nil {
		return err
	}
	if err := writeOmenProb(omenDir, omenKeyspace, omenLevels, numValid); err != nil {
		return err
	}

	return nil
}

func writeOmenConfig(dir string, info *ProgramInfo) error {
	f, err := os.Create(filepath.Join(dir, "config.txt"))
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "[training_settings]\n")
	fmt.Fprintf(f, "ngram = %d\n", info.NGram)
	fmt.Fprintf(f, "encoding = %s\n", info.Encoding)
	fmt.Fprintln(f)
	return nil
}

func writeAlphabet(dir string, alphabet []rune) error {
	f, err := os.Create(filepath.Join(dir, "alphabet.txt"))
	if err != nil {
		return err
	}
	defer f.Close()
	for _, r := range alphabet {
		fmt.Fprintf(f, "%c\n", r)
	}
	return nil
}

func writeIPLevel(dir string, ot *OmenTrainer) error {
	f, err := os.Create(filepath.Join(dir, "IP.level"))
	if err != nil {
		return err
	}
	defer f.Close()
	for key, ctx := range ot.Grammar {
		fmt.Fprintf(f, "%d\t%s\n", ctx.IPLevel, key)
	}
	return nil
}

func writeEPLevel(dir string, ot *OmenTrainer) error {
	f, err := os.Create(filepath.Join(dir, "EP.level"))
	if err != nil {
		return err
	}
	defer f.Close()
	for key, ctx := range ot.Grammar {
		fmt.Fprintf(f, "%d\t%s\n", ctx.EPLevel, key)
	}
	return nil
}

func writeCPLevel(dir string, ot *OmenTrainer) error {
	f, err := os.Create(filepath.Join(dir, "CP.level"))
	if err != nil {
		return err
	}
	defer f.Close()
	for prefix, ctx := range ot.Grammar {
		for char, cp := range ctx.NextLetter {
			fmt.Fprintf(f, "%d\t%s%s\n", cp.Level, prefix, string(char))
		}
	}
	return nil
}

func writeLNLevel(dir string, ot *OmenTrainer) error {
	f, err := os.Create(filepath.Join(dir, "LN.level"))
	if err != nil {
		return err
	}
	defer f.Close()
	for i, entry := range ot.SmoothedLN {
		fmt.Printf("PW Length %d : (%d, %d)\n", i+1, entry.PreNormLevel, entry.OriginalCount)
		fmt.Fprintf(f, "%d\n", entry.FinalLevel)
	}
	return nil
}

func writeOmenKeyspace(dir string, keyspace map[int]int64) error {
	f, err := os.Create(filepath.Join(dir, "omen_keyspace.txt"))
	if err != nil {
		return err
	}
	defer f.Close()

	type kv struct {
		Level    int
		Keyspace int64
	}
	var pairs []kv
	for l, ks := range keyspace {
		pairs = append(pairs, kv{l, ks})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Keyspace != pairs[j].Keyspace {
			return pairs[i].Keyspace < pairs[j].Keyspace
		}
		return pairs[i].Level < pairs[j].Level
	})
	for _, p := range pairs {
		fmt.Fprintf(f, "%d\t%d\n", p.Level, p.Keyspace)
	}
	return nil
}

func writeOmenPWsPerLevel(dir string, levels map[int]int) error {
	f, err := os.Create(filepath.Join(dir, "omen_pws_per_level.txt"))
	if err != nil {
		return err
	}
	defer f.Close()

	type kv struct {
		Level int
		Count int
	}
	var pairs []kv
	for l, c := range levels {
		pairs = append(pairs, kv{l, c})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Level < pairs[j].Level
	})
	for _, p := range pairs {
		fmt.Fprintf(f, "%d\t%d\n", p.Level, p.Count)
	}
	return nil
}

func writeOmenProb(dir string, keyspace map[int]int64, levels map[int]int, numValid int) error {
	f, err := os.Create(filepath.Join(dir, "pcfg_omen_prob.txt"))
	if err != nil {
		return err
	}
	defer f.Close()

	type kv struct {
		Level int
		Prob  float64
	}
	var pairs []kv
	for level, ks := range keyspace {
		if ks == 0 {
			continue
		}
		numInstances := levels[level]
		percentageCracked := float64(numInstances) / float64(numValid)
		prob := percentageCracked / float64(ks)
		pairs = append(pairs, kv{level, prob})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Prob != pairs[j].Prob {
			return pairs[i].Prob > pairs[j].Prob
		}
		return pairs[i].Level < pairs[j].Level
	})
	for _, p := range pairs {
		fmt.Fprintf(f, "%d\t%s\n", p.Level, strconv.FormatFloat(p.Prob, 'g', -1, 64))
	}
	return nil
}
