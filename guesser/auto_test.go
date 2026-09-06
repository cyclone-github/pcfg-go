package guesser

import (
	"bytes"
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

func TestExtractFoundsPlaintext(t *testing.T) {
	tests := []struct {
		name string
		line []byte
		want string
		ok   bool
	}{
		{"hash:password", []byte("5f4dcc3b5aa765d61d8327deb882cf99:password"), "password", true},
		{"hash:salt:password", []byte("hash:salt:password"), "password", true},
		{"crlf", []byte("hash:password"), "password", true},
		{"empty plaintext", []byte("hash:"), "", false},
		{"missing colon", []byte("password-only"), "", false},
		{"empty line", []byte(""), "", false},
		{"colon only", []byte(":"), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractFoundsPlaintext(tt.line)
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && string(got) != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestStripEOL(t *testing.T) {
	if string(stripEOL([]byte("hash:pass\r\n"))) != "hash:pass" {
		t.Fatal(string(stripEOL([]byte("hash:pass\r\n"))))
	}
	if string(stripEOL([]byte("hash:pass\n"))) != "hash:pass" {
		t.Fatal("lf")
	}
	if string(stripEOL([]byte("hash:pass\r"))) != "hash:pass" {
		t.Fatal("cr")
	}
}

func TestDecodeFoundsHEX(t *testing.T) {
	pw, ok := decodeFoundsPlaintext([]byte("$HEX[68656c6c6f]"))
	if !ok || pw != "hello" {
		t.Fatalf("got %q ok=%v", pw, ok)
	}
	if _, ok := decodeFoundsPlaintext([]byte("$HEX[zz]")); ok {
		t.Fatal("invalid hex should fail")
	}
}

func TestDecodeFoundsDoesNotMutateBuffer(t *testing.T) {
	buf := []byte("h:$HEX[abc\nnext:hello1234")
	line := buf[:len("h:$HEX[abc")]
	plain, ok := extractFoundsPlaintext(line)
	if !ok {
		t.Fatal("extract")
	}
	before := append([]byte(nil), buf...)
	decodeFoundsPlaintext(plain)
	if !bytes.Equal(buf, before) {
		t.Fatalf("mutated backing buffer: %q -> %q", before, buf)
	}
}

func TestClassifyASCIIUTF8(t *testing.T) {
	key, ok := classifyPassword("hello1234")
	if !ok || key != "A5D4" {
		t.Fatalf("hello1234 -> %q ok=%v, want A5D4", key, ok)
	}
	key, ok = classifyPassword("привет")
	if !ok || key[0] != 'A' {
		t.Fatalf("cyrillic -> %q ok=%v", key, ok)
	}
}

func TestClassifyUsesTrainerMultiword(t *testing.T) {
	g := pcfg.Grammar{
		"A5": {{Values: []string{"hello", "world"}, Prob: 1}},
	}
	ig := newIndexedGrammar(g, []pcfg.BaseStructure{{Prob: 1, Replacements: []string{"A5"}}})
	p := newAutoParser(ig)
	key, ok := p.BaseStructureOf("helloworld")
	if !ok || key != "A5A5" {
		t.Fatalf("seeded multiword helloworld -> %q ok=%v, want A5A5", key, ok)
	}
	key, ok = classifyPassword("helloworld")
	if !ok || key != "A10" {
		t.Fatalf("unseeded helloworld -> %q ok=%v, want A10", key, ok)
	}
}

func TestClassifyRejectsEmptyAndControls(t *testing.T) {
	if _, ok := classifyPassword(""); ok {
		t.Fatal("empty")
	}
	if _, ok := classifyPassword("a\tb"); ok {
		t.Fatal("tab")
	}
}

func TestAutoPriorNormalization(t *testing.T) {
	base := []pcfg.BaseStructure{
		{Prob: 0.12, Replacements: []string{"A5", "D4"}},
		{Prob: 0.48, Replacements: []string{"A6"}},
		{Prob: 0.40, Replacements: []string{"M"}},
	}
	s := newAutoSteerer(base)

	if got := s.prior["A5D4"]; math.Abs(got-0.20) > 1e-12 {
		t.Fatalf("A5D4 prior=%v want 0.20", got)
	}
	if got := s.prior["A6"]; math.Abs(got-0.80) > 1e-12 {
		t.Fatalf("A6 prior=%v want 0.80", got)
	}

	sum := 0.0
	for _, p := range s.prior {
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-12 {
		t.Fatalf("prior sum=%v want 1.0", sum)
	}
}

func TestAutoMatchingDistributionStaysNeutral(t *testing.T) {
	base := []pcfg.BaseStructure{
		{Prob: 0.12, Replacements: []string{"A5", "D4"}},
		{Prob: 0.48, Replacements: []string{"A6"}},
		{Prob: 0.40, Replacements: []string{"M"}},
	}
	s := newAutoSteerer(base)
	s.ApplyBatch(map[string]int{"A5D4": 20, "A6": 80}, 100)

	if got := s.Multiplier("A5D4"); math.Abs(got-1.0) > 1e-12 {
		t.Fatalf("A5D4 multiplier=%v want 1.0", got)
	}
	if got := s.Multiplier("A6"); math.Abs(got-1.0) > 1e-12 {
		t.Fatalf("A6 multiplier=%v want 1.0", got)
	}
}

func TestSmoothingAndBounds(t *testing.T) {
	s := &AutoSteerer{
		prior: map[string]float64{"A5D4": 0.05, "A6D4": 0.10, "D4A4": 0.20},
		state: map[string]float64{},
	}

	s.ApplyBatch(map[string]int{"A5D4": 99}, 99)
	if len(s.state) != 0 {
		t.Fatalf("99 founds should not tune: %+v", s.state)
	}

	s.ApplyBatch(map[string]int{"A5D4": 100}, 100)
	m := s.Multiplier("A5D4")
	if m <= 1.0 || m > autoMaxMult {
		t.Fatalf("first batch multiplier %v", m)
	}
	if m > 1.0+autoAlpha*(autoMaxMult-1.0)+1e-9 {
		t.Fatalf("single burst dominated: %v", m)
	}

	first := m
	s.ApplyBatch(map[string]int{"A5D4": 100}, 100)
	second := s.Multiplier("A5D4")
	if second <= first {
		t.Fatalf("repeated batches should increase: %v then %v", first, second)
	}

	for i := 0; i < 40; i++ {
		s.ApplyBatch(map[string]int{"A5D4": 100}, 100)
	}
	if s.Multiplier("A5D4") > autoMaxMult+1e-12 {
		t.Fatalf("exceeded max: %v", s.Multiplier("A5D4"))
	}

	s.state["A6D4"] = 0.5
	s.ApplyBatch(map[string]int{"A5D4": 100}, 100)
	if s.Multiplier("A6D4") < autoMinMult-1e-12 {
		t.Fatalf("below min: %v", s.Multiplier("A6D4"))
	}

	s.state["NaNpath"] = math.NaN()
	if clampMult(math.NaN()) != 1 || clampMult(math.Inf(1)) != 1 {
		t.Fatal("nan/inf")
	}
	if s.Multiplier("missing") != 1 {
		t.Fatal("missing key")
	}
	if s.Multiplier("") != 1 {
		t.Fatal("empty key")
	}
}

func TestEWMAFormula(t *testing.T) {
	s := &AutoSteerer{
		prior: map[string]float64{"A4": 0.01},
		state: map[string]float64{"A4": 1.0},
	}
	s.ApplyBatch(map[string]int{"A4": 100}, 100)
	want := clampMult(autoRetain*1.0 + autoAlpha*autoMaxMult)
	got := s.Multiplier("A4")
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}

func testGrammar() (pcfg.Grammar, []pcfg.BaseStructure) {
	g := pcfg.Grammar{
		"A5": {{Values: []string{"hello"}, Prob: 1.0}},
		"D4": {{Values: []string{"1234"}, Prob: 1.0}},
		"A6": {{Values: []string{"abcdef"}, Prob: 1.0}},
	}
	base := []pcfg.BaseStructure{
		{Prob: 0.7, Replacements: []string{"A5", "D4"}},
		{Prob: 0.3, Replacements: []string{"A6"}},
	}
	return g, base
}

func TestQueueReweightAndChildren(t *testing.T) {
	g, base := testGrammar()
	q := NewPcfgQueue(g, base)
	q.steer = newAutoSteerer(base)

	trainedA5D4 := 0.0
	for _, idx := range q.heap.h {
		e := q.entries[idx]
		key := q.ig.structureKey(q.ptSlice(e.PTOff, e.PTLen))
		if key == "A5D4" {
			trainedA5D4 = e.Trained
			if e.Prob != e.Trained {
				t.Fatalf("without updates Prob should equal Trained")
			}
		}
	}
	if trainedA5D4 == 0 {
		t.Fatal("missing A5D4")
	}

	q.applyAutoBatch(autoBatch{counts: map[string]int{"A5D4": 100}, n: 100})
	var boosted, other float64
	for _, idx := range q.heap.h {
		e := q.entries[idx]
		key := q.ig.structureKey(q.ptSlice(e.PTOff, e.PTLen))
		if key == "A5D4" {
			boosted = e.Prob
			if e.Trained != trainedA5D4 {
				t.Fatal("trained mutated")
			}
			if e.Prob <= e.Trained {
				t.Fatalf("expected boost, trained=%v eff=%v", e.Trained, e.Prob)
			}
			if math.IsNaN(e.Prob) || math.IsInf(e.Prob, 0) || e.Prob <= 0 {
				t.Fatalf("bad effective %v", e.Prob)
			}
		} else {
			other = e.Prob
		}
	}
	if other != 0 && boosted <= other {
		t.Fatalf("A5D4 should still lead: boosted=%v other=%v", boosted, other)
	}

	item := q.Next()
	if item == nil {
		t.Fatal("empty queue")
	}
	releasePTWork(item)
	if q.MaxProbability <= 0 {
		t.Fatal("max prob")
	}

	for _, idx := range q.heap.h {
		e := q.entries[idx]
		sk := q.ig.structureKey(q.ptSlice(e.PTOff, e.PTLen))
		want := q.effective(q.ptSlice(e.PTOff, e.PTLen), e.Trained)
		if math.Abs(e.Prob-want) > 1e-12 {
			t.Fatalf("child %s Prob=%v want %v", sk, e.Prob, want)
		}
	}
}

func TestNonAutoQueueUnchanged(t *testing.T) {
	g, base := testGrammar()
	q := NewPcfgQueue(g, base)
	for _, idx := range q.heap.h {
		e := q.entries[idx]
		if e.Prob != e.Trained {
			t.Fatalf("non-auto Prob %v != Trained %v", e.Prob, e.Trained)
		}
	}
	item := q.Next()
	if item == nil {
		t.Fatal("nil")
	}
	releasePTWork(item)
}

func TestWatcherIgnoresExistingAndPartialAndThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "founds.txt")
	if err := os.WriteFile(path, []byte("oldhash:oldpass\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := startAutoWatcher(ctx, path, true, 20*time.Millisecond, 100, nil)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	writeN := func(n int, prefix string) {
		var buf bytes.Buffer
		for i := 0; i < n; i++ {
			buf.WriteString(prefix)
			buf.WriteByte('\n')
		}
		if _, err := f.Write(buf.Bytes()); err != nil {
			t.Fatal(err)
		}
	}

	writeN(99, "h:hello1234")
	select {
	case <-ch:
		t.Fatal("tuned at 99")
	case <-time.After(80 * time.Millisecond):
	}

	writeN(1, "h:hello1234")
	select {
	case batch := <-ch:
		if batch.n != 100 {
			t.Fatalf("n=%d", batch.n)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected tune at 100")
	}

	writeN(50, "h:hello1234")
	select {
	case <-ch:
		t.Fatal("tuned at 50")
	case <-time.After(80 * time.Millisecond):
	}
	writeN(50, "h:hello1234")
	select {
	case batch := <-ch:
		if batch.n != 100 {
			t.Fatalf("50+50 n=%d", batch.n)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected tune after 50+50")
	}

	writeN(120, "h:hello1234")
	select {
	case batch := <-ch:
		if batch.n != 120 {
			t.Fatalf("120 n=%d", batch.n)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected 120 batch")
	}

	if _, err := f.Write([]byte("h:partial")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ch:
		t.Fatal("partial line tuned")
	case <-time.After(80 * time.Millisecond):
	}
	if _, err := f.Write([]byte("hello1234\n")); err != nil {
		t.Fatal(err)
	}

	f.Close()
	if err := os.WriteFile(path, []byte("replaced:hello1234\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ch:
		t.Fatal("truncation should ignore existing replacement contents")
	case <-time.After(80 * time.Millisecond):
	}
}

func TestSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.sav")
	cfg := &SessionConfig{
		MinProbability: 0.1,
		MaxProbability: 0.9,
		NumGuesses:     12,
		NumParseTrees:  3,
	}
	if err := SaveSession(path, cfg, "rockyou", "uuid-1", false, false, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSession(path)
	if err != nil || got == nil {
		t.Fatalf("load %v %v", got, err)
	}
	if got.MinProbability != 0.1 || got.MaxProbability != 0.9 || got.NumGuesses != 12 {
		t.Fatalf("%+v", got)
	}
	if got.UUID != "uuid-1" || got.FirstStarted != "2026-01-01T00:00:00Z" {
		t.Fatalf("%+v", got)
	}
}

func TestStartAutoMissingFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := startAutoWatcher(ctx, filepath.Join(t.TempDir(), "nope.txt"), false, time.Second, 100, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFoundsColonLimitation(t *testing.T) {
	plain, ok := extractFoundsPlaintext([]byte("hash:salt:pass:word"))
	if !ok || string(plain) != "word" {
		t.Fatalf("last-colon limitation: got %q", plain)
	}
}
