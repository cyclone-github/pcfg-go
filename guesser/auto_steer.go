package guesser

import (
	"math"
	"sort"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

const (
	autoAlpha      = 0.20
	autoRetain     = 1.0 - autoAlpha
	autoMinMult    = 0.75
	autoMaxMult    = 1.50
	autoMinFounds  = 100
	autoPriorFloor = 1e-12
)

// runtime multipliers over trained structure keys; missing keys are 1.0
type AutoSteerer struct {
	prior map[string]float64
	state map[string]float64
}

type autoBatch struct {
	counts map[string]int
	n      int
}

func newAutoSteerer(base []pcfg.BaseStructure) *AutoSteerer {
	prior := make(map[string]float64, len(base))
	for i := range base {
		key := coreStructure(base[i].Replacements)
		if key == "" || key == "M" {
			continue
		}
		prior[key] += base[i].Prob
	}

	pcfgMass := 0.0
	for _, p := range prior {
		pcfgMass += p
	}
	if pcfgMass > autoPriorFloor {
		for key, p := range prior {
			prior[key] = p / pcfgMass
		}
	}

	return &AutoSteerer{
		prior: prior,
		state: make(map[string]float64),
	}
}

// skip C so keys match trainer BaseStructureCreation
func coreStructure(replacements []string) string {
	n := 0
	for _, r := range replacements {
		if len(r) > 0 && r[0] == 'C' {
			continue
		}
		n += len(r)
	}
	if n == 0 {
		return ""
	}
	b := make([]byte, 0, n)
	for _, r := range replacements {
		if len(r) > 0 && r[0] == 'C' {
			continue
		}
		b = append(b, r...)
	}
	return string(b)
}

func (s *AutoSteerer) Multiplier(key string) float64 {
	if s == nil || key == "" {
		return 1
	}
	m, ok := s.state[key]
	if !ok {
		return 1
	}
	return clampMult(m)
}

func (s *AutoSteerer) ApplyBatch(counts map[string]int, n int) {
	if s == nil || n < autoMinFounds || n <= 0 {
		return
	}

	seen := make(map[string]struct{}, len(counts))
	invN := 1.0 / float64(n)

	for key, c := range counts {
		if c <= 0 {
			continue
		}
		prior, ok := s.prior[key]
		if !ok || prior < autoPriorFloor {
			continue
		}
		obs := float64(c) * invN
		lift := obs / prior
		lift = clampMult(lift)
		prev := 1.0
		if v, ok := s.state[key]; ok {
			prev = v
		}
		s.state[key] = clampMult(autoRetain*prev + autoAlpha*lift)
		seen[key] = struct{}{}
	}

	for key, prev := range s.state {
		if _, ok := seen[key]; ok {
			continue
		}
		next := clampMult(autoRetain*prev + autoAlpha*1.0)
		if math.Abs(next-1.0) < 1e-9 {
			delete(s.state, key)
			continue
		}
		s.state[key] = next
	}
}

type autoBoost struct {
	Key  string
	Mult float64
}

func (s *AutoSteerer) topBoosts(k int) []autoBoost {
	if s == nil || k <= 0 {
		return nil
	}
	out := make([]autoBoost, 0, len(s.state))
	for key, m := range s.state {
		if m <= 1.0+1e-9 {
			continue
		}
		out = append(out, autoBoost{Key: key, Mult: m})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mult != out[j].Mult {
			return out[i].Mult > out[j].Mult
		}
		return out[i].Key < out[j].Key
	})
	if len(out) > k {
		out = out[:k]
	}
	return out
}

func clampMult(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 1
	}
	if v < autoMinMult {
		return autoMinMult
	}
	if v > autoMaxMult {
		return autoMaxMult
	}
	return v
}
