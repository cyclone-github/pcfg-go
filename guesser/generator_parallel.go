package guesser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"

	"github.com/cyclone-github/pcfg-go/guesser/omen"
	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// errStop is returned by the output callback when context is cancelled (Ctrl+C)
var errStop = errors.New("stop")

const (
	ptChanSize     = 1024            // PT items buffered for workers
	outputChanSize = 10000           // batches buffered for writer
	batchSize      = 65536           // 64KB per batch
	writerBufSize  = 8 * 1024 * 1024 // 8MB bufio
)

// ParallelGuessGenerator uses goroutines to parallelize guess generation
// Architecture: popper (1) -> workers (N) -> writer (1)
// workers batch guesses into []byte to reduce channel traffic
type ParallelGuessGenerator struct {
	Grammar     pcfg.Grammar
	Base        []pcfg.BaseStructure
	Queue       *PcfgQueue
	Debug       bool
	OmenGrammar *omen.Grammar

	outputChan    chan []byte
	totalGuesses  atomic.Int64
	numParseTrees atomic.Int64
	probCoverage  atomic.Int64 // scaled by 1e15 for precision
	startTime     time.Time
}

// creates a generator that uses parallel workers
func NewParallelGuessGenerator(grammar pcfg.Grammar, base []pcfg.BaseStructure, omenGrammar *omen.Grammar, debug bool) *ParallelGuessGenerator {
	return &ParallelGuessGenerator{
		Grammar:     grammar,
		Base:        base,
		Queue:       NewPcfgQueue(grammar, base),
		Debug:       debug,
		OmenGrammar: omenGrammar,
		outputChan:  make(chan []byte, outputChanSize),
		startTime:   time.Now(),
	}
}

// creates a generator with a pre-built queue (for session restore)
func NewParallelGuessGeneratorWithQueue(grammar pcfg.Grammar, base []pcfg.BaseStructure, queue *PcfgQueue, omenGrammar *omen.Grammar, debug bool) *ParallelGuessGenerator {
	return &ParallelGuessGenerator{
		Grammar:     grammar,
		Base:        base,
		Queue:       queue,
		Debug:       debug,
		OmenGrammar: omenGrammar,
		outputChan:  make(chan []byte, outputChanSize),
		startTime:   time.Now(),
	}
}

// generates guesses using all CPU cores
func (g *ParallelGuessGenerator) RunParallel(limit int64) (int64, error) {
	return g.runParallelWithCtx(context.Background(), limit)
}

// runs with session save/load. On Ctrl+C, saves and exits gracefully
func (g *ParallelGuessGenerator) RunParallelWithSession(limit int64, savePath, ruleName, ruleUUID string, skipBrute, skipCase bool) (int64, error) {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "Saving session...")
		cancel()
	}()

	total, err := g.runParallelWithCtx(ctx, limit)

	// save session on exit (normal or signal)
	cfg := &SessionConfig{
		NumGuesses:     g.totalGuesses.Load(),
		NumParseTrees:  g.numParseTrees.Load(),
		ProbCoverage:   0,
		RunningTime:    int64(time.Since(g.startTime).Seconds()),
		MinProbability: g.Queue.MinProbability,
		MaxProbability: g.Queue.MaxProbability,
	}
	if saveErr := SaveSession(savePath, cfg, ruleName, ruleUUID, skipBrute, skipCase); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", saveErr)
	}
	return total, err
}

