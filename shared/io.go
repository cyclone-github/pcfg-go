package pcfg

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const defaultBufSize = 4 * 1024 * 1024 // 4 MB

// returns a buffered scanner for the given file
func NewScanner(path string) (*bufio.Scanner, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening %s: %w", path, err)
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, defaultBufSize), defaultBufSize)
	return scanner, f, nil
}

// returns a buffered writer for the given file path (creates/truncates)
func NewWriter(path string) (*bufio.Writer, *os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("creating %s: %w", path, err)
	}
	w := bufio.NewWriterSize(f, defaultBufSize)
	return w, f, nil
}

// creates a directory (and parents) if it does not exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// stores a value and its probability for sorted output
type ProbPair struct {
	Value string
	Count int
}

// converts a map[string]int counter to a probability-sorted list
func CounterToProbs(counter map[string]int) []ProbEntry {
	if len(counter) == 0 {
		return nil
	}

	total := 0
	for _, c := range counter {
		total += c
	}

	pairs := make([]ProbEntry, 0, len(counter))
	for k, c := range counter {
		pairs = append(pairs, ProbEntry{Value: k, Prob: float64(c) / float64(total), Count: c})
	}

	// sort descending by count, then by value for stability
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Value < pairs[j].Value
	})

	return pairs
}

// stores a value, its probability, and count
type ProbEntry struct {
	Value string
	Prob  float64
	Count int
}

// writes probability entries to a file in "value\tprob\n" format
func WriteProbFile(path string, entries []ProbEntry) error {
	w, f, err := NewWriter(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range entries {
		_, err = fmt.Fprintf(w, "%s\t%s\n", e.Value, formatProb(e.Prob))
		if err != nil {
			return err
		}
	}
	return w.Flush()
}

func formatProb(p float64) string {
	return strconv.FormatFloat(p, 'g', -1, 64)
}

// removes all files in a directory (not subdirectories)
func CleanExistingFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// creates the directory structure for a pcfg rule
func CreateRuleFolders(baseDir string) error {
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "Masks"),
		filepath.Join(baseDir, "Prince"),
		filepath.Join(baseDir, "Grammar"),
		filepath.Join(baseDir, "Alpha"),
		filepath.Join(baseDir, "Capitalization"),
		filepath.Join(baseDir, "Digits"),
		filepath.Join(baseDir, "Years"),
		filepath.Join(baseDir, "Other"),
		filepath.Join(baseDir, "Context"),
		filepath.Join(baseDir, "Keyboard"),
		filepath.Join(baseDir, "Websites"),
		filepath.Join(baseDir, "Emails"),
		filepath.Join(baseDir, "Omen"),
	}
	for _, d := range dirs {
		if err := EnsureDir(d); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

// splits a tab-separated line into value and probability string
func ParseTSVLine(line string) (string, string, bool) {
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
