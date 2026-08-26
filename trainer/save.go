package trainer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// SavePCFGData saves all PCFG training data to disk.
func SavePCFGData(baseDir string, pcfgParser *PCFGParser, encoding string, saveSensitive bool) error {
	return runParallelTasks(
		func() error {
			if err := saveLenIndexedCounters(filepath.Join(baseDir, "Keyboard"), pcfgParser.CountKeyboard); err != nil {
				return fmt.Errorf("saving keyboard data: %w", err)
			}
			return nil
		},
		func() error {
			dir := filepath.Join(baseDir, "Emails")
			if err := pcfg.CleanExistingFiles(dir); err != nil {
				return err
			}
			if err := saveCounter(filepath.Join(dir, "email_providers.txt"), pcfgParser.CountEmailProv); err != nil {
				return fmt.Errorf("saving email providers: %w", err)
			}
			if saveSensitive {
				if err := saveCounter(filepath.Join(dir, "full_emails.txt"), pcfgParser.CountEmails); err != nil {
					return fmt.Errorf("saving full emails: %w", err)
				}
			}
			return nil
		},
		func() error {
			dir := filepath.Join(baseDir, "Websites")
			if err := pcfg.CleanExistingFiles(dir); err != nil {
				return err
			}
			if err := saveCounter(filepath.Join(dir, "website_hosts.txt"), pcfgParser.CountWebsiteHosts); err != nil {
				return fmt.Errorf("saving website hosts: %w", err)
			}
			if err := saveCounter(filepath.Join(dir, "website_prefixes.txt"), pcfgParser.CountWebsitePfx); err != nil {
				return fmt.Errorf("saving website prefixes: %w", err)
			}
			if saveSensitive {
				if err := saveCounter(filepath.Join(dir, "website_urls.txt"), pcfgParser.CountWebsiteURLs); err != nil {
					return fmt.Errorf("saving website urls: %w", err)
				}
			}
			return nil
		},
		func() error {
			dir := filepath.Join(baseDir, "Years")
			if err := pcfg.CleanExistingFiles(dir); err != nil {
				return err
			}
			if err := saveCounter(filepath.Join(dir, "1.txt"), pcfgParser.CountYears); err != nil {
				return fmt.Errorf("saving years: %w", err)
			}
			return nil
		},
		func() error {
			dir := filepath.Join(baseDir, "Context")
			if err := pcfg.CleanExistingFiles(dir); err != nil {
				return err
			}
			if err := saveCounter(filepath.Join(dir, "1.txt"), pcfgParser.CountContext); err != nil {
				return fmt.Errorf("saving context: %w", err)
			}
			return nil
		},
		func() error {
			if err := saveLenIndexedCounters(filepath.Join(baseDir, "Alpha"), pcfgParser.CountAlpha); err != nil {
				return fmt.Errorf("saving alpha: %w", err)
			}
			return nil
		},
		func() error {
			if err := saveLenIndexedCounters(filepath.Join(baseDir, "Capitalization"), pcfgParser.CountAlphaMasks); err != nil {
				return fmt.Errorf("saving capitalization: %w", err)
			}
			return nil
		},
		func() error {
			if err := saveLenIndexedCounters(filepath.Join(baseDir, "Digits"), pcfgParser.CountDigits); err != nil {
				return fmt.Errorf("saving digits: %w", err)
			}
			return nil
		},
		func() error {
			if err := saveLenIndexedCounters(filepath.Join(baseDir, "Other"), pcfgParser.CountOther); err != nil {
				return fmt.Errorf("saving other: %w", err)
			}
			return nil
		},
		func() error {
			dir := filepath.Join(baseDir, "Grammar")
			if err := pcfg.CleanExistingFiles(dir); err != nil {
				return err
			}
			if err := saveCounter(filepath.Join(dir, "grammar.txt"), pcfgParser.CountBaseStructs); err != nil {
				return fmt.Errorf("saving grammar: %w", err)
			}
			if err := saveCounter(filepath.Join(dir, "raw_grammar.txt"), pcfgParser.CountRawBaseStructs); err != nil {
				return fmt.Errorf("saving raw grammar: %w", err)
			}
			return nil
		},
		func() error {
			dir := filepath.Join(baseDir, "Prince")
			if err := pcfg.CleanExistingFiles(dir); err != nil {
				return err
			}
			if err := saveCounter(filepath.Join(dir, "grammar.txt"), pcfgParser.CountPrince); err != nil {
				return fmt.Errorf("saving prince: %w", err)
			}
			return nil
		},
	)
}

func saveLenIndexedCounters(dir string, counters *LenIndexedCounters) error {
	if err := pcfg.CleanExistingFiles(dir); err != nil {
		return err
	}

	keys := counters.Keys()
	sort.Ints(keys)

	tasks := make([]func() error, 0, len(keys))
	for _, length := range keys {
		c := counters.Get(length)
		if c == nil {
			continue
		}
		filename := filepath.Join(dir, strconv.Itoa(length)+".txt")
		tasks = append(tasks, func() error {
			return saveCounter(filename, c)
		})
	}
	return runParallelTasks(tasks...)
}

func saveCounter(filename string, counter *Counter) error {
	snap := counter.Snapshot()
	entries := pcfg.CounterToProbs(snap)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 1<<20)
	buf := make([]byte, 0, 96)
	for _, e := range entries {
		// Python outputs: str(value) + '\t' + str(probability) + '\n'
		// Python's str(float) uses repr-like output
		buf = buf[:0]
		buf = append(buf, e.Value...)
		buf = append(buf, '\t')
		buf = strconv.AppendFloat(buf, e.Prob, 'g', -1, 64)
		buf = append(buf, '\n')
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
	return w.Flush()
}
