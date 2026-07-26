### v0.5.3; 2026-07-25
- `pcfg_guesser` ~2× faster than v0.5.2
- ~2,578% faster than Python3 pcfg_guesser.py
- Reduce long-run RAM growth in `pcfg_guesser` (priority-queue frontier)
  - Compact parse-tree representation (interned type IDs, packed nodes, arena reuse)
  - Contiguous index-based heap (fewer per-item allocations)
  - Reuse OMEN optimizer cache per worker
  - Prefer `[]byte` guess building to cut string churn
- Same generation logic; parallel stdout interleaving may differ across runs
- Rename `trainer` to `pcfg_trainer`

### v0.5.2; 2026-03-23
- Fixed ([#5](https://github.com/cyclone-github/pcfg-go/issues/5)) -n {nth} not outputing

### v0.5.1; 2026-03-17
### Session Save/Restore
- Save on SIGINT/SIGTERM
- Resume with `-l` now accumulates stats
- Preserve `first_started`; update only `last_updated` on resume
- Fix session file deletion issue during writability checks
- Faster restore via parallel loading
- Improved session path handling and save path visibility
- Add warning for long-running session restores
### Bug Fixes
- Fix session save failures when downstream pipe exits early ([#2](https://github.com/cyclone-github/pcfg-go/issues/2))
- Address Unicode handling issues in OMEN and parser logic ([#3](https://github.com/cyclone-github/pcfg-go/issues/3))
  - Replace byte slicing with rune-safe handling
  - Fix context parsing edge case (e.g., `pass#123`)
  - Correct UTF-8 handling in website/TLD parsing

### v0.5.0; 2026-03-16
- initial github release
### Overview
- Pure Go rewrite of the Python3 `pcfg_cracker`, designed as a near drop-in replacement with significant performance gains and expanded features
### Highlights
- ~3× faster trainer
- ~40× faster pcfg_guesser
- `$HEX[]` input/output support
- Full multi-byte / Unicode support (not supported in Compiled C Edition)
- Improved and expanded keyboard detection: Fixed/tuned: QWERTY, JCUKEN Added: AZERTY, QWERTZ, Dvorak
- Expanded detection for TLDs, URLs, and emails in trainer
- Auto-save and resume support in pcfg_guesser
- Multi-threaded for high-throughput performance
