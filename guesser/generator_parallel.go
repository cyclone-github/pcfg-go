package guesser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/cyclone-github/pcfg-go/guesser/omen"
	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// errStop is returned by the output callback when context is cancelled (Ctrl+C)
var errStop = errors.New("stop")

const (
	ptChanSize     = 1024            // PT items buffered for workers
	outputChanSize = 256             // enough that workers rarely stall; RAM stays bounded
	batchSize      = 65536           // 64KB per batch
	batchCap       = batchSize * 2   // worker scratch / pooled buffer capacity
	writerBufSize  = 8 * 1024 * 1024 // 8MB bufio
)

// reused 64KB output batches to avoid a heap alloc+copy on every flush
var batchPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, batchCap)
		return b
	},
}

func getBatch() []byte {
	return batchPool.Get().([]byte)[:0]
}

func putBatch(b []byte) {
	if cap(b) == 0 || cap(b) > batchCap*4 {
		return
	}
	batchPool.Put(b[:0])
}

// worker-local scratch so capitalization never allocates per mask
type guessScratch struct {
	suffix []byte
	runes  []rune
}

func isASCIIBytes(b []byte) bool {
	for _, c := range b {
		if c >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func isASCIIString(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// ParallelGuessGenerator uses goroutines to parallelize guess generation
// Architecture: popper (1) -> workers (N) -> writer (1)
// workers batch guesses into []byte to reduce channel traffic
type ParallelGuessGenerator struct {
	Base        []pcfg.BaseStructure
	Queue       *PcfgQueue
	Debug       bool
	OmenGrammar *omen.Grammar

	outputChan    chan []byte
	totalGuesses  atomic.Int64
	numParseTrees atomic.Int64
	startTime     time.Time

	// from previous session when resuming with -l (accumulated stats)
	prevRunningTime      int64
	originalFirstStarted string // RFC3339, preserved when resuming

	autoPath string
}

func (g *ParallelGuessGenerator) applyAutoUpdate(batch autoBatch) {
	g.Queue.applyAutoBatch(batch)
	fmt.Fprintf(os.Stderr, "[auto] %d new founds analyzed, PCFG priorities updated\n", batch.n)
	if g.Queue.steer == nil {
		return
	}
	tops := g.Queue.steer.topBoosts(3)
	if len(tops) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "[auto] top boosts:")
	for i, t := range tops {
		if i > 0 {
			fmt.Fprint(os.Stderr, ",")
		}
		fmt.Fprintf(os.Stderr, " %s %.2fx", t.Key, t.Mult)
	}
	fmt.Fprintln(os.Stderr)
}

func (g *ParallelGuessGenerator) SetAuto(path string) {
	g.autoPath = path
}

// creates a generator that uses parallel workers
func NewParallelGuessGenerator(grammar pcfg.Grammar, base []pcfg.BaseStructure, omenGrammar *omen.Grammar, debug bool) *ParallelGuessGenerator {
	return &ParallelGuessGenerator{
		Base:        base,
		Queue:       NewPcfgQueue(grammar, base),
		Debug:       debug,
		OmenGrammar: omenGrammar,
		outputChan:  make(chan []byte, outputChanSize),
		startTime:   time.Now(),
	}
}

// creates a generator with a pre-built queue and restores accumulated stats from a previous session
func NewParallelGuessGeneratorWithQueueAndRestore(grammar pcfg.Grammar, base []pcfg.BaseStructure, queue *PcfgQueue, omenGrammar *omen.Grammar, debug bool, sav *SessionConfig) *ParallelGuessGenerator {
	g := &ParallelGuessGenerator{
		Base:                 base,
		Queue:                queue,
		Debug:                debug,
		OmenGrammar:          omenGrammar,
		outputChan:           make(chan []byte, outputChanSize),
		startTime:            time.Now(),
		prevRunningTime:      sav.RunningTime,
		originalFirstStarted: sav.FirstStarted,
	}
	g.totalGuesses.Store(sav.NumGuesses)
	g.numParseTrees.Store(sav.NumParseTrees)
	return g
}

// runs with session save/load, on Ctrl+C, saves and exits gracefully
// save runs on every exit path: normal completion, signal (SIGINT/SIGTERM), or panic
func (g *ParallelGuessGenerator) RunParallelWithSession(limit int64, savePath, ruleName, ruleUUID string, skipBrute, skipCase bool) (int64, error) {
	// Ignore SIGPIPE so piping to pv, head, etc. doesn't kill us before save on Ctrl+C
	signal.Ignore(syscall.SIGPIPE)

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "Saving session...")
		cancel()
	}()

	var autoCh <-chan autoBatch
	if g.autoPath != "" {
		g.Queue.steer = newAutoSteerer(g.Base)
		ch, err := startAutoWatcher(ctx, g.autoPath, g.Debug, autoPollInterval, autoMinFounds, newAutoParser(g.Queue.IndexedGrammar()).BaseStructureOf)
		if err != nil {
			cancel()
			return 0, err
		}
		autoCh = ch
		fmt.Fprintln(os.Stderr, "[auto] running")
	}

	// always save on exit: normal, signal, or panic. Works for first run and -l (load)
	defer func() {
		currentRunTime := int64(time.Since(g.startTime).Seconds())
		cfg := &SessionConfig{
			NumGuesses:     g.totalGuesses.Load(),
			NumParseTrees:  g.numParseTrees.Load(),
			ProbCoverage:   0,
			RunningTime:    g.prevRunningTime + currentRunTime,
			MinProbability: g.Queue.MinProbability,
			MaxProbability: g.Queue.MaxProbability,
		}
		if saveErr := SaveSession(savePath, cfg, ruleName, ruleUUID, skipBrute, skipCase, g.originalFirstStarted); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", saveErr)
		} else {
			fmt.Fprintf(os.Stderr, "Session saved to %s\n", savePath)
		}
	}()

	return g.runParallelWithCtx(ctx, limit, cancel, autoCh)
}

