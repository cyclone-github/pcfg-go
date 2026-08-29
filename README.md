[![Readme Card](https://github-readme-stats-fast.vercel.app/api/pin/?username=cyclone-github&repo=pcfg-go&theme=gruvbox)](https://github.com/cyclone-github/pcfg-go)

[![GitHub issues](https://img.shields.io/github/issues/cyclone-github/pcfg-go.svg)](https://github.com/cyclone-github/pcfg-go/issues)
[![License](https://img.shields.io/github/license/cyclone-github/pcfg-go.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/cyclone-github/pcfg-go.svg)](https://github.com/cyclone-github/pcfg-go/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/cyclone-github/pcfg-go.svg)](https://pkg.go.dev/github.com/cyclone-github/pcfg-go)

# pcfg-go - The Fast PCFG

- Probabilistic Context-Free Grammar (PCFG) password generator - Pure Go Edition
  - pcfg-go is a Pure Go rewrite of the Python3 [pcfg_cracker](https://github.com/lakiw/pcfg_cracker)
  - The goal of this Go implementation is to provide a substantial performance improvement for both trainer and guesser over the original Python3 version, while also adding features such as supporting `$HEX[]` input and multi-byte character support — which is not implemented in the [Pure C pcfg_guesser](https://github.com/lakiw/compiled-pcfg)
  - Credits for the original python3 pcfg_cracker belong to the author [@lakiw](https://github.com/lakiw)
  - Rules trained by `pcfg_trainer` are compatible with hashcat's new PCFG `-a 4` attack mode. See the [hashcat PCFG implementation commit](https://github.com/hashcat/hashcat/commit/ce90358b2251726c99742e5761ae3dfd41dbd4b2) for details.
  
---

## Install

**pcfg_trainer:**
```bash
go install -ldflags="-s -w" github.com/cyclone-github/pcfg-go/cmd/pcfg_trainer@main
```

**pcfg_guesser:**
```bash
go install -ldflags="-s -w" github.com/cyclone-github/pcfg-go/cmd/pcfg_guesser@main
```

---

### Additions & improvements

* **Performance** — ~53.94× faster `pcfg_trainer` and ~12.16× faster `pcfg_guesser` vs Python 3 (see Benchmarks below)
* **$HEX[] input** — Trainer accepts `$HEX[...]` encoded passwords in the training wordlist (multi-byte support)
* **Ctrl+C handling** — Pressing Ctrl+C auto-saves the session in `pcfg_guesser`
* **Multi-keyboard layouts** — QWERTY, AZERTY, QWERTZ, Dvorak, JCUKEN (Russian Cyrillic)
* **Expanded TLD list** — Legacy TLDs, ccTLDs, gTLDs (`.info`, `.xyz`, `.app`, `.dev`, etc.), and short TLDs (`.co`, `.io`, `.ai`, `.me`, `.gg`); improves both website and email detection
* **Improved website detection** — Broader URL/prefix detection (`http://`, `https://`, `www.`, etc.) and host extraction
* **Multi-threaded architecture** — `pcfg_guesser` is multi-threaded for increased performance
* **Compiled binary** — No fuss; pcfg-go uses compiled binaries for speed and easy deployment

---

## Benchmarks on **rockyou training set**

### `pcfg_trainer -r rockyou -t rockyou.txt`
- `Python3 trainer: 13m21s (801s)`
- `Go pcfg_trainer: 14.85s`
- **Speedup: ~53.94×**

### `pcfg_guesser -r rockyou -n 100000000` (100 million guesses written to stdout)
- `Python3 pcfg_guesser: 195.5s`
- `Go pcfg_guesser: 16.08s`
- **Speedup: ~12.16×**

---

## Usage

### pcfg_trainer

Train a new ruleset from wordlist:

```bash
pcfg_trainer -r rule_name -t wordlist.txt
```

### pcfg_guesser

Generate guesses from a trained ruleset:

```bash
pcfg_guesser -r rule_name
```

Auto PCFG steering from new founds.

```bash
pcfg_guesser -r rule_name -auto founds_file.pot | hashcat -m 0 hashes.txt -o founds_file.pot ...
```

Session save/restore:

```bash
pcfg_guesser -r rule_name -s my_session   # save to my_session.sav on exit
pcfg_guesser -r rule_name -s my_session -l # load and resume
```

Press Ctrl+C to save session and exit.

### Piping into hashcat

```bash
pcfg_guesser -r rule_name -s my_session | hashcat -m 0 hashes.txt...
```

### Using trained rules with hashcat -a 4 PCFG mode

```bash
hashcat -m 0 -a 4 hashes.txt path_to_pcfg_rule_dir ...
```

---

## Flags

**pcfg_trainer**

pcfg-go vs pcfg-python3 flags

| Go | Python3 | Description |
|----|---------|-------------|
| -r | --rule | Ruleset name |
| -t | --training | Training wordlist (required) |
| -e | --encoding | File encoding |
| -C | --comments | Config comments |
| -S | --save_sensitive | Save emails, URLs |
| -p | --prefixcount | Lines prefixed with count |
| -n | --ngram | OMEN ngram size (2-5) |
| -a | --alphabet | Alphabet size for Markov |
| -c | --coverage | PCFG vs OMEN coverage |
| -m | --multiword | Pre-train multiword file |
| -h | --help | Help |
| -version | --version | Version info |

**pcfg_guesser**

pcfg-go vs pcfg-python3 flags

| Go | Python3 | Description |
|----|---------|-------------|
| -r | --rule | Ruleset name |
| -s | --session | Session name |
| -l | --load | Load previous session |
| -n | --limit | Max guesses |
| -b | --skip_brute | Skip OMEN/Markov |
| -a | --all_lower | No case mangling |
| -d | --debug | Debug output |
| -auto | Auto-steer PCFG from founds file |
| -h | --help | Help |
| -version | --version | Version info |

---

## Compile from source (linux)

Requires Go and Git.

```bash
git clone https://github.com/cyclone-github/pcfg-go.git
cd pcfg-go
go mod tidy
mkdir -p bin
go build -ldflags="-s -w" -o bin/pcfg_trainer ./cmd/pcfg_trainer
go build -ldflags="-s -w" -o bin/pcfg_guesser ./cmd/pcfg_guesser
```

**Install to $GOPATH/bin:**
```bash
go install -ldflags="-s -w" ./cmd/pcfg_trainer
go install -ldflags="-s -w" ./cmd/pcfg_guesser
```

[Compile from source how-to](https://github.com/cyclone-github/scripts/blob/main/intro_to_go.txt)
