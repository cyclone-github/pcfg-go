package guesser

import "testing"

func TestPcfgQueueAllocPTReusesAndReleasesChunks(t *testing.T) {
	q := &PcfgQueue{ptFree: make([][]uint32, maxPTScratch+1), ptChunks: make([]ptChunk, 1, 2)}

	off1 := q.allocPT(4)
	if off1 != 1 {
		t.Fatalf("expected first allocation at slot 1, got %d", off1)
	}
	if len(q.ptChunks) != 2 {
		t.Fatalf("expected two chunk slots, got %d", len(q.ptChunks))
	}

	q.ptSlice(off1, 4)[0] = packedNode{Type: 1}
	q.freePT(off1, 4)

	if q.ptChunks[off1].nodes != nil {
		t.Fatal("expected freed chunk to release backing memory")
	}

	off2 := q.allocPT(4)
	if off2 != off1 {
		t.Fatalf("expected allocator to reuse freed slot, got %d want %d", off2, off1)
	}
}

func TestPcfgQueueStopsGrowingWhenFull(t *testing.T) {
	q := &PcfgQueue{
		maxEntries: 2,
		entries:    make([]queueEntry, 0, 4),
		ptChunks:   make([]ptChunk, 1, 4),
		ptFree:     make([][]uint32, maxPTScratch+1),
	}
	q.heap.q = q
	q.heap.h = make([]int32, 0, 4)

	if !q.pushEntry(0.1, 1.0, 1, 1) {
		t.Fatal("expected first push to succeed")
	}
	if !q.pushEntry(0.2, 1.0, 1, 1) {
		t.Fatal("expected second push to succeed")
	}
	if q.pushEntry(0.3, 1.0, 1, 1) {
		t.Fatal("expected third push to be rejected when queue is full")
	}
}
