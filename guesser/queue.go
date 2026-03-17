package guesser

import (
	"container/heap"
	"sync/atomic"

	pcfg "github.com/cyclone-github/pcfg-go/shared"
)

// wrap PTItem for the priority queue
type queueItem struct {
	item  pcfg.PTItem
	seq   int64 // insertion order for stable tie-breaking
	index int   // heap index
}

// priorityQueue implements heap.Interface for max-probability ordering
type priorityQueue []*queueItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].item.Prob != pq[j].item.Prob {
		return pq[i].item.Prob > pq[j].item.Prob
	}
	// Stable tie-breaking: earlier insertions come first
	return pq[i].seq < pq[j].seq
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*queueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// manage priority queue for guess generation
type PcfgQueue struct {
	pq             priorityQueue
	grammar        pcfg.Grammar
	base           []pcfg.BaseStructure
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
	q := &PcfgQueue{
		grammar:        grammar,
		base:           base,
		MaxProbability: maxProb,
		MinProbability: minProb,
	}

	heap.Init(&q.pq)

	for _, b := range base {
		pt := make([]pcfg.PTNode, len(b.Replacements))
		for i, r := range b.Replacements {
			pt[i] = pcfg.PTNode{Type: r, Index: 0}
		}

		prob := findProb(grammar, pt, b.Prob)
		item := pcfg.PTItem{
			Prob:     prob,
			PT:       pt,
			BaseProb: b.Prob,
		}
		if minProb > 0 || maxProb < 1 {
			restoreProbOrder(q, grammar, &item, minProb, maxProb)
		} else {
			seq := q.seqCounter.Add(1)
			heap.Push(&q.pq, &queueItem{item: item, seq: seq})
		}
	}

	return q
}

// recursively restores queue items in [minProb, maxProb]
func restoreProbOrder(q *PcfgQueue, grammar pcfg.Grammar, ptItem *pcfg.PTItem, minProb, maxProb float64) {
	prob := ptItem.Prob
	if prob < minProb {
		return
	}
	if prob <= maxProb {
		if !isParentInQueue(grammar, ptItem, maxProb) {
			seq := q.seqCounter.Add(1)
			heap.Push(&q.pq, &queueItem{item: *ptItem, seq: seq})
		}
		return
	}
	children := findChildren(grammar, ptItem)
	for _, child := range children {
		restoreProbOrder(q, grammar, &child, minProb, maxProb)
	}
}

// returns true if any "parent" of ptItem would be in the queue
func isParentInQueue(grammar pcfg.Grammar, ptItem *pcfg.PTItem, maxProb float64) bool {
	for pos, node := range ptItem.PT {
		if node.Index == 0 {
			continue
		}
		parent := make([]pcfg.PTNode, len(ptItem.PT))
		copy(parent, ptItem.PT)
		parent[pos] = pcfg.PTNode{Type: parent[pos].Type, Index: parent[pos].Index - 1}
		parentProb := findProb(grammar, parent, ptItem.BaseProb)
		if parentProb <= maxProb {
			return true
		}
	}
	return false
}

// pops the highest probability item and pushes its children
func (q *PcfgQueue) Next() *pcfg.PTItem {
	if q.pq.Len() == 0 {
		return nil
	}

	qi := heap.Pop(&q.pq).(*queueItem)
	q.MaxProbability = qi.item.Prob

	children := findChildren(q.grammar, &qi.item)
	for _, child := range children {
		seq := q.seqCounter.Add(1)
		heap.Push(&q.pq, &queueItem{item: child, seq: seq})
	}

	return &qi.item
}

// returns current queue size
func (q *PcfgQueue) QueueSize() int {
	return q.pq.Len()
}

func findProb(grammar pcfg.Grammar, pt []pcfg.PTNode, baseProb float64) float64 {
	prob := baseProb
	for _, node := range pt {
		entries := grammar[node.Type]
		if node.Index < len(entries) {
			prob *= entries[node.Index].Prob
		}
	}
	return prob
}

func findChildren(grammar pcfg.Grammar, ptItem *pcfg.PTItem) []pcfg.PTItem {
	parentPT := ptItem.PT
	var children []pcfg.PTItem

	for pos, node := range parentPT {
		entries := grammar[node.Type]
		if len(entries) == node.Index+1 {
			continue
		}

		child := make([]pcfg.PTNode, len(parentPT))
		copy(child, parentPT)
		child[pos] = pcfg.PTNode{Type: child[pos].Type, Index: child[pos].Index + 1}

		if areYouMyChild(grammar, child, ptItem.BaseProb, pos, ptItem.Prob) {
			childProb := findProb(grammar, child, ptItem.BaseProb)
			children = append(children, pcfg.PTItem{
				PT:       child,
				BaseProb: ptItem.BaseProb,
				Prob:     childProb,
			})
		}
	}

	return children
}

func areYouMyChild(grammar pcfg.Grammar, child []pcfg.PTNode, baseProb float64, parentPos int, parentProb float64) bool {
	for pos, node := range child {
		if pos == parentPos {
			continue
		}
		if node.Index == 0 {
			continue
		}

		newParent := make([]pcfg.PTNode, len(child))
		copy(newParent, child)
		newParent[pos] = pcfg.PTNode{Type: newParent[pos].Type, Index: newParent[pos].Index - 1}

		newParentProb := findProb(grammar, newParent, baseProb)
		if newParentProb < parentProb {
			return false
		}
		if newParentProb == parentProb && pos < parentPos {
			return false
		}
	}
	return true
}
