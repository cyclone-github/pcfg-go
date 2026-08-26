package trainer

import (
	"sort"
)

// Counter is a simple string->count map. It is intentionally lock-free: during
// the parallel passes each worker owns its own Counter instances, and all
// merges, snapshots, and reads happen sequentially on the main goroutine after
// the workers have finished. Avoiding a mutex removes significant overhead from
// the per-password hot path.
type Counter struct {
	M map[string]int
}

func NewCounter() *Counter {
	return &Counter{M: make(map[string]int)}
}

func (c *Counter) Add(key string, n int) {
	c.M[key] += n
}

func (c *Counter) Inc(key string) {
	c.M[key]++
}

func (c *Counter) AddBatch(entries map[string]int) {
	if len(entries) == 0 {
		return
	}
	for k, v := range entries {
		c.M[k] += v
	}
}

func (c *Counter) MergeFrom(other *Counter) {
	if len(c.M) == 0 {
		c.M = other.M
		return
	}
	if len(other.M) > len(c.M) {
		c.M, other.M = other.M, c.M
	}
	c.AddBatch(other.M)
}

// Snapshot returns the underlying map. Callers must not mutate it. It is
// returned directly (no copy) since all access is single-goroutine.
func (c *Counter) Snapshot() map[string]int {
	return c.M
}

type CountEntry struct {
	Key   string
	Count int
}

func (c *Counter) TopN(n int) []CountEntry {
	entries := make([]CountEntry, 0, len(c.M))
	for k, v := range c.M {
		entries = append(entries, CountEntry{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].Key < entries[j].Key
	})
	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n]
}

type LenIndexedCounters struct {
	M map[int]*Counter
}

func NewLenIndexedCounters() *LenIndexedCounters {
	return &LenIndexedCounters{M: make(map[int]*Counter)}
}

func (l *LenIndexedCounters) Inc(length int, value string) {
	c, ok := l.M[length]
	if !ok {
		c = NewCounter()
		l.M[length] = c
	}
	c.Inc(value)
}

func (l *LenIndexedCounters) Keys() []int {
	keys := make([]int, 0, len(l.M))
	for k := range l.M {
		keys = append(keys, k)
	}
	return keys
}

func (l *LenIndexedCounters) Get(length int) *Counter {
	return l.M[length]
}

func (l *LenIndexedCounters) MergeFrom(other *LenIndexedCounters) {
	for length, otherCounter := range other.M {
		if len(otherCounter.M) == 0 {
			continue
		}
		c, ok := l.M[length]
		if !ok {
			l.M[length] = otherCounter
			continue
		}
		c.MergeFrom(otherCounter)
	}
}
