[![License](https://img.shields.io/github/license/cyclone-github/pcfg-go.svg)](LICENSE)

# pcfg-go -auto POC

## This is a dev branch of pcfg-go to test an -auto POC feature which introduces a live feedback loop to the guesser. This allows the PCFG guesser to prioritize previously trained probabilities which are producing founds, and de-prioritize those which are underperforming. This uses the same pcfg_trainer and training rules as before and only affects pcfg_guesser when `-auto {founds_file}` is used. 

Auto PCFG steering from new founds.

```bash
pcfg_guesser -r rule_name -auto founds_file.pot | hashcat -m 0 hashes.txt -o founds_file.pot ...
```

## Install `-auto POC`

**pcfg_guesser:**
```bash
go install -ldflags="-s -w" github.com/cyclone-github/pcfg-go/cmd/pcfg_guesser@pcfg-auto
```

---

## Flags

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
| -auto | N/A |Auto-steer PCFG from founds file |
| -h | --help | Help |
| -version | --version | Version info |

---

## Compile from source (linux)

Requires Go.

```bash
go mod tidy
mkdir -p bin
go build -ldflags="-s -w" -o bin/pcfg_guesser ./cmd/pcfg_guesser
```
