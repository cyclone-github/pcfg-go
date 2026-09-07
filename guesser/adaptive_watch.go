package guesser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

const adaptivePollInterval = 300 * time.Second

type adaptiveWatcher struct {
	path     string
	interval time.Duration
	minCount int
	debug    bool
	offset   int64
	partial  []byte
	pending  map[string]int
	pendingN int
	classify func(string) (string, bool)
}

func startAdaptiveWatcher(ctx context.Context, path string, debug bool, interval time.Duration, minCount int, classify func(string) (string, bool)) (<-chan adaptiveBatch, error) {
	if interval <= 0 {
		interval = adaptivePollInterval
	}
	if minCount <= 0 {
		minCount = adaptiveMinFounds
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("opening -adaptive founds file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("-adaptive path is a directory: %s", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening -adaptive founds file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("opening -adaptive founds file: %w", err)
	}

	if classify == nil {
		classify = classifyPassword
	}

	w := &adaptiveWatcher{
		path:     path,
		interval: interval,
		minCount: minCount,
		debug:    debug,
		offset:   info.Size(),
		pending:  make(map[string]int),
		classify: classify,
	}

	ch := make(chan adaptiveBatch, 2)
	go w.loop(ctx, ch)
	return ch, nil
}

func (w *adaptiveWatcher) loop(ctx context.Context, ch chan<- adaptiveBatch) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.poll(); err != nil {
				fmt.Fprintf(os.Stderr, "[adaptive] read error: %v\n", err)
				continue
			}
			if w.pendingN < w.minCount {
				if w.debug {
					fmt.Fprintf(os.Stderr, "[adaptive] pending founds: %d (need %d)\n", w.pendingN, w.minCount)
				}
				continue
			}
			batch := adaptiveBatch{counts: w.pending, n: w.pendingN}
			select {
			case ch <- batch:
				w.pending = make(map[string]int)
				w.pendingN = 0
			case <-ctx.Done():
				return
			}
		}
	}
}

func (w *adaptiveWatcher) poll() error {
	f, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	if size < w.offset {
		// file truncated/replaced; skip existing contents
		if w.debug {
			fmt.Fprintf(os.Stderr, "[adaptive] founds file truncated; ignoring existing contents\n")
		}
		w.offset = size
		w.partial = w.partial[:0]
		return nil
	}
	if size == w.offset && len(w.partial) == 0 {
		return nil
	}

	if _, err := f.Seek(w.offset, io.SeekStart); err != nil {
		return err
	}

	buf := make([]byte, size-w.offset)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return err
	}
	buf = buf[:n]
	w.offset += int64(n)

	data := buf
	if len(w.partial) > 0 {
		joined := make([]byte, 0, len(w.partial)+len(buf))
		joined = append(joined, w.partial...)
		joined = append(joined, buf...)
		data = joined
		w.partial = w.partial[:0]
	}

	for {
		nl := bytes.IndexByte(data, '\n')
		if nl < 0 {
			if len(data) > 0 {
				w.partial = append(w.partial[:0], data...)
			}
			break
		}
		line := stripEOL(data[:nl])
		data = data[nl+1:]
		w.ingestLine(line)
	}
	return nil
}

func (w *adaptiveWatcher) ingestLine(line []byte) {
	plain, ok := extractFoundsPlaintext(line)
	if !ok {
		return
	}
	pw, ok := decodeFoundsPlaintext(plain)
	if !ok {
		return
	}
	key, ok := w.classify(pw)
	if !ok {
		return
	}
	w.pending[key]++
	w.pendingN++
}
