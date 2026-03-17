package trainer

import (
	"sort"
	"sync"
)

type Counter struct {
	mu sync.Mutex
	M  map[string]int
}

func NewCounter() *Counter {
	return &Counter{M: make(map[string]int)}
}

func (c *Counter) Add(key string, n int) {
	c.mu.Lock()
	c.M[key] += n
	c.mu.Unlock()
}

func (c *Counter) Inc(key string) {
	c.mu.Lock()
	c.M[key]++
	c.mu.Unlock()
}

func (c *Counter) AddBatch(entries map[string]int) {
	if len(entries) == 0 {
		return
	}
	c.mu.Lock()
	for k, v := range entries {
		c.M[k] += v
	}
	c.mu.Unlock()
}

func (c *Counter) MergeFrom(other *Counter) {
	c.AddBatch(other.Snapshot())
}

func (c *Counter) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make(map[string]int, len(c.M))
	for k, v := range c.M {
		cp[k] = v
	}
	return cp
}

type CountEntry struct {
	Key   string
	Count int
}

func (c *Counter) TopN(n int) []CountEntry {
	snap := c.Snapshot()
	entries := make([]CountEntry, 0, len(snap))
	for k, v := range snap {
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
	mu sync.Mutex
	M  map[int]*Counter
}

func NewLenIndexedCounters() *LenIndexedCounters {
	return &LenIndexedCounters{M: make(map[int]*Counter)}
}

func (l *LenIndexedCounters) Inc(length int, value string) {
	l.mu.Lock()
	c, ok := l.M[length]
	if !ok {
		c = NewCounter()
		l.M[length] = c
	}
	l.mu.Unlock()
	c.Inc(value)
}

func (l *LenIndexedCounters) Keys() []int {
	l.mu.Lock()
	defer l.mu.Unlock()
	keys := make([]int, 0, len(l.M))
	for k := range l.M {
		keys = append(keys, k)
	}
	return keys
}

func (l *LenIndexedCounters) Get(length int) *Counter {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.M[length]
}

func (l *LenIndexedCounters) MergeFrom(other *LenIndexedCounters) {
	for _, length := range other.Keys() {
		otherCounter := other.Get(length)
		snap := otherCounter.Snapshot()
		if len(snap) == 0 {
			continue
		}
		l.mu.Lock()
		c, ok := l.M[length]
		if !ok {
			c = NewCounter()
			l.M[length] = c
		}
		l.mu.Unlock()
		c.AddBatch(snap)
	}
}
