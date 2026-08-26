package trainer

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync/atomic"
	"unicode/utf8"
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

// MergeFrom folds another generator's rune counts into this one. Counts are
// summed, so the resulting alphabet (sorted by count then rune) is independent
// of how the passwords were partitioned across workers.
func (ag *AlphabetGenerator) MergeFrom(other *AlphabetGenerator) {
	for r, c := range other.Counts {
		ag.Counts[r] += c
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
	cachedChar rune
	cachedCP   *cpData

	// keyspace-calculation scratch (populated by CalcOmenKeyspace only)
	ksTrans       []ksEdge
	ksLevelCounts [11]uint16
	ksCache       atomic.Pointer[keyspaceMemo]
}

type keyspaceMemo struct {
	values []atomic.Int64
}

// ksEdge is a precomputed transition used during keyspace calculation: the
// level cost of appending a letter and the destination context (nil if the
// resulting prefix is not in the grammar).
type ksEdge struct {
	level  int
	target *omenContext
}

type smoothedLNEntry struct {
	FinalLevel int

	PreNormLevel int

	OriginalCount int
}

type OmenTrainer struct {
	Alphabet   []rune
	alphaSet   map[rune]bool
	asciiAlpha [128]bool
	NGram      int
	MaxLen     int
	MinLen     int
	Grammar    map[string]*omenContext
	LNLookup   []int

	LNCounter int
	IPCounter int
	EPCounter int

	SmoothedLN []smoothedLNEntry

	// dimensions for the flat per-context keyspace cache
	ksLenDim   int
	ksLevelDim int

	// per-trainer scratch reused across Parse calls to avoid per-password
	// allocations. Each OmenTrainer is owned by a single goroutine.
	byteOffScratch []int
}

func NewOmenTrainer(alphabet []rune, ngram, maxLen int) *OmenTrainer {
	alphaSet := make(map[rune]bool, len(alphabet))
	var asciiAlpha [128]bool
	for _, r := range alphabet {
		alphaSet[r] = true
		if r >= 0 && r < utf8.RuneSelf {
			asciiAlpha[byte(r)] = true
		}
	}
	minLen := ngram
	if minLen < 1 {
		minLen = 1
	}
	return &OmenTrainer{
		Alphabet:   alphabet,
		alphaSet:   alphaSet,
		asciiAlpha: asciiAlpha,
		NGram:      ngram,
		MaxLen:     maxLen,
		MinLen:     minLen,
		Grammar:    make(map[string]*omenContext),
		LNLookup:   make([]int, maxLen),
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
	ascii := true
	for i := 0; i < len(password); i++ {
		if password[i] >= utf8.RuneSelf {
			ascii = false
			break
		}
	}
	if ascii {
		ot.parseASCII(password)
		return
	}

	// Build byte offsets of each rune start (plus final length) using reusable
	// scratch, so grammar keys can be taken as zero-copy substrings of the
	// still-live password rather than freshly allocated strings.
	offs := ot.byteOffScratch[:0]
	for i := range password {
		offs = append(offs, i)
	}
	offs = append(offs, len(password))
	ot.byteOffScratch = offs

	pwLen := len(offs) - 1
	if pwLen < ot.MinLen || pwLen > ot.MaxLen {
		return
	}

	ot.LNLookup[pwLen-1]++
	ot.LNCounter++

	prefixLen := ot.NGram - 1

	for i := 0; i <= pwLen-ot.NGram+1; i++ {
		prefix := password[offs[i]:offs[i+prefixLen]]

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
			endChar, _ := utf8.DecodeRuneInString(password[offs[i+prefixLen]:])
			cp := ctx.cachedCP
			if cp == nil || ctx.cachedChar != endChar {
				cp = ctx.NextLetter[endChar]
				if cp != nil {
					ctx.cachedChar = endChar
					ctx.cachedCP = cp
				}
			}
			if cp != nil {
				cp.Count++
				ctx.CPCount++
			} else if ot.alphaSet[endChar] {
				cp = &cpData{Count: 1}
				ctx.NextLetter[endChar] = cp
				ctx.cachedChar = endChar
				ctx.cachedCP = cp
				ctx.CPCount++
			}
		} else {
			ctx.EPCount++
			ot.EPCounter++
		}
	}
}

func (ot *OmenTrainer) parseASCII(password string) {
	pwLen := len(password)
	if pwLen < ot.MinLen || pwLen > ot.MaxLen {
		return
	}

	ot.LNLookup[pwLen-1]++
	ot.LNCounter++

	prefixLen := ot.NGram - 1
	for i := 0; i <= pwLen-ot.NGram+1; i++ {
		prefix := password[i : i+prefixLen]
		ctx, inGrammar := ot.Grammar[prefix]
		if !inGrammar {
			inAlphabet := true
			for j := 0; j < len(prefix); j++ {
				if !ot.asciiAlpha[prefix[j]] {
					inAlphabet = false
					break
				}
			}
			if !inAlphabet {
				continue
			}
			ctx = &omenContext{NextLetter: make(map[rune]*cpData)}
			ot.Grammar[prefix] = ctx
		}

		if i == 0 {
			ctx.IPCount++
			ot.IPCounter++
		}

		if i != pwLen-prefixLen {
			endChar := rune(password[i+prefixLen])
			cp := ctx.cachedCP
			if cp == nil || ctx.cachedChar != endChar {
				cp = ctx.NextLetter[endChar]
				if cp != nil {
					ctx.cachedChar = endChar
					ctx.cachedCP = cp
				}
			}
			if cp != nil {
				cp.Count++
				ctx.CPCount++
			} else if ot.asciiAlpha[byte(endChar)] {
				cp = &cpData{Count: 1}
				ctx.NextLetter[endChar] = cp
				ctx.cachedChar = endChar
				ctx.cachedCP = cp
				ctx.CPCount++
			}
		} else {
			ctx.EPCount++
			ot.EPCounter++
		}
	}
}

func (ot *OmenTrainer) ApplySmoothing() {
	contexts := make([]*omenContext, 0, len(ot.Grammar))
	for _, ctx := range ot.Grammar {
		contexts = append(contexts, ctx)
	}

	workers := runtime.GOMAXPROCS(0)
	if workers > len(contexts) {
		workers = len(contexts)
	}
	if workers > 0 {
		chunkSize := (len(contexts) + workers - 1) / workers
		tasks := make([]func() error, 0, workers)
		for worker := 0; worker < workers; worker++ {
			start := worker * chunkSize
			end := start + chunkSize
			if end > len(contexts) {
				end = len(contexts)
			}
			if start >= end {
				continue
			}
			tasks = append(tasks, func() error {
				for _, ctx := range contexts[start:end] {
					ctx.IPLevel = calcLevel(ctx.IPCount, ot.IPCounter, 250)
					ctx.EPLevel = calcLevel(ctx.EPCount, ot.EPCounter, 250)
					for _, cp := range ctx.NextLetter {
						cp.Level = calcLevel(cp.Count, ctx.CPCount, 2)
					}
				}
				return nil
			})
		}
		_ = runParallelTasks(tasks...)
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

const omenMaxKeyspaceLevel = 18

func packShortASCII(value string) (uint32, bool) {
	if len(value) == 0 || len(value) > 4 {
		return 0, false
	}
	var packed uint32
	for i := 0; i < len(value); i++ {
		if value[i] >= utf8.RuneSelf {
			return 0, false
		}
		packed = packed<<8 | uint32(value[i])
	}
	return packed, true
}

func nextPackedASCII(prefix uint32, char byte, prefixLen int) uint32 {
	if prefixLen == 1 {
		return uint32(char)
	}
	keepBits := uint((prefixLen - 1) * 8)
	mask := uint32(1<<keepBits) - 1
	return (prefix&mask)<<8 | uint32(char)
}

func sortKeyspaceEdges(edges []ksEdge) {
	for i := 1; i < len(edges); i++ {
		edge := edges[i]
		j := i
		for j > 0 && edges[j-1].level > edge.level {
			edges[j] = edges[j-1]
			j--
		}
		edges[j] = edge
	}
}

func prepareKeyspaceContext(
	ot *OmenTrainer,
	prefix string,
	ctx *omenContext,
	asciiGrammar map[uint32]*omenContext,
) {
	packedPrefix, prefixIsASCII := packShortASCII(prefix)
	prefixLen := ot.NGram - 1
	var suffix string
	suffixReady := false

	transitions := make([]ksEdge, 0, len(ctx.NextLetter))
	ctx.ksLevelCounts = [11]uint16{}
	for char, cp := range ctx.NextLetter {
		var target *omenContext
		if prefixIsASCII && char >= 0 && char < utf8.RuneSelf {
			target = asciiGrammar[nextPackedASCII(packedPrefix, byte(char), prefixLen)]
		} else {
			if !suffixReady {
				if prefixIsASCII {
					suffix = prefix[1:]
				} else {
					prefixRunes := []rune(prefix)
					suffix = string(prefixRunes[1:])
				}
				suffixReady = true
			}
			target = ot.Grammar[suffix+string(char)]
		}
		transitions = append(transitions, ksEdge{level: cp.Level, target: target})
		ctx.ksLevelCounts[cp.Level]++
	}
	sortKeyspaceEdges(transitions)
	ctx.ksTrans = transitions
	ctx.ksCache.Store(nil)
}

func CalcOmenKeyspace(ot *OmenTrainer) map[int]int64 {
	maxLevel := omenMaxKeyspaceLevel
	maxKeyspace := int64(10000000000)
	keyspace := make(map[int]int64)

	// Cache dimensions: length shrinks by 1 each recursion from a maximum of
	// (maxLen - ngram + 1); level is bounded by maxLevel.
	ot.ksLenDim = ot.MaxLen - ot.NGram + 2
	if ot.ksLenDim < 2 {
		ot.ksLenDim = 2
	}
	ot.ksLevelDim = maxLevel + 1

	ipKeys := make([]string, 0, len(ot.Grammar))
	asciiGrammar := make(map[uint32]*omenContext, len(ot.Grammar))
	for ip, ctx := range ot.Grammar {
		ipKeys = append(ipKeys, ip)
		if packed, ok := packShortASCII(ip); ok {
			asciiGrammar[packed] = ctx
		}
	}
	sort.Strings(ipKeys)
	if len(ipKeys) == 0 {
		return keyspace
	}

	workers := runtime.GOMAXPROCS(0)
	if workers > len(ipKeys) {
		workers = len(ipKeys)
	}
	chunkSize := (len(ipKeys) + workers - 1) / workers
	tasks := make([]func() error, 0, workers)
	for worker := 0; worker < workers; worker++ {
		start := worker * chunkSize
		end := start + chunkSize
		if end > len(ipKeys) {
			end = len(ipKeys)
		}
		if start >= end {
			continue
		}
		tasks = append(tasks, func() error {
			for _, prefix := range ipKeys[start:end] {
				prepareKeyspaceContext(ot, prefix, ot.Grammar[prefix], asciiGrammar)
			}
			return nil
		})
	}
	_ = runParallelTasks(tasks...)

	ipContexts := make([]*omenContext, len(ipKeys))
	for i, ip := range ipKeys {
		ipContexts[i] = ot.Grammar[ip]
	}
	lengthEntries := len(ot.SmoothedLN)
	contributions := make([]int64, len(ipKeys)*lengthEntries)

	for level := 1; level <= maxLevel; level++ {
		levelTasks := make([]func() error, 0, workers)
		for worker := 0; worker < workers; worker++ {
			start := worker * chunkSize
			end := start + chunkSize
			if end > len(ipContexts) {
				end = len(ipContexts)
			}
			if start >= end {
				continue
			}
			levelTasks = append(levelTasks, func() error {
				for ipIndex := start; ipIndex < end; ipIndex++ {
					ctx := ipContexts[ipIndex]
					levelMinusIP := level - ctx.IPLevel
					base := ipIndex * lengthEntries
					for idx, lnEntry := range ot.SmoothedLN {
						length := idx + 1
						if length > ot.NGram && levelMinusIP > 0 && lnEntry.FinalLevel <= levelMinusIP {
							contributions[base+idx] = recCalcKeyspace(
								ot,
								ctx,
								levelMinusIP-lnEntry.FinalLevel,
								length-ot.NGram+1,
							)
						} else {
							contributions[base+idx] = 0
						}
					}
				}
				return nil
			})
		}
		_ = runParallelTasks(levelTasks...)

		for ipIndex, ctx := range ipContexts {
			levelMinusIP := level - ctx.IPLevel
			if levelMinusIP <= 0 {
				continue
			}
			base := ipIndex * lengthEntries
			for idx, lnEntry := range ot.SmoothedLN {
				length := idx + 1
				if length <= ot.NGram || lnEntry.FinalLevel > levelMinusIP {
					continue
				}
				keyspace[level] += contributions[base+idx]
				if keyspace[level] > maxKeyspace {
					return keyspace
				}
			}
		}
		fmt.Printf("OMEN Keyspace for Level : %d : %d\n", level, keyspace[level])
	}

	return keyspace
}

func recCalcKeyspace(ot *OmenTrainer, ctx *omenContext, level, length int) int64 {
	if ctx == nil {
		return 0
	}

	// Fast path: flat per-context cache indexed by (length, level). The
	// recursion is a DAG (length strictly decreases), so a value is computed
	// exactly once and reused.
	if level >= 0 && level < ot.ksLevelDim && length >= 0 && length < ot.ksLenDim {
		memo := ctx.ksCache.Load()
		if memo == nil {
			candidate := &keyspaceMemo{
				values: make([]atomic.Int64, ot.ksLenDim*ot.ksLevelDim),
			}
			if ctx.ksCache.CompareAndSwap(nil, candidate) {
				memo = candidate
			} else {
				memo = ctx.ksCache.Load()
			}
		}
		cacheIdx := length*ot.ksLevelDim + level
		if cached := memo.values[cacheIdx].Load(); cached != 0 {
			return cached - 1
		}

		var sum int64
		if length == 1 {
			if level < len(ctx.ksLevelCounts) {
				sum = int64(ctx.ksLevelCounts[level])
			}
		} else {
			for _, e := range ctx.ksTrans {
				if e.level > level {
					break
				}
				sum += recCalcKeyspace(ot, e.target, level-e.level, length-1)
			}
		}

		memo.values[cacheIdx].Store(sum + 1)
		return sum
	}

	// Uncached fallback for out-of-range dimensions (should not occur in
	// practice); preserves identical semantics without memoization.
	var sum int64
	if length == 1 {
		if level >= 0 && level < len(ctx.ksLevelCounts) {
			sum = int64(ctx.ksLevelCounts[level])
		}
	} else {
		for _, e := range ctx.ksTrans {
			if e.level > level {
				break
			}
			sum += recCalcKeyspace(ot, e.target, level-e.level, length-1)
		}
	}
	return sum
}

func (ot *OmenTrainer) MergeFrom(other *OmenTrainer) {
	for prefix, oCtx := range other.Grammar {
		ctx, exists := ot.Grammar[prefix]
		if !exists {
			ot.Grammar[prefix] = oCtx
			continue
		}
		ctx.IPCount += oCtx.IPCount
		ctx.EPCount += oCtx.EPCount
		ctx.CPCount += oCtx.CPCount
		for char, oCP := range oCtx.NextLetter {
			if cp, ok := ctx.NextLetter[char]; ok {
				cp.Count += oCP.Count
			} else {
				ctx.NextLetter[char] = &cpData{Count: oCP.Count}
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
	ascii := true
	for i := 0; i < len(password); i++ {
		if password[i] >= utf8.RuneSelf {
			ascii = false
			break
		}
	}
	if ascii {
		return findOmenLevelASCII(ot, password)
	}

	// Stack-local byte offsets of rune starts (plus final length). Passwords
	// longer than MaxLen return -1, so a fixed buffer well above MaxLen is safe
	// and keeps this hot, concurrent path allocation-free.
	var offBuf [256]int
	n := 0
	for i := range password {
		if n >= len(offBuf)-1 {
			return -1
		}
		offBuf[n] = i
		n++
	}
	offBuf[n] = len(password)
	pwLen := n

	if pwLen < ot.MinLen || pwLen > ot.MaxLen {
		return -1
	}

	ngram := ot.NGram
	lnLevel := ot.SmoothedLN[pwLen-1].FinalLevel

	chunk := password[offBuf[0]:offBuf[ngram-1]]
	ctx, ok := ot.Grammar[chunk]
	if !ok {
		return -1
	}
	chainLevel := ctx.IPLevel

	for endPos := ngram; endPos <= pwLen; endPos++ {
		prefix := password[offBuf[endPos-ngram]:offBuf[endPos-1]]
		lastChar, _ := utf8.DecodeRuneInString(password[offBuf[endPos-1]:])

		pCtx, pOK := ot.Grammar[prefix]
		if !pOK {
			return -1
		}
		cp := pCtx.cachedCP
		if cp == nil || pCtx.cachedChar != lastChar {
			cp = pCtx.NextLetter[lastChar]
		}
		if cp == nil {
			return -1
		}
		chainLevel += cp.Level
	}

	return lnLevel + chainLevel
}

func findOmenLevelASCII(ot *OmenTrainer, password string) int {
	pwLen := len(password)
	if pwLen < ot.MinLen || pwLen > ot.MaxLen {
		return -1
	}

	ngram := ot.NGram
	lnLevel := ot.SmoothedLN[pwLen-1].FinalLevel
	ctx := ot.Grammar[password[:ngram-1]]
	if ctx == nil {
		return -1
	}
	chainLevel := ctx.IPLevel

	for endPos := ngram; endPos <= pwLen; endPos++ {
		prefixCtx := ot.Grammar[password[endPos-ngram:endPos-1]]
		if prefixCtx == nil {
			return -1
		}
		lastChar := rune(password[endPos-1])
		cp := prefixCtx.cachedCP
		if cp == nil || prefixCtx.cachedChar != lastChar {
			cp = prefixCtx.NextLetter[lastChar]
		}
		if cp == nil {
			return -1
		}
		chainLevel += cp.Level
	}
	return lnLevel + chainLevel
}

func SaveOmenRules(baseDir string, ot *OmenTrainer, omenKeyspace map[int]int64, omenLevels map[int]int, numValid int, info *ProgramInfo) error {
	omenDir := filepath.Join(baseDir, "Omen")
	return runParallelTasks(
		func() error { return writeIPLevel(omenDir, ot) },
		func() error { return writeEPLevel(omenDir, ot) },
		func() error { return writeCPLevel(omenDir, ot) },
		func() error { return writeLNLevel(omenDir, ot) },
		func() error { return writeOmenConfig(omenDir, info) },
		func() error { return writeAlphabet(omenDir, ot.Alphabet) },
		func() error { return writeOmenKeyspace(omenDir, omenKeyspace) },
		func() error { return writeOmenPWsPerLevel(omenDir, omenLevels) },
		func() error { return writeOmenProb(omenDir, omenKeyspace, omenLevels, numValid) },
	)
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
	w := bufio.NewWriterSize(f, 1<<20)
	buf := make([]byte, 0, 32)
	for key, ctx := range ot.Grammar {
		buf = buf[:0]
		buf = strconv.AppendInt(buf, int64(ctx.IPLevel), 10)
		buf = append(buf, '\t')
		buf = append(buf, key...)
		buf = append(buf, '\n')
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
	return w.Flush()
}

func writeEPLevel(dir string, ot *OmenTrainer) error {
	f, err := os.Create(filepath.Join(dir, "EP.level"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 1<<20)
	buf := make([]byte, 0, 32)
	for key, ctx := range ot.Grammar {
		buf = buf[:0]
		buf = strconv.AppendInt(buf, int64(ctx.EPLevel), 10)
		buf = append(buf, '\t')
		buf = append(buf, key...)
		buf = append(buf, '\n')
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
	return w.Flush()
}

func writeCPLevel(dir string, ot *OmenTrainer) error {
	f, err := os.Create(filepath.Join(dir, "CP.level"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 1<<20)
	buf := make([]byte, 0, 32)
	var charBuf [4]byte
	for prefix, ctx := range ot.Grammar {
		for char, cp := range ctx.NextLetter {
			buf = buf[:0]
			buf = strconv.AppendInt(buf, int64(cp.Level), 10)
			buf = append(buf, '\t')
			buf = append(buf, prefix...)
			n := utf8.EncodeRune(charBuf[:], char)
			buf = append(buf, charBuf[:n]...)
			buf = append(buf, '\n')
			if _, err := w.Write(buf); err != nil {
				return err
			}
		}
	}
	return w.Flush()
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
