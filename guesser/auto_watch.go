package guesser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

const autoPollInterval = 300 * time.Second

type autoWatcher struct {
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

func startAutoWatcher(ctx context.Context, path string, debug bool, interval time.Duration, minCount int, classify func(string) (string, bool)) (<-chan autoBatch, error) {
	if interval <= 0 {
		interval = autoPollInterval
	}
	if minCount <= 0 {
		minCount = autoMinFounds
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("opening -auto founds file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("-auto path is a directory: %s", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening -auto founds file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("opening -auto founds file: %w", err)
	}

	if classify == nil {
		classify = classifyPassword
	}

	w := &autoWatcher{
		path:     path,
		interval: interval,
		minCount: minCount,
		debug:    debug,
		offset:   info.Size(),
		pending:  make(map[string]int),
		classify: classify,
	}

	ch := make(chan autoBatch, 2)
	go w.loop(ctx, ch)
	return ch, nil
}

func (w *autoWatcher) loop(ctx context.Context, ch chan<- autoBatch) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.poll(); err != nil {
				fmt.Fprintf(os.Stderr, "[auto] read error: %v\n", err)
				continue
			}
			if w.pendingN < w.minCount {
				if w.debug {
					fmt.Fprintf(os.Stderr, "[auto] pending founds: %d (need %d)\n", w.pendingN, w.minCount)
				}
				continue
			}
			batch := autoBatch{counts: w.pending, n: w.pendingN}
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

func (w *autoWatcher) poll() error {
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
			fmt.Fprintf(os.Stderr, "[auto] founds file truncated; ignoring existing contents\n")
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

func (w *autoWatcher) ingestLine(line []byte) {
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