func (g *ParallelGuessGenerator) runParallelWithCtx(ctx context.Context, limit int64) (int64, error) {
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	writer := bufio.NewWriterSize(os.Stdout, writerBufSize)

	var wg sync.WaitGroup

	// writer goroutine: consume batches from outputChan
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer writer.Flush()
		for buf := range g.outputChan {
			writer.Write(buf)
		}
	}()

	// popper goroutine: pop PT items, send to workers
	ptChan := make(chan *pcfg.PTItem, ptChanSize)

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
				fmt.Fprintf(os.Stderr, "PT: %v Prob: %g\n", ptItem.PT, ptItem.Prob)
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

	// worker goroutines
	workerWg := sync.WaitGroup{}
	workerWg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func() {
			defer workerWg.Done()
			batch := make([]byte, 0, batchSize*2)
			output := func(guess string) error {
				if ctx.Err() != nil {
					return errStop
				}
				if limit > 0 {
					v := remaining.Add(-1)
					if v < 0 {
						return nil
					}
				}
				g.totalGuesses.Add(1)
				batch = append(batch, guess...)
				batch = append(batch, '\n')
				if len(batch) >= batchSize {
					out := make([]byte, len(batch))
					copy(out, batch)
					g.outputChan <- out
					batch = batch[:0]
				}
				return nil
			}
			for ptItem := range ptChan {
				g.createGuessesWithOutput("", ptItem.PT, 0, output)
			}
			if len(batch) > 0 {
				out := make([]byte, len(batch))
				copy(out, batch)
				g.outputChan <- out
			}
		}()
	}

	go func() {
		workerWg.Wait()
		close(g.outputChan)
	}()

	wg.Wait()
	return g.totalGuesses.Load(), nil
}

func (g *ParallelGuessGenerator) createGuessesWithOutput(curGuess string, pt []pcfg.PTNode, limit int64, output func(string) error) (int64, error) {
	return g.recursiveGuessesWithOutput(curGuess, pt, limit, output)
}

func (g *ParallelGuessGenerator) recursiveGuessesWithOutput(curGuess string, pt []pcfg.PTNode, limit int64, output func(string) error) (int64, error) {
	if len(pt) == 0 {
		return 0, nil
	}

	var numGuesses int64
	category := pt[0].Type[0]
	ptType := pt[0].Type
	idx := pt[0].Index

	entries := g.Grammar[ptType]
	if idx >= len(entries) {
		return 0, nil
	}

	switch category {
	case 'M':
		if g.OmenGrammar == nil {
			return 0, nil
		}
		levelStr := entries[idx].Values[0]
		level, err := strconv.Atoi(levelStr)
		if err != nil {
			return 0, nil
		}
		return g.omenGuessesWithOutput(curGuess, pt[1:], level, limit, output)

	case 'C':
		values := entries[idx].Values
		if len(values) == 0 {
			return 0, nil
		}
		maskLen := len([]rune(values[0]))
		guessRunes := []rune(curGuess)
		if maskLen > len(guessRunes) {
			return 0, nil
		}
		startWord := string(guessRunes[:len(guessRunes)-maskLen])
		endWord := guessRunes[len(guessRunes)-maskLen:]

		for _, mask := range values {
			maskRunes := []rune(mask)
			var newEnd strings.Builder
			for i, m := range maskRunes {
				if i >= len(endWord) {
					break
				}
				if m == 'L' {
					newEnd.WriteRune(endWord[i])
				} else {
					newEnd.WriteRune(unicode.ToUpper(endWord[i]))
				}
			}
			newGuess := startWord + newEnd.String()

			if len(pt) == 1 {
				output(newGuess)
				numGuesses++
			} else {
				n, _ := g.recursiveGuessesWithOutput(newGuess, pt[1:], limit, output)
				numGuesses += n
			}
		}

	default:
		for _, value := range entries[idx].Values {
			newGuess := curGuess + value
			if len(pt) == 1 {
				output(newGuess)
				numGuesses++
			} else {
				n, _ := g.recursiveGuessesWithOutput(newGuess, pt[1:], limit, output)
				numGuesses += n
			}
		}
	}

	return numGuesses, nil
}

func (g *ParallelGuessGenerator) omenGuessesWithOutput(curGuess string, ptRest []pcfg.PTNode, level int, limit int64, output func(string) error) (int64, error) {
	opt := omen.NewOptimizer(4)
	cracker := omen.NewMarkovCracker(g.OmenGrammar, level, opt)

	var numGuesses int64
	for {
		omenGuess := cracker.NextGuess()
		if omenGuess == "" {
			break
		}
		fullGuess := curGuess + omenGuess

		if len(ptRest) == 0 {
			if err := output(fullGuess); err != nil {
				return numGuesses, err
			}
			numGuesses++
		} else {
			n, err := g.recursiveGuessesWithOutput(fullGuess, ptRest, limit, output)
			numGuesses += n
			if err != nil {
				return numGuesses, err
			}
		}
	}
	return numGuesses, nil
}
