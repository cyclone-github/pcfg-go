package guesser

import (
	"container/heap"
	"runtime"
	"sync"
	"sync/atomic"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// max parse-tree depth observed in rulesets is ~168; leave headroom for scratch
const maxPTScratch = 256

// stored by value in a contiguous slice (no per-item heap object)
type queueEntry struct {
	Prob     float64
	BaseProb float64
	Seq      int64
	PTOff    uint32
	PTLen    uint16
}

// max-probability heap of indices into PcfgQueue.entries
type indexHeap struct {
	q *PcfgQueue
	h []int32
}

func (h indexHeap) Len() int { return len(h.h) }

func (h indexHeap) Less(i, j int) bool {
	a := &h.q.entries[h.h[i]]
	b := &h.q.entries[h.h[j]]
	if a.Prob != b.Prob {
		return a.Prob > b.Prob
	}
	return a.Seq < b.Seq
}

func (h indexHeap) Swap(i, j int) {
	h.h[i], h.h[j] = h.h[j], h.h[i]
}

func (h *indexHeap) Push(x interface{}) {
	h.h = append(h.h, x.(int32))
}

func (h *indexHeap) Pop() interface{} {
	old := h.h
	n := len(old)
	x := old[n-1]
	h.h = old[:n-1]
	return x
}

// popped parse tree handed to a guess worker (short-lived)
type PTWork struct {
	Prob float64
	PT   []packedNode
}

// manage priority queue for guess generation
type PcfgQueue struct {
	heap           indexHeap
	entries        []queueEntry
	freeEntries    []int32
	ptArena        []packedNode
	ptFree         [][]uint32 // length -> free offsets into ptArena
	ig             *IndexedGrammar
	MaxProbability float64
	MinProbability float64
	seqCounter     atomic.Int64
}

// create and initializes a priority queue with base structures
func NewPcfgQueue(grammar pcfg.Grammar, base []pcfg.BaseStructure) *PcfgQueue {
	return newPcfgQueueWithSave(grammar, base, 0, 1)
}

// creates queue restored from session (minProb, maxProb)
func NewPcfgQueueFromSave(grammar pcfg.Grammar, base []pcfg.BaseStructure, minProb, maxProb float64) *PcfgQueue {
	return newPcfgQueueWithSave(grammar, base, minProb, maxProb)
}

func newPcfgQueueWithSave(grammar pcfg.Grammar, base []pcfg.BaseStructure, minProb, maxProb float64) *PcfgQueue {
	ig := newIndexedGrammar(grammar, base)
	q := &PcfgQueue{
		ig:             ig,
		MaxProbability: maxProb,
		MinProbability: minProb,
		entries:        make([]queueEntry, 0, len(base)),
		ptArena:        make([]packedNode, 0, len(base)*4),
		ptFree:         make([][]uint32, maxPTScratch+1),
	}
	q.heap.q = q
	q.heap.h = make([]int32, 0, len(base))

	if minProb > 0 || maxProb < 1 {
		// restore: run tree traversal in parallel across base structures
		restoreProbOrderParallel(q, base, minProb, maxProb)
	} else {
		for i := range base {
			b := &base[i]
			off := q.allocPT(len(b.Replacements))
			pt := q.ptSlice(off, uint16(len(b.Replacements)))
			for j, r := range b.Replacements {
				pt[j] = packedNode{Type: ig.intern(r), Index: 0}
			}
			prob := ig.findProb(pt, b.Prob)
			q.pushEntry(prob, b.Prob, off, uint16(len(b.Replacements)))
		}
	}

	return q
}

// interned grammar used by this queue
func (q *PcfgQueue) IndexedGrammar() *IndexedGrammar {
	return q.ig
}

func (q *PcfgQueue) ptSlice(off uint32, length uint16) []packedNode {
	return q.ptArena[off : off+uint32(length)]
}

func (q *PcfgQueue) allocPT(n int) uint32 {
	if n <= 0 {
		return 0
	}
	if n < len(q.ptFree) {
		if freelist := q.ptFree[n]; len(freelist) > 0 {
			off := freelist[len(freelist)-1]
			q.ptFree[n] = freelist[:len(freelist)-1]
			return off
		}
	}
	off := uint32(len(q.ptArena))
	q.ptArena = append(q.ptArena, make([]packedNode, n)...)
	return off
}

func (q *PcfgQueue) freePT(off uint32, length uint16) {
	if length == 0 {
		return
	}
	n := int(length)
	if n >= len(q.ptFree) {
		// rare oversized PT: grow freelist table; still reclaim
		newFree := make([][]uint32, n+1)
		copy(newFree, q.ptFree)
		q.ptFree = newFree
	}
	q.ptFree[n] = append(q.ptFree[n], off)
}

func (q *PcfgQueue) allocEntry() int32 {
	if n := len(q.freeEntries); n > 0 {
		idx := q.freeEntries[n-1]
		q.freeEntries = q.freeEntries[:n-1]
		return idx
	}
	idx := int32(len(q.entries))
	q.entries = append(q.entries, queueEntry{})
	return idx
}

func (q *PcfgQueue) freeEntry(idx int32) {
	e := &q.entries[idx]
	q.freePT(e.PTOff, e.PTLen)
	*e = queueEntry{}
	q.freeEntries = append(q.freeEntries, idx)
}

func (q *PcfgQueue) pushEntry(prob, baseProb float64, ptOff uint32, ptLen uint16) {
	idx := q.allocEntry()
	seq := q.seqCounter.Add(1)
	q.entries[idx] = queueEntry{
		Prob:     prob,
		BaseProb: baseProb,
		Seq:      seq,
		PTOff:    ptOff,
		PTLen:    ptLen,
	}
	heap.Push(&q.heap, idx)
}

// restore per base structure in parallel, then merge
func restoreProbOrderParallel(q *PcfgQueue, base []pcfg.BaseStructure, minProb, maxProb float64) {
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}
	if numWorkers > len(base) {
		numWorkers = len(base)
	}

	type collected struct {
		prob, baseProb float64
		pt             []packedNode
	}

	workCh := make(chan int, len(base))
	for i := range base {
		workCh <- i
	}
	close(workCh)

	var mu sync.Mutex
	var all []collected

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make([]collected, 0, 4096)
			var scratch, scratch2 [maxPTScratch]packedNode
			for bi := range workCh {
				b := &base[bi]
				pt := q.ig.packPT(b.Replacements)
				prob := q.ig.findProb(pt, b.Prob)
				restoreProbOrderToSlice(q.ig, pt, prob, b.Prob, minProb, maxProb, &scratch, &scratch2, func(cPT []packedNode, cProb float64) {
					cp := make([]packedNode, len(cPT))
					copy(cp, cPT)
					local = append(local, collected{prob: cProb, baseProb: b.Prob, pt: cp})
				})
			}
			if len(local) > 0 {
				mu.Lock()
				all = append(all, local...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	for i := range all {
		c := &all[i]
		off := q.allocPT(len(c.pt))
		copy(q.ptSlice(off, uint16(len(c.pt))), c.pt)
		q.pushEntry(c.prob, c.baseProb, off, uint16(len(c.pt)))
	}
}

func restoreProbOrderToSlice(
	ig *IndexedGrammar,
	pt []packedNode,
	prob, baseProb, minProb, maxProb float64,
	scratch, scratch2 *[maxPTScratch]packedNode,
	collect func(pt []packedNode, prob float64),
) {
	if prob < minProb {
		return
	}
	if prob <= maxProb {
		if !isParentInQueue(ig, pt, baseProb, maxProb, scratch) {
			collect(pt, prob)
		}
		return
	}
	// walk children without retaining them in a slice
	n := len(pt)
	if n > maxPTScratch {
		// pathological: fall back to heap alloc for this rare structure
		big := make([]packedNode, n)
		for pos := 0; pos < n; pos++ {
			entries := ig.entries(pt[pos].Type)
			if int(pt[pos].Index)+1 >= len(entries) {
				continue
			}
			copy(big, pt)
			big[pos].Index = pt[pos].Index + 1
			if areYouMyChild(ig, big, baseProb, pos, prob, scratch) {
				childProb := ig.findProb(big, baseProb)
				cp := make([]packedNode, n)
				copy(cp, big)
				restoreProbOrderToSlice(ig, cp, childProb, baseProb, minProb, maxProb, scratch, scratch2, collect)
			}
		}
		return
	}

	for pos := 0; pos < n; pos++ {
		entries := ig.entries(pt[pos].Type)
		if int(pt[pos].Index)+1 >= len(entries) {
			continue
		}
		copy(scratch[:n], pt)
		scratch[pos].Index = pt[pos].Index + 1
		child := scratch[:n]
		// scratch2 is used by areYouMyChild so it does not alias child
		if areYouMyChild(ig, child, baseProb, pos, prob, scratch2) {
			childProb := ig.findProb(child, baseProb)
			cp := make([]packedNode, n)
			copy(cp, child)
			restoreProbOrderToSlice(ig, cp, childProb, baseProb, minProb, maxProb, scratch, scratch2, collect)
		}
	}
}

func isParentInQueue(ig *IndexedGrammar, pt []packedNode, baseProb, maxProb float64, scratch *[maxPTScratch]packedNode) bool {
	n := len(pt)
	if n > maxPTScratch {
		buf := make([]packedNode, n)
		for pos, node := range pt {
			if node.Index == 0 {
				continue
			}
			copy(buf, pt)
			buf[pos].Index = node.Index - 1
			if ig.findProb(buf, baseProb) <= maxProb {
				return true
			}
		}
		return false
	}
	copy(scratch[:n], pt)
	for pos, node := range pt {
		if node.Index == 0 {
			continue
		}
		scratch[pos].Index = node.Index - 1
		parentProb := ig.findProb(scratch[:n], baseProb)
		scratch[pos].Index = node.Index
		if parentProb <= maxProb {
			return true
		}
	}
	return false
}

// pops the highest probability item and pushes its children
func (q *PcfgQueue) Next() *PTWork {
	if q.heap.Len() == 0 {
		return nil
	}

	idx := heap.Pop(&q.heap).(int32)
	e := q.entries[idx]
	q.MaxProbability = e.Prob

	parent := q.ptSlice(e.PTOff, e.PTLen)

	// owned copy for workers so the arena slot can be freed after children are pushed
	workPT := make([]packedNode, len(parent))
	copy(workPT, parent)

	q.pushChildren(parent, e.BaseProb, e.Prob)
	q.freeEntry(idx)

	return &PTWork{Prob: e.Prob, PT: workPT}
}

func (q *PcfgQueue) pushChildren(parent []packedNode, baseProb, parentProb float64) {
	var scratch [maxPTScratch]packedNode
	n := len(parent)
	for pos := 0; pos < n; pos++ {
		node := parent[pos]
		entries := q.ig.entries(node.Type)
		if int(node.Index)+1 >= len(entries) {
			continue
		}

		off := q.allocPT(n)
		child := q.ptSlice(off, uint16(n))
		copy(child, parent)
		child[pos].Index = node.Index + 1

		if areYouMyChild(q.ig, child, baseProb, pos, parentProb, &scratch) {
			childProb := q.ig.findProb(child, baseProb)
			q.pushEntry(childProb, baseProb, off, uint16(n))
		} else {
			q.freePT(off, uint16(n))
		}
	}
}

// returns current queue size
func (q *PcfgQueue) QueueSize() int {
	return q.heap.Len()
}

// child is adopted only by its lowest-probability parent (adoption / deadbeat dad)
// scratch is used for temporary parent mutations; child must not alias scratch[:len(child)]
func areYouMyChild(ig *IndexedGrammar, child []packedNode, baseProb float64, parentPos int, parentProb float64, scratch *[maxPTScratch]packedNode) bool {
	n := len(child)
	if n > maxPTScratch {
		buf := make([]packedNode, n)
		copy(buf, child)
		for pos, node := range child {
			if pos == parentPos || node.Index == 0 {
				continue
			}
			buf[pos].Index = node.Index - 1
			newParentProb := ig.findProb(buf, baseProb)
			buf[pos].Index = node.Index
			if newParentProb < parentProb {
				return false
			}
			if newParentProb == parentProb && pos < parentPos {
				return false
			}
		}
		return true
	}

	copy(scratch[:n], child)
	for pos, node := range child {
		if pos == parentPos || node.Index == 0 {
			continue
		}
		scratch[pos].Index = node.Index - 1
		newParentProb := ig.findProb(scratch[:n], baseProb)
		scratch[pos].Index = node.Index
		if newParentProb < parentProb {
			return false
		}
		if newParentProb == parentProb && pos < parentPos {
			return false
		}
	}
	return true
}
