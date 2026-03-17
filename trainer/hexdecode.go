package trainer

import (
	"bytes"
	"encoding/hex"
)

type Result struct {
	Decoded    []byte
	HexContent []byte
	IsHex      bool
	HadError   bool
}

func Decode(line []byte) Result {
	hexPrefix := []byte("$HEX[")

	if !bytes.HasPrefix(line, hexPrefix) {
		return Result{Decoded: line, HexContent: line, IsHex: false, HadError: false}
	}

	hadError := false

	if len(line) == 0 {
		return Result{Decoded: line, HexContent: line, IsHex: true, HadError: true}
	}

	if line[len(line)-1] != ']' {
		line = append(line, ']')
		hadError = true
	}

	startIdx := bytes.IndexByte(line, '[')
	endIdx := bytes.LastIndexByte(line, ']')
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return Result{Decoded: line, HexContent: line, IsHex: true, HadError: true}
	}

	hexContent := line[startIdx+1 : endIdx]

	decodedBytes := make([]byte, hex.DecodedLen(len(hexContent)))
	n, err := hex.Decode(decodedBytes, hexContent)
	if err != nil {
		cleaned := cleanHex(hexContent)
		decodedBytes = make([]byte, hex.DecodedLen(len(cleaned)))
		n, err = hex.Decode(decodedBytes, cleaned)
		if err != nil {
			return Result{Decoded: line, HexContent: line, IsHex: true, HadError: true}
		}
		hadError = true
	}
	decodedBytes = decodedBytes[:n]

	return Result{Decoded: decodedBytes, HexContent: hexContent, IsHex: true, HadError: hadError}
}

func cleanHex(content []byte) []byte {
	cleaned := make([]byte, 0, len(content))
	for _, b := range content {
		if isHexChar(b) {
			cleaned = append(cleaned, b)
		}
	}
	if len(cleaned)%2 != 0 {
		cleaned = append([]byte{'0'}, cleaned...)
	}
	return cleaned
}

func isHexChar(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func IsHexEncoded(line []byte) bool {
	return bytes.HasPrefix(line, []byte("$HEX["))
}
