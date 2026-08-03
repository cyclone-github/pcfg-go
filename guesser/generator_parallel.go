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
	outputChanSize = 1024            // batches buffered for writer
	batchSize      = 32768           // 32KB per batch
	writerBufSize  = 1024 * 1024     // 1MB bufio
)

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
	probCoverage  atomic.Int64 // scaled by 1e15 for precision
	startTime     time.Time

	// from previous session when resuming with -l (accumulated stats)
	prevRunningTime      int64
	originalFirstStarted string // RFC3339, preserved when resuming
	autoSaveInterval     time.Duration
	savePath             string
	ruleName             string
	ruleUUID             string
	skipBrute            bool
	skipCase             bool
}

// creates a generator that uses parallel workers
func NewParallelGuessGenerator(grammar pcfg.Grammar, base []pcfg.BaseStructure, omenGrammar *omen.Grammar, debug bool) *ParallelGuessGenerator {
	return NewParallelGuessGeneratorWithLimit(grammar, base, omenGrammar, debug, defaultMaxQueueEntries)
}

func NewParallelGuessGeneratorWithLimit(grammar pcfg.Grammar, base []pcfg.BaseStructure, omenGrammar *omen.Grammar, debug bool, maxEntries int) *ParallelGuessGenerator {
	return &ParallelGuessGenerator{
		Base:              base,
		Queue:             NewPcfgQueueWithLimit(grammar, base, maxEntries),
		Debug:             debug,
		OmenGrammar:       omenGrammar,
		outputChan:        make(chan []byte, outputChanSize),
		startTime:         time.Now(),
		autoSaveInterval: 10 * time.Second,
	}
}

// creates a generator with a pre-built queue (for session restore)
func NewParallelGuessGeneratorWithQueue(grammar pcfg.Grammar, base []pcfg.BaseStructure, queue *PcfgQueue, omenGrammar *omen.Grammar, debug bool) *ParallelGuessGenerator {
	return &ParallelGuessGenerator{
		Base:        base,
		Queue:       queue,
		Debug:       debug,
		OmenGrammar: omenGrammar,
		outputChan:  make(chan []byte, outputChanSize),
		startTime:   time.Now(),
	}
}

// creates a generator with a pre-built queue and restores accumulated stats from a previous session
func (g *ParallelGuessGenerator) AutoSaveInterval(interval time.Duration) {
	if interval > 0 {
		g.autoSaveInterval = interval
	} else {
		g.autoSaveInterval = 10 * time.Second
	}
}

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
		autoSaveInterval:     10 * time.Second,
	}
	g.totalGuesses.Store(sav.NumGuesses)
	g.numParseTrees.Store(sav.NumParseTrees)
	return g
}

// generates guesses using all CPU cores
func (g *ParallelGuessGenerator) RunParallel(limit int64) (int64, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return g.runParallelWithCtx(ctx, limit, cancel)
}

// runs with session save/load, on Ctrl+C, saves and exits gracefully
// save runs on every exit path: normal completion, signal (SIGINT/SIGTERM), or panic
func (g *ParallelGuessGenerator) RunParallelWithSession(limit int64, savePath, ruleName, ruleUUID string, skipBrute, skipCase bool) (int64, error) {
	g.savePath = savePath
	g.ruleName = ruleName
	g.ruleUUID = ruleUUID
	g.skipBrute = skipBrute
	g.skipCase = skipCase
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

	// always save on exit: normal, signal, or panic. Works for first run and -l (load)
	defer func() {
		if saveErr := g.saveSession(savePath, ruleName, ruleUUID, skipBrute, skipCase); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", saveErr)
		} else {
			fmt.Fprintf(os.Stderr, "Session saved to %s\n", savePath)
		}
	}()

	return g.runParallelWithCtx(ctx, limit, cancel)
}

func (g *ParallelGuessGenerator) saveSession(savePath, ruleName, ruleUUID string, skipBrute, skipCase bool) error {
	currentRunTime := int64(time.Since(g.startTime).Seconds())
	cfg := &SessionConfig{
		NumGuesses:     g.totalGuesses.Load(),
		NumParseTrees:  g.numParseTrees.Load(),
		ProbCoverage:   0,
		RunningTime:    g.prevRunningTime + currentRunTime,
		MinProbability: g.Queue.MinProbability,
		MaxProbability: g.Queue.MaxProbability,
	}
	return SaveSession(savePath, cfg, ruleName, ruleUUID, skipBrute, skipCase, g.originalFirstStarted)
}

