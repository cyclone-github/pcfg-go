package trainer

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ProgramInfo struct {
	Name          string
	Version       string
	Author        string
	Contact       string
	RuleName      string
	TrainingFile  string
	Encoding      string
	Comments      string
	SaveSensitive bool
	PrefixCount   bool
	NGram         int
	AlphabetSize  int
	Coverage      float64
	MaxLen        int
	Multiword     string
}

func SaveConfigFile(baseDir string, info *ProgramInfo, fileInput *FileInput, pcfgParser *PCFGParser) error {
	path := filepath.Join(baseDir, "config.ini")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	uid := hex.EncodeToString(b)

	fmt.Fprintln(f, "[TRAINING_PROGRAM_DETAILS]")
	fmt.Fprintf(f, "contact = %s\n", info.Contact)
	fmt.Fprintf(f, "author = %s\n", info.Author)
	fmt.Fprintf(f, "program = %s\n", info.Name)
	fmt.Fprintf(f, "version = %s\n", info.Version)
	fmt.Fprintln(f)

	fmt.Fprintln(f, "[TRAINING_DATASET_DETAILS]")
	fmt.Fprintf(f, "comments = %s\n", info.Comments)
	fmt.Fprintf(f, "filename = %s\n", filepath.Base(info.TrainingFile))
	fmt.Fprintf(f, "encoding = %s\n", info.Encoding)
	fmt.Fprintf(f, "uuid = %s\n", uid)
	fmt.Fprintf(f, "number_of_passwords_in_set = %d\n", fileInput.NumPasswords)
	fmt.Fprintf(f, "number_of_encoding_errors = %d\n", fileInput.NumEncodingErr)
	fmt.Fprintln(f)

	replacements := []map[string]string{
		{"Config_id": "BASE_A", "Transition_id": "A"},
		{"Config_id": "BASE_D", "Transition_id": "D"},
		{"Config_id": "BASE_O", "Transition_id": "O"},
		{"Config_id": "BASE_K", "Transition_id": "K"},
		{"Config_id": "BASE_X", "Transition_id": "X"},
		{"Config_id": "BASE_Y", "Transition_id": "Y"},
	}
	replJSON, _ := marshalPythonJSON(replacements)

	fmt.Fprintln(f, "[START]")
	fmt.Fprintln(f, "name = Base Structure")
	fmt.Fprintln(f, "function = Transparent")
	fmt.Fprintln(f, "directory = Grammar")
	fmt.Fprintln(f, "comments = Base structures as defined by the original PCFG Paper, with some renaming to prevent naming collisions. Examples are A4D2 from the training word pass12")
	fmt.Fprintln(f, "file_type = Flat")
	fmt.Fprintln(f, "inject_type = Wordlist")
	fmt.Fprintf(f, "is_terminal = %s\n", "False")
	fmt.Fprintf(f, "replacements = %s\n", string(replJSON))
	fmt.Fprintf(f, "filenames = %s\n", `["grammar.txt"]`)
	fmt.Fprintln(f)

	alphaFiles := createFilenameList(pcfgParser.CountAlpha)
	alphaRepl, _ := marshalPythonJSON([]map[string]string{{"Config_id": "CAPITALIZATION", "Transition_id": "Capitalization"}})
	alphaFilesJSON, _ := marshalPythonJSON(alphaFiles)

	fmt.Fprintln(f, "[BASE_A]")
	fmt.Fprintln(f, "name = A")
	fmt.Fprintln(f, "function = Shadow")
	fmt.Fprintln(f, "directory = Alpha")
	fmt.Fprintln(f, "comments = (A)lpha letter replacements for base structure. Aka pass12 = A4D2, so this is the A4. Note, this is encoding specific so non-ASCII characters may be considered alpha. For example Cyrillic characters will be considered alpha characters")
	fmt.Fprintln(f, "file_type = Length")
	fmt.Fprintln(f, "inject_type = Wordlist")
	fmt.Fprintf(f, "is_terminal = %s\n", "False")
	fmt.Fprintf(f, "replacements = %s\n", string(alphaRepl))
	fmt.Fprintf(f, "filenames = %s\n", string(alphaFilesJSON))
	fmt.Fprintln(f)

	digitFiles := createFilenameList(pcfgParser.CountDigits)
	digitFilesJSON, _ := marshalPythonJSON(digitFiles)

	fmt.Fprintln(f, "[BASE_D]")
	fmt.Fprintln(f, "name = D")
	fmt.Fprintln(f, "function = Copy")
	fmt.Fprintln(f, "directory = Digits")
	fmt.Fprintln(f, "comments = (D)igit replacement for base structure. Aka pass12 = L4D2, so this is the D2")
	fmt.Fprintln(f, "file_type = Length")
	fmt.Fprintln(f, "inject_type = Copy")
	fmt.Fprintf(f, "is_terminal = %s\n", "True")
	fmt.Fprintf(f, "filenames = %s\n", string(digitFilesJSON))
	fmt.Fprintln(f)

	otherFiles := createFilenameList(pcfgParser.CountOther)
	otherFilesJSON, _ := marshalPythonJSON(otherFiles)

	fmt.Fprintln(f, "[BASE_O]")
	fmt.Fprintln(f, "name = O")
	fmt.Fprintln(f, "function = Copy")
	fmt.Fprintln(f, "directory = Other")
	fmt.Fprintln(f, "comments = (O)ther character replacement for base structure. Aka pass$$ = L4S2, so this is the S2")
	fmt.Fprintln(f, "file_type = Length")
	fmt.Fprintln(f, "inject_type = Copy")
	fmt.Fprintf(f, "is_terminal = %s\n", "True")
	fmt.Fprintf(f, "filenames = %s\n", string(otherFilesJSON))
	fmt.Fprintln(f)

	kbFiles := createFilenameList(pcfgParser.CountKeyboard)
	kbFilesJSON, _ := marshalPythonJSON(kbFiles)

	fmt.Fprintln(f, "[BASE_K]")
	fmt.Fprintln(f, "name = K")
	fmt.Fprintln(f, "function = Copy")
	fmt.Fprintln(f, "directory = Keyboard")
	fmt.Fprintln(f, "comments = (K)eyboard replacement for base structure. Aka test1qaz2wsx = L4K4K4, so this is the K4s")
	fmt.Fprintln(f, "file_type = Length")
	fmt.Fprintln(f, "inject_type = Copy")
	fmt.Fprintf(f, "is_terminal = %s\n", "True")
	fmt.Fprintf(f, "filenames = %s\n", string(kbFilesJSON))
	fmt.Fprintln(f)

	fmt.Fprintln(f, "[BASE_X]")
	fmt.Fprintln(f, "name = X")
	fmt.Fprintln(f, "function = Copy")
	fmt.Fprintln(f, "directory = Context")
	fmt.Fprintln(f, "comments = conte(X)t sensitive replacements to the base structure. This is mostly a grab bag of things like #1 or ;p")
	fmt.Fprintln(f, "file_type = Flat")
	fmt.Fprintln(f, "inject_type = Copy")
	fmt.Fprintf(f, "is_terminal = %s\n", "True")
	fmt.Fprintf(f, "filenames = %s\n", `["1.txt"]`)
	fmt.Fprintln(f)

	fmt.Fprintln(f, "[BASE_Y]")
	fmt.Fprintln(f, "name = Y")
	fmt.Fprintln(f, "function = Copy")
	fmt.Fprintln(f, "directory = Years")
	fmt.Fprintln(f, "comments = Years to replace with")
	fmt.Fprintln(f, "file_type = Flat")
	fmt.Fprintln(f, "inject_type = Copy")
	fmt.Fprintf(f, "is_terminal = %s\n", "True")
	fmt.Fprintf(f, "filenames = %s\n", `["1.txt"]`)
	fmt.Fprintln(f)

	capFiles := createFilenameList(pcfgParser.CountAlphaMasks)
	capFilesJSON, _ := marshalPythonJSON(capFiles)

	fmt.Fprintln(f, "[CAPITALIZATION]")
	fmt.Fprintln(f, "name = C")
	fmt.Fprintln(f, "function = Capitalization")
	fmt.Fprintln(f, "directory = Capitalization")
	fmt.Fprintln(f, "comments = Capitalization Masks for words. Aka LLLLUUUU for passWORD")
	fmt.Fprintln(f, "file_type = Length")
	fmt.Fprintln(f, "inject_type = Copy")
	fmt.Fprintf(f, "is_terminal = %s\n", "True")
	fmt.Fprintf(f, "filenames = %s\n", string(capFilesJSON))
	fmt.Fprintln(f)

	return nil
}

func marshalPythonJSON(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	s := string(data)
	s = strings.ReplaceAll(s, ",", ", ")
	s = strings.ReplaceAll(s, ":", ": ")
	return []byte(s), nil
}

func createFilenameList(counters *LenIndexedCounters) []string {
	keys := counters.Keys()
	sort.Ints(keys)
	filenames := make([]string, 0, len(keys))
	for _, k := range keys {
		filenames = append(filenames, fmt.Sprintf("%d.txt", k))
	}
	return filenames
}
