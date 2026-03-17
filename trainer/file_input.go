package trainer

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

type FileInput struct {
	Filename                string
	Encoding                string
	PrefixCount             bool
	NumPasswords            int
	NumEncodingErr          int
	DuplicatesFound         bool
	NumToCheckForDuplicates int
}

func (fi *FileInput) ReadPasswords(callback func(password string)) error {
	f, err := os.Open(fi.Filename)
	if err != nil {
		return fmt.Errorf("opening training file: %w", err)
	}
	defer f.Close()

	dupDetection := make(map[string]bool)
	numToCheck := 100000
	fi.NumToCheckForDuplicates = numToCheck

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		cleanPassword := strings.TrimRight(line, "\r\n")

		n := 1
		if fi.PrefixCount {
			trimmed := strings.TrimLeft(cleanPassword, " ")
			parts := strings.SplitN(trimmed, " ", 2)
			if len(parts) != 2 {
				continue
			}
			count, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			n = count
			cleanPassword = parts[1]
		}

		if strings.HasPrefix(cleanPassword, "$HEX[") && strings.HasSuffix(cleanPassword, "]") {
			result := Decode([]byte(cleanPassword))
			if result.HadError {
				fi.NumEncodingErr += n
				continue
			}
			cleanPassword = string(result.Decoded)
		}

		if !parser.CheckValid(cleanPassword) {
			continue
		}

		fi.NumPasswords += n

		if fi.PrefixCount && n > 1 {
			fi.DuplicatesFound = true
		}
		if !fi.DuplicatesFound && fi.NumPasswords < fi.NumToCheckForDuplicates {
			if dupDetection[cleanPassword] {
				fi.DuplicatesFound = true
				dupDetection = nil

			} else {
				dupDetection[cleanPassword] = true
			}
		}

		for i := 0; i < n; i++ {
			callback(cleanPassword)
		}
	}

	return scanner.Err()
}