// stops the popper and workers (SIGINT/SIGTERM, broken pipe, or -n limit reached)
func (g *ParallelGuessGenerator) runParallelWithCtx(ctx context.Context, limit int64, cancelRun func()) (int64, error) {
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
			if _, err := writer.Write(buf); err != nil && cancelRun != nil {
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
			select {
			case <-ctx.Done():
				return
			default:
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
				continue
			}
			select {
			case ptChan <- ptItem:
			case <-ctx.Done():
				return
			}
		}
	}()

	// remaining: atomic counter for -n
	var remaining atomic.Int64
	remaining.Store(limit)

	if g.autoSaveInterval > 0 {
		go func() {
			ticker := time.NewTicker(g.autoSaveInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := g.saveSession(g.savePath, g.ruleName, g.ruleUUID, g.skipBrute, g.skipCase); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not autosave session: %v\n", err)
					}
				}
			}
		}()
	}

	// worker goroutines
	workerWg := sync.WaitGroup{}
	workerWg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func() {
			defer workerWg.Done()
			batch := make([]byte, 0, batchSize*2)
			// reuse one OMEN TMTO cache for all Markov PTs on this worker
			var omenOpt *omen.Optimizer
			if g.OmenGrammar != nil {
				omenOpt = omen.NewOptimizer(4)
			}
			// reusable prefix buffer for recursive guess building
			guessBuf := make([]byte, 0, 64)

			flushBatch := func() {
				if len(batch) == 0 {
					return
				}
				out := make([]byte, len(batch))
				copy(out, batch)
				g.outputChan <- out
				batch = batch[:0]
			}

			output := func(guess []byte) error {
				if ctx.Err() != nil {
					return errStop
				}
				if limit > 0 {
					v := remaining.Add(-1)
					if v < 0 {
						// stop popper/workers so batches flush; otherwise buffer only drains at queue end or SIGINT
						if cancelRun != nil {
							cancelRun()
						}
						return nil
					}
				}
				g.totalGuesses.Add(1)
				batch = append(batch, guess...)
				batch = append(batch, '\n')
				if len(batch) >= batchSize {
					flushBatch()
				}
				return nil
			}

			for ptItem := range ptChan {
				guessBuf = guessBuf[:0]
				g.recursiveGuesses(ig, guessBuf, ptItem.PT, output, omenOpt)
			}
			flushBatch()
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
		return g.omenGuesses(ig, curGuess, pt[1:], level, output, omenOpt)

	case 'C':
		values := entries[idx].Values
		if len(values) == 0 {
			return nil
		}
		maskLen := utf8.RuneCountInString(values[0])
		guessRunes := []rune(string(curGuess))
		if maskLen > len(guessRunes) {
			return nil
		}
		startWord := string(guessRunes[:len(guessRunes)-maskLen])
		endWord := guessRunes[len(guessRunes)-maskLen:]

		for _, mask := range values {
			// build mangled end into curGuess buffer: startWord + newEnd
			curGuess = append(curGuess[:0], startWord...)
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

			if len(pt) == 1 {
				if err := output(curGuess); err != nil {
					return err
				}
			} else {
				if err := g.recursiveGuesses(ig, curGuess, pt[1:], output, omenOpt); err != nil {
					return err
				}
			}
		}
		return nil

	default:
		baseLen := len(curGuess)
		for _, value := range entries[idx].Values {
			curGuess = append(curGuess[:baseLen], value...)
			if len(pt) == 1 {
				if err := output(curGuess); err != nil {
					return err
				}
			} else {
				if err := g.recursiveGuesses(ig, curGuess, pt[1:], output, omenOpt); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (g *ParallelGuessGenerator) omenGuesses(
	ig *IndexedGrammar,
	curGuess []byte,
	ptRest []packedNode,
	level int,
	output func([]byte) error,
	omenOpt *omen.Optimizer,
) error {
	cracker := omen.NewMarkovCracker(g.OmenGrammar, level, omenOpt)
	baseLen := len(curGuess)

	for {
		omenGuess := cracker.NextGuess()
		if omenGuess == "" {
			break
		}
		curGuess = append(curGuess[:baseLen], omenGuess...)

		if len(ptRest) == 0 {
			if err := output(curGuess); err != nil {
				return err
			}
		} else {
			if err := g.recursiveGuesses(ig, curGuess, ptRest, output, omenOpt); err != nil {
				return err
			}
		}
	}
	return nil
}
