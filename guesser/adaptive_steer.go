package guesser

import (
	"math"
	"sort"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

const (
	adaptiveAlpha      = 0.20
	adaptiveRetain     = 1.0 - adaptiveAlpha
	adaptiveMinMult    = 0.75
	adaptiveMaxMult    = 1.50
	adaptiveMinFounds  = 100
	adaptivePriorFloor = 1e-12
)

// runtime multipliers over trained structure keys; missing keys are 1.0
type AdaptiveSteerer struct {
	prior map[string]float64
	state map[string]float64
}

type adaptiveBatch struct {
	counts map[string]int
	n      int
}

func newAdaptiveSteerer(base []pcfg.BaseStructure) *AdaptiveSteerer {
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
	if pcfgMass > adaptivePriorFloor {
		for key, p := range prior {
			prior[key] = p / pcfgMass
		}
	}

	return &AdaptiveSteerer{
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

func (s *AdaptiveSteerer) Multiplier(key string) float64 {
	if s == nil || key == "" {
		return 1
	}
	m, ok := s.state[key]
	if !ok {
		return 1
	}
	return clampMult(m)
}

func (s *AdaptiveSteerer) ApplyBatch(counts map[string]int, n int) {
	if s == nil || n < adaptiveMinFounds || n <= 0 {
		return
	}

	seen := make(map[string]struct{}, len(counts))
	invN := 1.0 / float64(n)

	for key, c := range counts {
		if c <= 0 {
			continue
		}
		prior, ok := s.prior[key]
		if !ok || prior < adaptivePriorFloor {
			continue
		}
		obs := float64(c) * invN
		lift := obs / prior
		lift = clampMult(lift)
		prev := 1.0
		if v, ok := s.state[key]; ok {
			prev = v
		}
		s.state[key] = clampMult(adaptiveRetain*prev + adaptiveAlpha*lift)
		seen[key] = struct{}{}
	}

	for key, prev := range s.state {
		if _, ok := seen[key]; ok {
			continue
		}
		next := clampMult(adaptiveRetain*prev + adaptiveAlpha*1.0)
		if math.Abs(next-1.0) < 1e-9 {
			delete(s.state, key)
			continue
		}
		s.state[key] = next
	}
}

type adaptiveBoost struct {
	Key  string
	Mult float64
}

func (s *AdaptiveSteerer) topBoosts(k int) []adaptiveBoost {
	if s == nil || k <= 0 {
		return nil
	}
	out := make([]adaptiveBoost, 0, len(s.state))
	for key, m := range s.state {
		if m <= 1.0+1e-9 {
			continue
		}
		out = append(out, adaptiveBoost{Key: key, Mult: m})
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
	if v < adaptiveMinMult {
		return adaptiveMinMult
	}
	if v > adaptiveMaxMult {
		return adaptiveMaxMult
	}
	return v
}
