package guesser

import (
	"bytes"
	"strings"

	"github.com/cyclone-github/pcfg-go/trainer"
	"github.com/cyclone-github/pcfg-go/trainer/parser"
)

// plaintext after last ':'; lines with no colon are ignored
func extractFoundsPlaintext(line []byte) ([]byte, bool) {
	if len(line) == 0 {
		return nil, false
	}
	idx := bytes.LastIndexByte(line, ':')
	if idx < 0 {
		return nil, false
	}
	plain := line[idx+1:]
	if len(plain) == 0 {
		return nil, false
	}
	return plain, true
}

func stripEOL(line []byte) []byte {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line
}

// same $HEX[] + CheckValid path as trainer file input
func decodeFoundsPlaintext(plain []byte) (string, bool) {
	if len(plain) == 0 {
		return "", false
	}
	pw := string(plain)
	if strings.HasPrefix(pw, "$HEX[") && strings.HasSuffix(pw, "]") {
		result := trainer.Decode([]byte(pw))
		if result.HadError {
			return "", false
		}
		pw = string(result.Decoded)
	}
	if !parser.CheckValid(pw) {
		return "", false
	}
	return pw, true
}
