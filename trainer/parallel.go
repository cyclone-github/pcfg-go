package trainer

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

type Pass2Result struct {
	Omen *OmenTrainer
	PCFG *PCFGParser
}

func runParallelTasks(tasks ...func() error) error {
	errors := make([]error, len(tasks))
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for i, task := range tasks {
		go func(index int, task func() error) {
			defer wg.Done()
			errors[index] = task()
		}(i, task)
	}
	wg.Wait()

	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

// print "N Million" whenever a worker's batch crosses a million-password boundary
func reportMillions(processed *atomic.Int64, n int) {
	if n <= 0 {
		return
	}
	prev := processed.Add(int64(n))
	old := prev - int64(n)
	for m := old/1000000 + 1; m <= prev/1000000; m++ {
		fmt.Printf("%d Million\n", m)
	}
}

// RunPass1Parallel computes the alphabet (rune frequencies) and trains the
// multiword detector across all passwords in parallel. Each worker accumulates
// into private instances that are merged (additively) into the shared ag/mwd
// afterwards, producing results identical to sequential processing.
func RunPass1Parallel(passwords []string, ag *AlphabetGenerator, mwd *parser.TrieMultiWordDetector, numWorkers int) {
	if len(passwords) == 0 {
		return
	}
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}
	if numWorkers > len(passwords) {
		numWorkers = len(passwords)
	}

	chunkSize := (len(passwords) + numWorkers - 1) / numWorkers
	ags := make([]*AlphabetGenerator, numWorkers)
	mwds := make([]*parser.TrieMultiWordDetector, numWorkers)
	const batchSize = 8192
	var nextBatch atomic.Int64
	var processed atomic.Int64
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			lag := NewAlphabetGenerator(ag.MaxSize, ag.NGram)
			lmwd := parser.NewTrieMultiWordDetector(mwd.Threshold, mwd.MinLen, mwd.MaxLen)
			lmwd.Counts = make(map[string]int, chunkSize/2)
			for {
				start := int(nextBatch.Add(batchSize)) - batchSize
				if start >= len(passwords) {
					break
				}
				end := start + batchSize
				if end > len(passwords) {
					end = len(passwords)
				}
				for _, pw := range passwords[start:end] {
					lag.ProcessPassword(pw)
					lmwd.Train(pw, false)
				}
				reportMillions(&processed, end-start)
			}
			ags[workerID] = lag
			mwds[workerID] = lmwd
		}(w)
	}

	wg.Wait()

	if len(mwd.Counts) == 0 {
		totalEntries := 0
		for _, worker := range mwds {
			if worker != nil {
				totalEntries += len(worker.Counts)
			}
		}
		mwd.Counts = make(map[string]int, totalEntries)
	}

	for w := 0; w < numWorkers; w++ {
		if ags[w] != nil {
			ag.MergeFrom(ags[w])
		}
		if mwds[w] != nil {
			mwd.MergeFrom(mwds[w])
		}
	}
}

func RunPass2Parallel(passwords []string, alphabet []rune, ngram, maxLen int, mwd *parser.TrieMultiWordDetector, numWorkers int) (*OmenTrainer, *PCFGParser, error) {
	if len(passwords) == 0 {
		return NewOmenTrainer(alphabet, ngram, maxLen), NewPCFGParser(mwd), nil
	}
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}
	if numWorkers > len(passwords) {
		numWorkers = len(passwords)
	}

	results := make([]Pass2Result, numWorkers)
	const batchSize = 4096
	var nextBatch atomic.Int64
	var processed atomic.Int64
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ot := NewOmenTrainer(alphabet, ngram, maxLen)
			pp := NewPCFGParser(mwd)
			for {
				start := int(nextBatch.Add(batchSize)) - batchSize
				if start >= len(passwords) {
					break
				}
				end := start + batchSize
				if end > len(passwords) {
					end = len(passwords)
				}
				for _, pw := range passwords[start:end] {
					ot.Parse(pw)
					pp.Parse(pw)
				}
				reportMillions(&processed, end-start)
			}
			results[workerID] = Pass2Result{Omen: ot, PCFG: pp}
		}(w)
	}

	wg.Wait()

	active := make([]Pass2Result, 0, numWorkers)
	for i := 0; i < numWorkers; i++ {
		if results[i].Omen != nil && results[i].PCFG != nil {
			active = append(active, results[i])
		}
	}
	if len(active) == 0 {
		return NewOmenTrainer(alphabet, ngram, maxLen), NewPCFGParser(mwd), nil
	}

	for len(active) > 1 {
		pairs := len(active) / 2
		next := make([]Pass2Result, (len(active)+1)/2)
		var mergeWG sync.WaitGroup
		mergeWG.Add(pairs)
		for pair := 0; pair < pairs; pair++ {
			left := active[pair*2]
			right := active[pair*2+1]
			go func(index int, left, right Pass2Result) {
				defer mergeWG.Done()
				left.Omen.MergeFrom(right.Omen)
				left.PCFG.MergeFrom(right.PCFG)
				next[index] = left
			}(pair, left, right)
		}
		if len(active)%2 != 0 {
			next[len(next)-1] = active[len(active)-1]
		}
		mergeWG.Wait()
		active = next
	}

	return active[0].Omen, active[0].PCFG, nil
}

func RunPass3Parallel(passwords []string, omenTrainer *OmenTrainer, numWorkers int) map[int]int {
	if len(passwords) == 0 {
		return make(map[int]int)
	}
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}
	if numWorkers > len(passwords) {
		numWorkers = len(passwords)
	}

	workerLevels := make([]map[int]int, numWorkers)
	const batchSize = 8192
	var nextBatch atomic.Int64
	var processed atomic.Int64
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			levels := make(map[int]int)
			for {
				start := int(nextBatch.Add(batchSize)) - batchSize
				if start >= len(passwords) {
					break
				}
				end := start + batchSize
				if end > len(passwords) {
					end = len(passwords)
				}
				for _, pw := range passwords[start:end] {
					level := FindOmenLevel(omenTrainer, pw)
					levels[level]++
				}
				reportMillions(&processed, end-start)
			}
			workerLevels[workerID] = levels
		}(w)
	}

	wg.Wait()

	merged := make(map[int]int)
	for _, levels := range workerLevels {
		for level, count := range levels {
			merged[level] += count
		}
	}
	return merged
}
