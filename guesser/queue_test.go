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
