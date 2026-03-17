package trainer

import (
	"runtime"
	"sync"

	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

type Pass2Result struct {
	Omen *OmenTrainer
	PCFG *PCFGParser
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

	chunkSize := (len(passwords) + numWorkers - 1) / numWorkers
	results := make([]Pass2Result, numWorkers)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(passwords) {
			end = len(passwords)
		}
		if start >= end {
			continue
		}
		chunk := passwords[start:end]
		wg.Add(1)
		go func(workerID int, chunk []string) {
			defer wg.Done()
			ot := NewOmenTrainer(alphabet, ngram, maxLen)
			pp := NewPCFGParser(mwd)
			for _, pw := range chunk {
				ot.Parse(pw)
				pp.Parse(pw)
			}
			results[workerID] = Pass2Result{Omen: ot, PCFG: pp}
		}(w, chunk)
	}

	wg.Wait()

	var mergedOmen *OmenTrainer
	var mergedPCFG *PCFGParser
	for i := 0; i < numWorkers; i++ {
		if results[i].Omen != nil || results[i].PCFG != nil {
			mergedOmen = results[i].Omen
			mergedPCFG = results[i].PCFG
			break
		}
	}
	if mergedOmen == nil {
		mergedOmen = NewOmenTrainer(alphabet, ngram, maxLen)
	}
	if mergedPCFG == nil {
		mergedPCFG = NewPCFGParser(mwd)
	}
	for i := 0; i < numWorkers; i++ {
		if results[i].Omen != nil && results[i].Omen != mergedOmen {
			mergedOmen.MergeFrom(results[i].Omen)
		}
		if results[i].PCFG != nil && results[i].PCFG != mergedPCFG {
			mergedPCFG.MergeFrom(results[i].PCFG)
		}
	}

	return mergedOmen, mergedPCFG, nil
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

	chunkSize := (len(passwords) + numWorkers - 1) / numWorkers
	workerLevels := make([]map[int]int, numWorkers)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(passwords) {
			end = len(passwords)
		}
		if start >= end {
			workerLevels[w] = make(map[int]int)
			continue
		}
		chunk := passwords[start:end]
		wg.Add(1)
		go func(workerID int, chunk []string) {
			defer wg.Done()
			levels := make(map[int]int)
			for _, pw := range chunk {
				level := FindOmenLevel(omenTrainer, pw)
				levels[level]++
			}
			workerLevels[workerID] = levels
		}(w, chunk)
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
