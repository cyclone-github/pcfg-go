package guesser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// session save/load state
type SessionConfig struct {
	RuleName       string
	UUID           string
	SkipBrute      bool
	SkipCase       bool
	MinProbability float64
	MaxProbability float64
	NumGuesses     int64
	NumParseTrees  int64
	ProbCoverage   float64
	RunningTime    int64
	OmenGuessNum   int64
	FirstStarted   string // RFC3339, preserved when resuming
}

// read .sav file, return nil if file doesn't exist or is invalid
func LoadSession(path string) (*SessionConfig, error) {
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // file missing is not an error, caller starts fresh
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	cfg := &SessionConfig{}
	cfg.MaxProbability = 1.0
	cfg.MinProbability = 0.0

	scanner := bufio.NewScanner(f)
	var curSection string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			curSection = line[1 : len(line)-1]
			continue
		}
		if idx := strings.Index(line, "="); idx != -1 && curSection != "" {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			switch curSection {
			case "rule_info":
				switch key {
				case "rule_name":
					cfg.RuleName = val
				case "uuid":
					cfg.UUID = val
				case "skip_brute":
					cfg.SkipBrute = val == "True" || val == "true"
				case "skip_case":
					cfg.SkipCase = val == "True" || val == "true"
				}
			case "guessing_info":
				switch key {
				case "min_probability":
					cfg.MinProbability, _ = strconv.ParseFloat(val, 64)
				case "max_probability":
					cfg.MaxProbability, _ = strconv.ParseFloat(val, 64)
				case "omen_guess_number":
					cfg.OmenGuessNum, _ = strconv.ParseInt(val, 10, 64)
				}
			case "session_info":
				switch key {
				case "first_started":
					cfg.FirstStarted = val
				case "num_guesses":
					cfg.NumGuesses, _ = strconv.ParseInt(val, 10, 64)
				case "num_parse_trees":
					cfg.NumParseTrees, _ = strconv.ParseInt(val, 10, 64)
				case "probability_coverage":
					cfg.ProbCoverage, _ = strconv.ParseFloat(val, 64)
				case "running_time":
					cfg.RunningTime, _ = strconv.ParseInt(val, 10, 64)
				}
			}
		}
	}

	return cfg, scanner.Err()
}

// write .sav file. firstStarted is preserved from load; use "" for new sessions.
func SaveSession(path string, cfg *SessionConfig, ruleName, uuid string, skipBrute, skipCase bool, firstStarted string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	// rule_info
	fmt.Fprintln(w, "[rule_info]")
	fmt.Fprintf(w, "rule_name = %s\n", ruleName)
	fmt.Fprintf(w, "uuid = %s\n", uuid)
	fmt.Fprintf(w, "skip_brute = %v\n", skipBrute)
	fmt.Fprintf(w, "skip_case = %v\n", skipCase)
	fmt.Fprintln(w, "")

	// session_info
	fmt.Fprintln(w, "[session_info]")
	if firstStarted != "" {
		fmt.Fprintf(w, "first_started = %s\n", firstStarted)
	} else {
		fmt.Fprintf(w, "first_started = %s\n", time.Now().Format(time.RFC3339))
	}
	fmt.Fprintf(w, "last_updated = %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, "num_guesses = %d\n", cfg.NumGuesses)
	fmt.Fprintf(w, "num_parse_trees = %d\n", cfg.NumParseTrees)
	fmt.Fprintf(w, "probability_coverage = %g\n", cfg.ProbCoverage)
	fmt.Fprintf(w, "running_time = %d\n", cfg.RunningTime)
	fmt.Fprintln(w, "")

	// guessing_info
	fmt.Fprintln(w, "[guessing_info]")
	fmt.Fprintf(w, "mode = priority_queue\n")
	fmt.Fprintf(w, "min_probability = %g\n", cfg.MinProbability)
	fmt.Fprintf(w, "max_probability = %g\n", cfg.MaxProbability)
	if cfg.OmenGuessNum > 0 {
		fmt.Fprintf(w, "omen_guess_number = %d\n", cfg.OmenGuessNum)
	}

	return w.Flush()
}
