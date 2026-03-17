package trainer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// SavePCFGData saves all PCFG training data to disk.
func SavePCFGData(baseDir string, pcfgParser *PCFGParser, encoding string, saveSensitive bool) error {
	// Keyboard
	if err := saveLenIndexedCounters(filepath.Join(baseDir, "Keyboard"), pcfgParser.CountKeyboard); err != nil {
		return fmt.Errorf("saving keyboard data: %w", err)
	}

	// Emails
	emailDir := filepath.Join(baseDir, "Emails")
	if err := pcfg.CleanExistingFiles(emailDir); err != nil {
		return err
	}
	if err := saveCounter(filepath.Join(emailDir, "email_providers.txt"), pcfgParser.CountEmailProv); err != nil {
		return fmt.Errorf("saving email providers: %w", err)
	}
	if saveSensitive {
		if err := saveCounter(filepath.Join(emailDir, "full_emails.txt"), pcfgParser.CountEmails); err != nil {
			return fmt.Errorf("saving full emails: %w", err)
		}
	}

	// Websites
	webDir := filepath.Join(baseDir, "Websites")
	if err := pcfg.CleanExistingFiles(webDir); err != nil {
		return err
	}
	if err := saveCounter(filepath.Join(webDir, "website_hosts.txt"), pcfgParser.CountWebsiteHosts); err != nil {
		return fmt.Errorf("saving website hosts: %w", err)
	}
	if err := saveCounter(filepath.Join(webDir, "website_prefixes.txt"), pcfgParser.CountWebsitePfx); err != nil {
		return fmt.Errorf("saving website prefixes: %w", err)
	}
	if saveSensitive {
		if err := saveCounter(filepath.Join(webDir, "website_urls.txt"), pcfgParser.CountWebsiteURLs); err != nil {
			return fmt.Errorf("saving website urls: %w", err)
		}
	}

	// Years
	yearDir := filepath.Join(baseDir, "Years")
	if err := pcfg.CleanExistingFiles(yearDir); err != nil {
		return err
	}
	if err := saveCounter(filepath.Join(yearDir, "1.txt"), pcfgParser.CountYears); err != nil {
		return fmt.Errorf("saving years: %w", err)
	}

	// Context
	ctxDir := filepath.Join(baseDir, "Context")
	if err := pcfg.CleanExistingFiles(ctxDir); err != nil {
		return err
	}
	if err := saveCounter(filepath.Join(ctxDir, "1.txt"), pcfgParser.CountContext); err != nil {
		return fmt.Errorf("saving context: %w", err)
	}

	// Alpha
	if err := saveLenIndexedCounters(filepath.Join(baseDir, "Alpha"), pcfgParser.CountAlpha); err != nil {
		return fmt.Errorf("saving alpha: %w", err)
	}

	// Capitalization
	if err := saveLenIndexedCounters(filepath.Join(baseDir, "Capitalization"), pcfgParser.CountAlphaMasks); err != nil {
		return fmt.Errorf("saving capitalization: %w", err)
	}

	// Digits
	if err := saveLenIndexedCounters(filepath.Join(baseDir, "Digits"), pcfgParser.CountDigits); err != nil {
		return fmt.Errorf("saving digits: %w", err)
	}

	// Other
	if err := saveLenIndexedCounters(filepath.Join(baseDir, "Other"), pcfgParser.CountOther); err != nil {
		return fmt.Errorf("saving other: %w", err)
	}

	// Grammar (base structures)
	grammarDir := filepath.Join(baseDir, "Grammar")
	if err := pcfg.CleanExistingFiles(grammarDir); err != nil {
		return err
	}
	if err := saveCounter(filepath.Join(grammarDir, "grammar.txt"), pcfgParser.CountBaseStructs); err != nil {
		return fmt.Errorf("saving grammar: %w", err)
	}
	if err := saveCounter(filepath.Join(grammarDir, "raw_grammar.txt"), pcfgParser.CountRawBaseStructs); err != nil {
		return fmt.Errorf("saving raw grammar: %w", err)
	}

	// Prince
	princeDir := filepath.Join(baseDir, "Prince")
	if err := pcfg.CleanExistingFiles(princeDir); err != nil {
		return err
	}
	if err := saveCounter(filepath.Join(princeDir, "grammar.txt"), pcfgParser.CountPrince); err != nil {
		return fmt.Errorf("saving prince: %w", err)
	}

	return nil
}

func saveLenIndexedCounters(dir string, counters *LenIndexedCounters) error {
	if err := pcfg.CleanExistingFiles(dir); err != nil {
		return err
	}

	keys := counters.Keys()
	sort.Ints(keys)

	for _, length := range keys {
		c := counters.Get(length)
		if c == nil {
			continue
		}
		filename := filepath.Join(dir, fmt.Sprintf("%d.txt", length))
		if err := saveCounter(filename, c); err != nil {
			return err
		}
	}
	return nil
}

func saveCounter(filename string, counter *Counter) error {
	snap := counter.Snapshot()
	entries := pcfg.CounterToProbs(snap)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range entries {
		// Python outputs: str(value) + '\t' + str(probability) + '\n'
		// Python's str(float) uses repr-like output
		fmt.Fprintf(f, "%s\t%s\n", e.Value, formatFloat(e.Prob))
	}
	return nil
}

// formatFloat matches Python's str(float) output format.
// Python uses the shortest representation that uniquely identifies the value.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
