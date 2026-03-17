package omen

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Grammar struct {
	NGram    int
	MaxLevel int

	IP map[int][]string

	EP map[string]int

	CP map[string]map[int][]string

	LN map[int][]int
}

func LoadGrammar(baseDir string) (*Grammar, error) {
	omenDir := filepath.Join(baseDir, "Omen")

	g := &Grammar{
		MaxLevel: 10,
		IP:       make(map[int][]string),
		EP:       make(map[string]int),
		CP:       make(map[string]map[int][]string),
		LN:       make(map[int][]int),
	}

	if err := loadConfig(omenDir, g); err != nil {
		return nil, err
	}

	for level := 0; level <= g.MaxLevel; level++ {
		g.IP[level] = nil
		g.LN[level] = nil
	}

	if err := loadNgrams(omenDir, "IP.level", g, "ip"); err != nil {
		return nil, err
	}
	if err := loadNgrams(omenDir, "EP.level", g, "ep"); err != nil {
		return nil, err
	}
	if err := loadNgrams(omenDir, "CP.level", g, "cp"); err != nil {
		return nil, err
	}
	if err := loadLength(omenDir, "LN.level", g, g.NGram); err != nil {
		return nil, err
	}

	return g, nil
}

func loadConfig(omenDir string, g *Grammar) error {
	path := filepath.Join(omenDir, "config.txt")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var section string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		if section == "training_settings" {
			if idx := strings.Index(line, "="); idx != -1 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				switch key {
				case "ngram":
					g.NGram, _ = strconv.Atoi(val)
				}
			}
		}
	}
	return scanner.Err()
}

func loadNgrams(omenDir, filename string, g *Grammar, name string) error {
	path := filepath.Join(omenDir, filename)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		level, err := strconv.Atoi(parts[0])
		if err != nil || level < 0 || level > g.MaxLevel {
			continue
		}
		ngram := parts[1]

		switch name {
		case "ip":
			g.IP[level] = append(g.IP[level], ngram)
		case "ep":
			g.EP[ngram] = level
		case "cp":
			if len(ngram) < 1 {
				continue
			}
			searchStr := ngram[:len(ngram)-1]
			lastChar := ngram[len(ngram)-1:]
			if g.CP[searchStr] == nil {
				g.CP[searchStr] = make(map[int][]string)
			}
			g.CP[searchStr][level] = append(g.CP[searchStr][level], lastChar)
		}
	}
	return scanner.Err()
}

func loadLength(omenDir, filename string, g *Grammar, minSize int) error {
	path := filepath.Join(omenDir, filename)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	curLength := 1
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		level, err := strconv.Atoi(line)
		if err != nil || level < 0 || level > g.MaxLevel {
			curLength++
			continue
		}
		if curLength >= minSize {
			cpLength := curLength - (minSize - 1)
			g.LN[level] = append(g.LN[level], cpLength)
		}
		curLength++
	}
	return scanner.Err()
}