// stops the popper and workers (SIGINT/SIGTERM, broken pipe, or -n limit reached)
func (g *ParallelGuessGenerator) runParallelWithCtx(ctx context.Context, limit int64, cancelRun func(), autoCh <-chan autoBatch) (int64, error) {
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	ig := g.Queue.IndexedGrammar()
	writer := bufio.NewWriterSize(os.Stdout, writerBufSize)

	var wg sync.WaitGroup

	// writer goroutine: consume batches from outputChan
	// on broken pipe (e.g. pv exits), cancel so we save instead of spinning
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer writer.Flush()
		for buf := range g.outputChan {
			_, err := writer.Write(buf)
			putBatch(buf)
			if err != nil && cancelRun != nil {
				// broken pipe, reader exited - cancel to trigger save
				cancelRun()
			}
		}
	}()

	// popper goroutine: pop PT items, send to workers
	ptChan := make(chan *PTWork, ptChanSize)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ptChan)
		for {
			if autoCh != nil {
				select {
				case <-ctx.Done():
					return
				case batch := <-autoCh:
					g.applyAutoUpdate(batch)
				default:
				}
			} else {
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
			ptItem := g.Queue.Next()
			if ptItem == nil {
				return
			}
			g.numParseTrees.Add(1)
			if g.Debug {
				names := make([]string, len(ptItem.PT))
				for i, n := range ptItem.PT {
					names[i] = fmt.Sprintf("{%s %d}", ig.typeName(n.Type), n.Index)
				}
				fmt.Fprintf(os.Stderr, "PT: %v Prob: %g\n", names, ptItem.Prob)
				releasePTWork(ptItem)
				continue
			}
			select {
			case ptChan <- ptItem:
			case <-ctx.Done():
				releasePTWork(ptItem)
				return
			}
		}
	}()

	// remaining: atomic counter for -n
	var remaining atomic.Int64
	remaining.Store(limit)

	// worker goroutines
	workerWg := sync.WaitGroup{}
	workerWg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func() {
			defer workerWg.Done()
			batch := getBatch()
			// reuse one OMEN TMTO cache for all Markov PTs on this worker
			var omenOpt *omen.Optimizer
			if g.OmenGrammar != nil {
				omenOpt = omen.NewOptimizer(4)
			}
			// reusable prefix buffer for recursive guess building
			guessBuf := make([]byte, 0, 64)
			sc := &guessScratch{
				suffix: make([]byte, 0, 64),
				runes:  make([]rune, 0, 64),
			}

			flushBatch := func() {
				if len(batch) == 0 {
					return
				}
				g.outputChan <- batch
				batch = getBatch()
			}

			var localGuesses int64

			flushCounts := func() {
				if localGuesses != 0 {
					g.totalGuesses.Add(localGuesses)
					localGuesses = 0
				}
			}

			output := func(guess []byte) error {
				if limit > 0 {
					v := remaining.Add(-1)
					if v < 0 {
						// stop popper/workers so batches flush; otherwise buffer only drains at queue end or SIGINT
						if cancelRun != nil {
							cancelRun()
						}
						return errStop
					}
				}
				localGuesses++
				batch = append(batch, guess...)
				batch = append(batch, '\n')
				if len(batch) >= batchSize {
					flushBatch()
				}
				return nil
			}

			for ptItem := range ptChan {
				if ctx.Err() != nil {
					releasePTWork(ptItem)
					break
				}
				guessBuf = guessBuf[:0]
				g.recursiveGuesses(ig, guessBuf, ptItem.PT, output, omenOpt, sc, true)
				releasePTWork(ptItem)
			}
			flushBatch()
			flushCounts()
			putBatch(batch)
		}()
	}

	go func() {
		workerWg.Wait()
		close(g.outputChan)
	}()

	wg.Wait()
	return g.totalGuesses.Load(), nil
}

