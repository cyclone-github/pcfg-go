### v0.5.0; 2026-03-16
```
initial github release

### Overview
Pure Go rewrite of the Python3 `pcfg_cracker`, designed as a near drop-in replacement with significant performance gains and expanded features

### Highlights
~3× faster trainer
~40× faster pcfg_guesser
`$HEX[]` input/output support
Full multi-byte / Unicode support (not supported in Compiled C Edition)
Improved and expanded keyboard detection: Fixed/tuned: QWERTY, JCUKEN Added: AZERTY, QWERTZ, Dvorak
Expanded detection for TLDs, URLs, and emails in trainer
Auto-save and resume support in pcfg_guesser
Multi-threaded for high-throughput performance
```