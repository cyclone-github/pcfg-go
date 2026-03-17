/*
   pcfg-go trainer
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
	"unicode/utf8"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
	"github.com/cyclone-github/pcfg-go/trainer"
	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

func printBanner() {
	fmt.Println()
	fmt.Println(`    ____            __  __           ______            __   `)
	fmt.Println(`   / __ \________  / /_/ /___  __   / ____/___  ____  / /  `)
	fmt.Println(`  / /_/ / ___/ _ \/ __/ __/ / / /  / /   / __ \/ __ \/ /   `)
	fmt.Println(` / ____/ /  /  __/ /_/ /_/ /_/ /  / /___/ /_/ / /_/ / /     `)
	fmt.Println(`/_/ __/_/_  \___/\__/\__/\__, /   \__________/\____/_/     `)
	fmt.Println(`   / ____/_  __________  __/_/_   / ____/_  _____  _____________  _____`)
	fmt.Println(`  / /_  / / / /_  /_  / / / / /  / / __/ / / / _ \/ ___/ ___/ _ \/ ___/`)
	fmt.Println(` / __/ / /_/ / / /_/ /_/ /_/ /  / /_/ / /_/ /  __(__  |__  )  __/ /    `)
	fmt.Println(`/_/____/__,_/ /___//__/\__, /   \____/\__,_/\___/____/____/\___/_/     `)
	fmt.Println(` /_  __/________ _(_)___ /_/_ _____        `)
	fmt.Println(`  / / / ___/ __ ` + "`" + `/ / __ \/ _ \/ ___/        `)
	fmt.Println(` / / / /  / /_/ / / / / /  __/ /            `)
	fmt.Println(`/_/ /_/   \__,_/_/_/ /_/\___/_/       `)
	fmt.Println()
}

func detectEncoding(filepath string) (string, float64) {
	f, err := os.Open(filepath)
	if err != nil {
		return "utf-8", 0.5
	}
	defer f.Close()

	buf := make([]byte, 64*1024)
	n, _ := f.Read(buf)
	buf = buf[:n]

	if n >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF {
		return "utf-8", 1.0
	}
	if utf8.Valid(buf) {
		return "utf-8", 0.99
	}
	return "utf-8", 0.5
}

func printStatistics(pcfgParser *trainer.PCFGParser) {
	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Top 5 e-mail providers")
	fmt.Println("-------------------------------------------------")
	fmt.Println()
	for _, e := range pcfgParser.CountEmailProv.TopN(5) {
		fmt.Printf("%s : %d\n", e.Key, e.Count)
	}

	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Top 5 URL domains")
	fmt.Println("-------------------------------------------------")
	fmt.Println()
	for _, e := range pcfgParser.CountWebsiteHosts.TopN(5) {
		fmt.Printf("%s : %d\n", e.Key, e.Count)
	}

	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Top 10 Years found")
	fmt.Println("-------------------------------------------------")
	fmt.Println()
	for _, e := range pcfgParser.CountYears.TopN(10) {
		fmt.Printf("%s : %d\n", e.Key, e.Count)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cycloneFlag := flag.Bool("cyclone", false, "trainer")
	versionFlag := flag.Bool("version", false, "Display version")
	helpFlag := flag.Bool("h", false, "Display help")

	info := &trainer.ProgramInfo{
		Name:         "PCFG Trainer",
		Version:      "0.5.0 (Go)",
		Author:       "cyclone",
		Contact:      "https://github.com/cyclone-github/",
		RuleName:     "Default",
		Encoding:     "",
		Comments:     "",
		NGram:        4,
		AlphabetSize: 100,
		Coverage:     0.6,
		MaxLen:       21,
	}

	rule := flag.String("r", info.RuleName, "Name of generated ruleset")
	training := flag.String("t", "", "Training set of passwords (required)")
	encoding := flag.String("e", info.Encoding, "File encoding")
	comments := flag.String("C", info.Comments, "Comments for config")
	saveSensitive := flag.Bool("S", false, "Save sensitive info like emails")
	prefixcount := flag.Bool("p", false, "Lines prefixed with occurrence count")
	ngram := flag.Int("n", info.NGram, "OMEN ngram size (2-5)")
	alphabetSize := flag.Int("a", info.AlphabetSize, "Alphabet size for Markov")
	coverage := flag.Float64("c", info.Coverage, "PCFG vs OMEN coverage (0.0-1.0)")
	multiword := flag.String("m", "", "Pre-train multiword file")

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
		fmt.Fprintln(os.Stderr, "PCFG Trainer v0.5.0 (Go)")
		fmt.Fprintln(os.Stderr, "https://github.com/cyclone-github/pcfg-go/")
		os.Exit(0)
	}

	if *training == "" {
		fmt.Fprintln(os.Stderr, "Error: -t (training) is required")
		flag.Usage()
		os.Exit(1)
	}

	info.RuleName = *rule
	info.TrainingFile = *training
	info.Encoding = *encoding
	info.Comments = *comments
	info.SaveSensitive = *saveSensitive
	info.PrefixCount = *prefixcount
	info.NGram = *ngram
	info.AlphabetSize = *alphabetSize
	info.Coverage = *coverage
	info.Multiword = *multiword

	printBanner()
	fmt.Println("Version:", info.Version)

	// encoding detection
	if info.Encoding == "" {
		fmt.Println()
		fmt.Println("-----------------------------------------------------------------")
		fmt.Println("Attempting to autodetect file encoding of the training passwords")
		fmt.Println("-----------------------------------------------------------------")
		detectedEnc, confidence := detectEncoding(info.TrainingFile)
		info.Encoding = detectedEnc
		fmt.Printf("File Encoding Detected: %s\n", detectedEnc)
		fmt.Printf("Confidence for file encoding: %.2f\n", confidence)
		fmt.Println("If you think another file encoding might have been used please")
		fmt.Println("manually specify the file encoding and run the training program again")
	}

	// determine output directory (Rules/<rule> next to executable)
	exe, _ := os.Executable()
	baseDir := filepath.Join(filepath.Dir(exe), "Rules", info.RuleName)

	if err := pcfg.CreateRuleFolders(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating rule folders: %v\n", err)
		os.Exit(1)
	}

	if err := runTrainer(info, baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "Training failed: %v\n", err)
		os.Exit(1)
	}
}

func runTrainer(info *trainer.ProgramInfo, baseDir string) error {
	// load passwords once
	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Loading training passwords (single read, bytes optimized)")
	fmt.Println("-------------------------------------------------")
	fmt.Println()

	passwords, numValid, numEncodingErr, duplicatesFound, err := trainer.LoadPasswordsToSlice(info.TrainingFile, info.PrefixCount)
	if err != nil {
		return fmt.Errorf("loading passwords: %w", err)
	}

	if numValid == 0 {
		fmt.Println()
		fmt.Println("Error, no valid passwords were found when attempting to train ruleset.")
		return fmt.Errorf("no valid passwords found")
	}

	fmt.Printf("Loaded %d valid passwords\n", numValid)
	fmt.Printf("Number of Encoding Errors Found in Training Set: %d\n", numEncodingErr)
	fmt.Println()

	if !duplicatesFound {
		fmt.Println()
		fmt.Println("WARNING:")
		fmt.Printf("   No duplicate passwords were detected in the first 100000 parsed passwords\n")
		fmt.Println()
		fmt.Println("    This may be a problem since the training program needs to know frequency")
		fmt.Println("    info such as '123456' being more common than '629811'")
		fmt.Println()
	}

	// pass 1 (sequential: alphabet + multiword)
	fmt.Println("-------------------------------------------------")
	fmt.Println("Performing the first pass (alphabet + multiword)")
	fmt.Println("-------------------------------------------------")
	fmt.Println()

	ag := trainer.NewAlphabetGenerator(info.AlphabetSize, info.NGram)
	mwd := parser.NewTrieMultiWordDetector(5, 4, 21)

	if info.Multiword != "" {
		fmt.Println("-------------------------------------------------")
		fmt.Println("Pretraining multiword detection.")
		fmt.Println("-------------------------------------------------")
		fmt.Println()
		mwInput := &trainer.FileInput{
			Filename: info.Multiword,
			Encoding: info.Encoding,
		}
		err := mwInput.ReadPasswords(func(pw string) {
			mwd.Train(pw, true)
		})
		if err != nil {
			return fmt.Errorf("multiword pre-training: %w", err)
		}
	}

	for i, pw := range passwords {
		if (i+1)%1000000 == 0 {
			fmt.Printf("%d Million\n", (i+1)/1000000)
		}
		ag.ProcessPassword(pw)
		mwd.Train(pw, false)
	}

	alphabet := ag.GetAlphabet()

	// pass 2 (parallel: OMEN + PCFG)
	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Performing the second pass (parallel: OMEN + PCFG)")
	fmt.Println("-------------------------------------------------")
	fmt.Println()

	omenTrainer, pcfgParser, err := trainer.RunPass2Parallel(passwords, alphabet, info.NGram, info.MaxLen, mwd, 0)
	if err != nil {
		return fmt.Errorf("second pass: %w", err)
	}

	// calculate OMEN probabilities
	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Calculating Markov (OMEN) probabilities and keyspace")
	fmt.Println("This may take a few minutes")
	fmt.Println("-------------------------------------------------")
	fmt.Println()

	omenTrainer.ApplySmoothing()
	omenKeyspace := trainer.CalcOmenKeyspace(omenTrainer)

	// pass 3 (parallel: OMEN level distribution)
	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Performing third pass (parallel: OMEN level distribution)")
	fmt.Println("-------------------------------------------------")
	fmt.Println()

	omenLevels := trainer.RunPass3Parallel(passwords, omenTrainer, 0)

	// print statistics
	printStatistics(pcfgParser)

	// insert OMEN/Markov into base structures
	if info.Coverage != 1 {
		if !hasKeyspace(omenKeyspace) {
			fmt.Println("Error. The trainer was unable to create any Markov/OMEN NGrams for some reason")
			fmt.Println("If you want to re-try this without using Markov/OMEN, rerun the trainer with")
			fmt.Println("the argument '--coverage 1'")
			fmt.Println("Exiting without saving grammar")
			return fmt.Errorf("no OMEN keyspace generated")
		}

		if info.Coverage == 0 {
			pcfgParser.CountBaseStructs = trainer.NewCounter()
			pcfgParser.CountBaseStructs.Inc("M")
		} else {
			markovInstances := int((float64(numValid) / info.Coverage) - float64(numValid))
			pcfgParser.CountBaseStructs.Add("M", markovInstances)
		}
	}

	// save
	fmt.Println()
	fmt.Println("-------------------------------------------------")
	fmt.Println("Saving Data")
	fmt.Println("-------------------------------------------------")
	fmt.Println()

	fileInput := &trainer.FileInput{
		NumPasswords:            numValid,
		NumEncodingErr:          numEncodingErr,
		DuplicatesFound:         duplicatesFound,
		NumToCheckForDuplicates: 100000,
	}

	if err := trainer.SaveConfigFile(baseDir, info, fileInput, pcfgParser); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if err := trainer.SaveOmenRules(baseDir, omenTrainer, omenKeyspace, omenLevels, numValid, info); err != nil {
		return fmt.Errorf("saving OMEN rules: %w", err)
	}

	if err := trainer.SavePCFGData(baseDir, pcfgParser, info.Encoding, info.SaveSensitive); err != nil {
		return fmt.Errorf("saving PCFG data: %w", err)
	}

	fmt.Println()
	fmt.Println("Training completed successfully!")

	return nil
}

func hasKeyspace(ks map[int]int64) bool {
	for _, v := range ks {
		if v > 0 {
			return true
		}
	}
	return false
}