func (g *ParallelGuessGenerator) recursiveGuesses(
	ig *IndexedGrammar,
	curGuess []byte,
	pt []packedNode,
	output func([]byte) error,
	omenOpt *omen.Optimizer,
	sc *guessScratch,
	ascii bool,
) error {
	if len(pt) == 0 {
		return nil
	}

	node := pt[0]
	entries := ig.entries(node.Type)
	idx := int(node.Index)
	if idx >= len(entries) {
		return nil
	}

	switch ig.category(node.Type) {
	case 'M':
		if g.OmenGrammar == nil || omenOpt == nil {
			return nil
		}
		if !ig.hasMarkov || node.Type != ig.markovID || idx >= len(ig.markovLevel) {
			return nil
		}
		level := ig.markovLevel[idx]
		if level < 0 {
			// invalid omen level string (old code skipped on strconv.Atoi error)
			return nil
		}
		return g.omenGuesses(ig, curGuess, pt[1:], level, output, omenOpt, sc, ascii)

	case 'C':
		values := entries[idx].Values
		if len(values) == 0 {
			return nil
		}
		return g.applyCaseMasks(ig, curGuess, pt, values, output, omenOpt, sc, ascii)

	default:
		baseLen := len(curGuess)
		for _, value := range entries[idx].Values {
			curGuess = append(curGuess[:baseLen], value...)
			nextASCII := ascii && isASCIIString(value)
			if len(pt) == 1 {
				if err := output(curGuess); err != nil {
					return err
				}
			} else {
				if err := g.recursiveGuesses(ig, curGuess, pt[1:], output, omenOpt, sc, nextASCII); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (g *ParallelGuessGenerator) applyCaseMasks(
	ig *IndexedGrammar,
	curGuess []byte,
	pt []packedNode,
	values []string,
	output func([]byte) error,
	omenOpt *omen.Optimizer,
	sc *guessScratch,
	ascii bool,
) error {
	mask0 := values[0]
	emit := func(curGuess []byte, nextASCII bool) error {
		if len(pt) == 1 {
			return output(curGuess)
		}
		return g.recursiveGuesses(ig, curGuess, pt[1:], output, omenOpt, sc, nextASCII)
	}

	// fast path: password prefix and masks are ASCII (typical rockyou / English rules)
	if ascii && isASCIIString(mask0) {
		maskLen := len(mask0)
		if maskLen > len(curGuess) {
			return nil
		}
		prefixLen := len(curGuess) - maskLen
		sc.suffix = append(sc.suffix[:0], curGuess[prefixLen:]...)
		suffix := sc.suffix

		for _, mask := range values {
			curGuess = curGuess[:prefixLen]
			n := maskLen
			if len(mask) < n {
				n = len(mask)
			}
			if n > len(suffix) {
				n = len(suffix)
			}
			for i := 0; i < n; i++ {
				c := suffix[i]
				if mask[i] != 'L' && c >= 'a' && c <= 'z' {
					c -= 32
				}
				curGuess = append(curGuess, c)
			}
			if err := emit(curGuess, true); err != nil {
				return err
			}
		}
		return nil
	}

	maskLen := utf8.RuneCountInString(mask0)
	runes := sc.runes[:0]
	for i := 0; i < len(curGuess); {
		r, size := utf8.DecodeRune(curGuess[i:])
		runes = append(runes, r)
		i += size
	}
	sc.runes = runes
	if maskLen > len(runes) {
		return nil
	}

	prefixRunes := len(runes) - maskLen
	prefixLen := 0
	for i := 0; i < prefixRunes; i++ {
		prefixLen += utf8.RuneLen(runes[i])
	}
	endWord := runes[prefixRunes:]

	for _, mask := range values {
		curGuess = curGuess[:prefixLen]
		ri := 0
		for _, m := range mask {
			if ri >= len(endWord) {
				break
			}
			if m == 'L' {
				curGuess = utf8.AppendRune(curGuess, endWord[ri])
			} else {
				curGuess = utf8.AppendRune(curGuess, unicode.ToUpper(endWord[ri]))
			}
			ri++
		}
		if err := emit(curGuess, false); err != nil {
			return err
		}
	}
	return nil
}

func (g *ParallelGuessGenerator) omenGuesses(
	ig *IndexedGrammar,
	curGuess []byte,
	ptRest []packedNode,
	level int,
	output func([]byte) error,
	omenOpt *omen.Optimizer,
	sc *guessScratch,
	ascii bool,
) error {
	cracker := omen.NewMarkovCracker(g.OmenGrammar, level, omenOpt)
	baseLen := len(curGuess)

	for {
		var ok bool
		curGuess, ok = cracker.AppendNext(curGuess[:baseLen])
		if !ok {
			break
		}
		nextASCII := ascii && isASCIIBytes(curGuess[baseLen:])

		if len(ptRest) == 0 {
			if err := output(curGuess); err != nil {
				return err
			}
		} else {
			if err := g.recursiveGuesses(ig, curGuess, ptRest, output, omenOpt, sc, nextASCII); err != nil {
				return err
			}
		}
	}
	return nil
}
