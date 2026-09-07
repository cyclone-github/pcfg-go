/*
   pcfg-go guesser
   Dev: cyclone
   URL: https://github.com/cyclone-github/
   Repo: https://github.com/cyclone-github/pcfg-go/
   Credits: https://github.com/lakiw/pcfg_cracker/
*/

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cyclone-github/pcfg-go/guesser"
	"github.com/cyclone-github/pcfg-go/guesser/omen"
)

const version = "0.6.1-dev.20260907-1815 (Go)"

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cycloneFlag := flag.Bool("cyclone", false, "pcfg_guesser")
	versionFlag := flag.Bool("version", false, "Display version")
	helpFlag := flag.Bool("h", false, "Display help")

	rule := flag.String("r", "Default", "Ruleset name under Rules/")
	session := flag.String("s", "default_run", "Session name for save/restore")
	loadSession := flag.Bool("l", false, "Load previous session")
	limit := flag.Int64("n", 0, "Max number of guesses (0 = unlimited)")
	skipBrute := flag.Bool("b", false, "Skip OMEN/Markov guesses")
	allLower := flag.Bool("a", false, "No case mangling")
	debug := flag.Bool("d", false, "Debug output instead of guesses")
	adaptiveFile := flag.String("adaptive", "", "Adaptive PCFG steering using recovered founds")

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}
	if *cycloneFlag {
		codedBy := "Q29kZWQgYnkgY3ljbG9uZSA7KQo="
		decoded, _ := base64.StdEncoding.DecodeString(codedBy)
		fmt.Fprintln(os.Stderr, string(decoded))
		os.Exit(0)
	}
	if *versionFlag {
		fmt.Fprintf(os.Stderr, "PCFG Guesser v%s\n", version)
		fmt.Fprintln(os.Stderr, "https://github.com/cyclone-github/pcfg-go/")
		os.Exit(0)
	}

	if flag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "Error: unexpected positional argument: %s\n", flag.Arg(0))
		flag.Usage()
		os.Exit(1)
	}
	if *rule == "" {
		fmt.Fprintln(os.Stderr, "Error: -r (rule) cannot be empty")
		os.Exit(1)
	}
	if *session == "" {
		fmt.Fprintln(os.Stderr, "Error: -s (session) cannot be empty")
		os.Exit(1)
	}
	if *limit < 0 {
		fmt.Fprintln(os.Stderr, "Error: -n (limit) must be 0 or greater")
		os.Exit(1)
	}
	if *adaptiveFile != "" {
		if st, err := os.Stat(*adaptiveFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot open -adaptive founds file: %v\n", err)
			os.Exit(1)
		} else if st.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: -adaptive path is a directory: %s\n", *adaptiveFile)
			os.Exit(1)
		}
		f, err := os.Open(*adaptiveFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot read -adaptive founds file: %v\n", err)
			os.Exit(1)
		}
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot read -adaptive founds file: %v\n", err)
			os.Exit(1)
		}
	}

	exe, _ := os.Executable()
	baseDir := filepath.Join(filepath.Dir(exe), "Rules", *rule)

	// check if directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		altDir := filepath.Join(cwd, "Rules", *rule)
		if _, err := os.Stat(altDir); err == nil {
			baseDir = altDir
		} else {
			fmt.Fprintf(os.Stderr, "Error: Grammar directory not found: %s\n", baseDir)
			os.Exit(1)
		}
	}

	g, base, info, err := guesser.LoadGrammar(baseDir, version, *skipBrute, *allLower)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading grammar: %v\n", err)
		os.Exit(1)
	}
	info.RuleName = *rule

	var omenGrammar *omen.Grammar
	if !*skipBrute {
		omenGrammar, err = omen.LoadGrammar(baseDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading OMEN grammar: %v\n", err)
			os.Exit(1)
		}
	}

	if *debug {
		fmt.Fprintf(os.Stderr, "Loaded ruleset: %s (version %s, encoding %s)\n",
			info.RuleName, info.RuleVersion, info.Encoding)
		fmt.Fprintf(os.Stderr, "Base structures: %d\n", len(base))
		fmt.Fprintf(os.Stderr, "Grammar entries: %d\n", len(g))
	}

	exeDir := filepath.Dir(exe)
	savePath := filepath.Join(exeDir, *session+".sav")
	// Fallback to cwd if exe is in a temp dir (e.g. go run) or exe dir isn't writable
	if cwd, err := os.Getwd(); err == nil {
		useCwd := strings.HasPrefix(exeDir, "/tmp") || strings.Contains(exeDir, "go-build")
		if !useCwd {
			// test writability without truncating existing session file
			testPath := savePath + ".writetest"
			if f, err := os.Create(testPath); err != nil {
				useCwd = true
			} else {
				f.Close()
				os.Remove(testPath)
			}
		}
		if useCwd {
			savePath = filepath.Join(cwd, *session+".sav")
		}
	}

	var gen *guesser.ParallelGuessGenerator
	if *loadSession {
		sav, loadErr := guesser.LoadSession(savePath)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "Error loading session: %v\n", loadErr)
			os.Exit(1)
		}
		if sav != nil {
			if sav.UUID != "" && info.UUID != "" && sav.UUID != info.UUID {
				fmt.Fprintf(os.Stderr, "Error: Session ruleset UUID mismatch (ruleset was retrained)\n")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Restoring saved progress...")
			fmt.Fprintln(os.Stderr, "Note: Restore may take a long time for sessions that ran for hours or days.")
			queue := guesser.NewPcfgQueueFromSave(g, base, sav.MinProbability, sav.MaxProbability)
			gen = guesser.NewParallelGuessGeneratorWithQueueAndRestore(g, base, queue, omenGrammar, *debug, sav)
		} else {
			gen = guesser.NewParallelGuessGenerator(g, base, omenGrammar, *debug)
		}
	} else {
		gen = guesser.NewParallelGuessGenerator(g, base, omenGrammar, *debug)
	}

	if *adaptiveFile != "" {
		gen.SetAdaptive(*adaptiveFile)
	}

	totalGuesses, err := gen.RunParallelWithSession(*limit, savePath, info.RuleName, info.UUID, *skipBrute, *allLower)
	if err != nil {
		if !isPipeError(err) {
			fmt.Fprintf(os.Stderr, "Error during guess generation: %v\n", err)
		}
	}

	if *debug {
		fmt.Fprintf(os.Stderr, "Total guesses generated: %d\n", totalGuesses)
	}
}

func isPipeError(err error) bool {
	return err != nil && (err.Error() == "write /dev/stdout: broken pipe" ||
		err.Error() == "short write")
}
