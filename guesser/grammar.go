package guesser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// holds metadata about a loaded ruleset
type RulesetInfo struct {
	RuleName    string
	Version     string
	RuleVersion string
	Encoding    string
	UUID        string
}

// represents a parsed config.ini section
type ConfigSection struct {
	Name       string
	Directory  string
	Filenames  []string
	FileType   string
	InjectType string
	IsTerminal bool
}

// loads a complete grammar from disk
func LoadGrammar(baseDir, version string, skipBrute, skipCase bool) (pcfg.Grammar, []pcfg.BaseStructure, *RulesetInfo, error) {
	info := &RulesetInfo{Version: version}

	config, err := loadConfig(info, baseDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading config: %w", err)
	}

	grammar := make(pcfg.Grammar)

	if err := loadTerminals(info, grammar, baseDir, config, skipCase); err != nil {
		return nil, nil, nil, fmt.Errorf("loading terminals: %w", err)
	}

	var baseStructures []pcfg.BaseStructure
	if err := loadBaseStructures(&baseStructures, baseDir, skipBrute, "Grammar"); err != nil {
		return nil, nil, nil, fmt.Errorf("loading base structures: %w", err)
	}

	// add capitalization to base structures
	for i := range baseStructures {
		addCaseMangling(&baseStructures[i])
	}

	return grammar, baseStructures, info, nil
}

type iniConfig map[string]map[string]string

func loadConfig(info *RulesetInfo, baseDir string) (iniConfig, error) {
	path := filepath.Join(baseDir, "config.ini")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config.ini: %w", err)
	}
	defer f.Close()

	config := make(iniConfig)
	var currentSection string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			if _, exists := config[currentSection]; !exists {
				config[currentSection] = make(map[string]string)
			}
			continue
		}
		if idx := strings.Index(line, " = "); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+3:])
			config[currentSection][key] = value
		} else if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			config[currentSection][key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// extract key info
	if td, ok := config["TRAINING_PROGRAM_DETAILS"]; ok {
		info.RuleVersion = td["version"]
	}
	if dd, ok := config["TRAINING_DATASET_DETAILS"]; ok {
		info.Encoding = dd["encoding"]
		info.UUID = dd["uuid"]
	}

	return config, nil
}

func loadTerminals(info *RulesetInfo, grammar pcfg.Grammar, baseDir string, config iniConfig, skipCase bool) error {
	encoding := info.Encoding

	// Alpha
	if err := loadFromMultipleFiles(grammar, config["BASE_A"], baseDir, encoding); err != nil {
		return fmt.Errorf("alpha: %w", err)
	}

	// Capitalization
	if !skipCase {
		if err := loadFromMultipleFiles(grammar, config["CAPITALIZATION"], baseDir, encoding); err != nil {
			return fmt.Errorf("capitalization: %w", err)
		}
	} else {
		capConfig := config["CAPITALIZATION"]
		if capConfig != nil {
			var filenames []string
			if err := json.Unmarshal([]byte(capConfig["filenames"]), &filenames); err == nil {
				name := capConfig["name"]
				for _, file := range filenames {
					lenStr := strings.Split(file, ".")[0]
					length, _ := strconv.Atoi(lenStr)
					key := name + lenStr
					allLower := strings.Repeat("L", length)
					grammar[key] = []pcfg.GrammarEntry{{Values: []string{allLower}, Prob: 1.0}}
				}
			}
		}
	}

	// Digits
	if err := loadFromMultipleFiles(grammar, config["BASE_D"], baseDir, encoding); err != nil {
		return fmt.Errorf("digits: %w", err)
	}

	// Other
	if err := loadFromMultipleFiles(grammar, config["BASE_O"], baseDir, encoding); err != nil {
		return fmt.Errorf("other: %w", err)
	}

	// Keyboard
	if err := loadFromMultipleFiles(grammar, config["BASE_K"], baseDir, encoding); err != nil {
		return fmt.Errorf("keyboard: %w", err)
	}

	// Years
	if err := loadFromMultipleFiles(grammar, config["BASE_Y"], baseDir, encoding); err != nil {
		return fmt.Errorf("years: %w", err)
	}

	// Context
	if err := loadFromMultipleFiles(grammar, config["BASE_X"], baseDir, encoding); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	// OMEN probabilities
	omenPath := filepath.Join(baseDir, "Omen", "pcfg_omen_prob.txt")
	grammar["M"] = nil
	if entries, err := loadFromFile(omenPath); err == nil {
		grammar["M"] = entries
	}

	// Email providers
	emailPath := filepath.Join(baseDir, "Emails", "email_providers.txt")
	grammar["E"] = nil
	if entries, err := loadFromFile(emailPath); err == nil {
		grammar["E"] = entries
	}

	// Website hosts
	webPath := filepath.Join(baseDir, "Websites", "website_hosts.txt")
	grammar["W"] = nil
	if entries, err := loadFromFile(webPath); err == nil {
		grammar["W"] = entries
	}

	return nil
}

