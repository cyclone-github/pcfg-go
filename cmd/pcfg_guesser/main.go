/*
   pcfg-go guesser
   Dev: cyclone
   URL: https://github.com/cyclone-github/
   Repo: https://github.com/cyclone-github/pcfg-go/
   Credits: https://github.com/lakiw/pcfg_cracker/
   Version: 0.5.0 (Go)
*/

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cyclone-github/pcfg-go/guesser"
	"github.com/cyclone-github/pcfg-go/guesser/omen"
)

const version = "0.5.0 (Go)"

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
		fmt.Fprintln(os.Stderr, "PCFG Guesser v0.5.0 (Go)")
		fmt.Fprintln(os.Stderr, "https://github.com/cyclone-github/pcfg-go/")
		os.Exit(0)
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
		omenGrammar, _ = omen.LoadGrammar(baseDir)
	}

	if *debug {
		fmt.Fprintf(os.Stderr, "Loaded ruleset: %s (version %s, encoding %s)\n",
			info.RuleName, info.RuleVersion, info.Encoding)
		fmt.Fprintf(os.Stderr, "Base structures: %d\n", len(base))
		fmt.Fprintf(os.Stderr, "Grammar entries: %d\n", len(g))
	}

	exeDir := filepath.Dir(exe)
	savePath := filepath.Join(exeDir, *session+".sav")

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
			queue := guesser.NewPcfgQueueFromSave(g, base, sav.MinProbability, sav.MaxProbability)
			gen = guesser.NewParallelGuessGeneratorWithQueue(g, base, queue, omenGrammar, *debug)
		} else {
			gen = guesser.NewParallelGuessGenerator(g, base, omenGrammar, *debug)
		}
	} else {
		gen = guesser.NewParallelGuessGenerator(g, base, omenGrammar, *debug)
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
