# pcfg-go
*Coming soon...*
---
Probabilistic Context Free Grammar (PCFG) password generator, Pure Go Edition

pcfg-go is a Pure Go rewrite of the python3 pcfg_cracker - https://github.com/lakiw/pcfg_cracker 

The goal of this Go implementation of pcfg_cracker is to provide a substantial performance improvement over the original Python3 version, while also adding features such as supporting `$HEX[]` input/output and multi-byte char support -- which is not implemented in the [Pure C pcfg_guesser](https://github.com/lakiw/compiled-pcfg). 

All credits for pcfg_cracker belong to the original pcfg_cracker author, [lakiw](https://github.com/lakiw).

---

### trainer benchmark
- `1 million password training set`
  - `Python3 trainer: 97.2 seconds`
  - `Go trainer: 42.8 seconds`
  - `Go trainer benchmarked approximately 2.27×, or 227%, faster than Python3`

### pcfg_guesser benchmark
- `20-second head-to-head benchmark`
  - `Python3 guesser: 422,249 lines/sec`
  - `Go guesser: 4,863,412 lines/sec`
  - `Go guesser benchmarked approximately 11.5×, or 1152%, faster than Python3`