func loadFromMultipleFiles(grammar pcfg.Grammar, sectionConfig map[string]string, baseDir, encoding string) error {
	if sectionConfig == nil {
		return nil
	}

	directory := sectionConfig["directory"]
	name := sectionConfig["name"]

	var filenames []string
	if err := json.Unmarshal([]byte(sectionConfig["filenames"]), &filenames); err != nil {
		return fmt.Errorf("parsing filenames: %w", err)
	}

	for _, file := range filenames {
		fullPath := filepath.Join(baseDir, directory, file)
		key := name + strings.Split(file, ".")[0]

		entries, err := loadFromFile(fullPath)
		if err != nil {
			return fmt.Errorf("loading %s: %w", fullPath, err)
		}
		grammar[key] = entries
	}
	return nil
}

func loadFromFile(filename string) ([]pcfg.GrammarEntry, error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []pcfg.GrammarEntry
	prevProb := -1.0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		value := parts[0]
		prob, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		if prob == prevProb && len(entries) > 0 {
			entries[len(entries)-1].Values = append(entries[len(entries)-1].Values, value)
		} else {
			prevProb = prob
			entries = append(entries, pcfg.GrammarEntry{
				Values: []string{value},
				Prob:   prob,
			})
		}
	}

	return entries, scanner.Err()
}

func loadBaseStructures(bases *[]pcfg.BaseStructure, baseDir string, skipBrute bool, folder string) error {
	filename := filepath.Join(baseDir, folder, "grammar.txt")
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	totalProb := 1.0

	// first pass to find brute force probability if skip_brute
	if skipBrute {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			parts := strings.SplitN(scanner.Text(), "\t", 2)
			if len(parts) == 2 && parts[0] == "M" {
				prob, _ := strconv.ParseFloat(parts[1], 64)
				totalProb -= prob
			}
		}
		f.Seek(0, 0)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 2)
		if len(parts) != 2 {
			continue
		}

		value := parts[0]
		prob, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}
		prob /= totalProb

		base := pcfg.BaseStructure{
			Prob:         prob,
			Replacements: parseReplacements(value),
		}

		if !skipBrute || !containsMarkov(base.Replacements) {
			*bases = append(*bases, base)
		}
	}

	return scanner.Err()
}

func parseReplacements(value string) []string {
	var replacements []string
	for _, ch := range value {
		if ch >= 'A' && ch <= 'Z' {
			replacements = append(replacements, string(ch))
		} else {
			if len(replacements) > 0 {
				replacements[len(replacements)-1] += string(ch)
			}
		}
	}
	return replacements
}

func containsMarkov(replacements []string) bool {
	for _, r := range replacements {
		if r == "M" {
			return true
		}
	}
	return false
}

func addCaseMangling(base *pcfg.BaseStructure) {
	var newReplacements []string
	for _, r := range base.Replacements {
		if len(r) == 0 {
			continue
		}
		newReplacements = append(newReplacements, r)
		if r[0] == 'A' {
			lenStr := r[1:]
			newReplacements = append(newReplacements, "C"+lenStr)
		}
	}
	base.Replacements = newReplacements
}

// loads OMEN keyspace data
func LoadOmenKeyspace(baseDir string) map[int]int {
	filename := filepath.Join(baseDir, "Omen", "omen_keyspace.txt")
	keyspace := make(map[int]int)

	f, err := os.Open(filename)
	if err != nil {
		return keyspace
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 2)
		if len(parts) == 2 {
			level, _ := strconv.Atoi(parts[0])
			ks, _ := strconv.Atoi(parts[1])
			keyspace[level] = ks
		}
	}
	return keyspace
}
