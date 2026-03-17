package trainer

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

func LoadPasswordsToSlice(filename string, prefixCount bool) ([]string, int, int, bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("opening training file: %w", err)
	}
	defer f.Close()

	var passwords []string
	var numPasswords, numEncodingErr int
	dupDetection := make(map[string]bool)
	numToCheck := 100000
	var duplicatesFound bool

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	var lineBuf []byte
	for scanner.Scan() {
		lineBuf = scanner.Bytes()

		lineBuf = bytes.TrimRight(lineBuf, "\r\n")
		if len(lineBuf) == 0 {
			continue
		}

		cleanPassword := string(lineBuf)

		n := 1

		if prefixCount {
			trimmed := bytes.TrimLeft(lineBuf, " \t")
			sepIdx := bytes.IndexAny(trimmed, " \t")
			if sepIdx < 0 {
				continue
			}
			count, err := strconv.Atoi(string(trimmed[:sepIdx]))
			if err != nil {
				continue
			}
			n = count
			cleanPassword = string(bytes.TrimSpace(trimmed[sepIdx+1:]))
		}

		if len(cleanPassword) >= 7 && cleanPassword[:5] == "$HEX[" && cleanPassword[len(cleanPassword)-1] == ']' {
			result := Decode([]byte(cleanPassword))
			if result.HadError {
				numEncodingErr += n
				continue
			}
			cleanPassword = string(result.Decoded)
		}

		if !parser.CheckValid(cleanPassword) {
			continue
		}

		numPasswords += n

		if prefixCount && n > 1 {
			duplicatesFound = true
		}
		if !duplicatesFound && numPasswords < numToCheck {
			if dupDetection[cleanPassword] {
				duplicatesFound = true
				dupDetection = nil
			} else {
				dupDetection[cleanPassword] = true
			}
		}

		for i := 0; i < n; i++ {
			passwords = append(passwords, cleanPassword)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, 0, false, err
	}

	return passwords, numPasswords, numEncodingErr, duplicatesFound, nil
}
