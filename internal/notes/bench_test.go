package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

var benchData struct {
	once      sync.Once
	cleanOnce sync.Once
	root      string
	total     int
	err       error
}

// prepareBenchData builds (or reuses) a synthetic tree of note files.
// You can override the note count via BENCH_NOTES_COUNT to make it cheaper
// on constrained environments.
func prepareBenchData(b *testing.B) (string, int) {
	b.Helper()
	benchData.once.Do(func() {
		total := 90_000
		if v := os.Getenv("BENCH_NOTES_COUNT"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				total = n
			}
		}
		benchData.total = total

		root, err := os.MkdirTemp("", "thumbr-bench-")
		if err != nil {
			benchData.err = fmt.Errorf("tempdir: %w", err)
			return
		}
		benchData.root = root

		for i := 0; i < total; i++ {
			dir := filepath.Join(root, fmt.Sprintf("dir-%03d", i/1000))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				benchData.err = fmt.Errorf("mkdir: %w", err)
				return
			}
			path := filepath.Join(dir, fmt.Sprintf("note-%05d.txt", i))
			if err := os.WriteFile(path, []byte("line\n"), 0o644); err != nil {
				benchData.err = fmt.Errorf("write: %w", err)
				return
			}
		}
	})

	if benchData.err != nil {
		b.Fatalf("bench setup: %v", benchData.err)
	}
	return benchData.root, benchData.total
}

// BenchmarkLoadCardsFromDir90k measures discovery + sort time over a large tree.
func BenchmarkLoadCardsFromDir90k(b *testing.B) {
	root, total := prepareBenchData(b)

	opts := LoadOptions{}

	b.ReportAllocs()
	b.ResetTimer()

	start := time.Now()
	for i := 0; i < b.N; i++ {
		cards, err := LoadCardsFromDir(root, opts)
		if err != nil {
			b.Fatalf("load: %v", err)
		}
		if len(cards) != total {
			b.Fatalf("expected %d cards, got %d", total, len(cards))
		}
	}
	elapsed := time.Since(start)
	// Explicitly emit time/op since some toolchains hide it when using custom benchtime.
	if b.N > 0 {
		nsPerOp := float64(elapsed.Nanoseconds()) / float64(b.N)
		b.ReportMetric(nsPerOp, "ns/op")
		secPerOp := float64(elapsed) / float64(time.Second) / float64(b.N)
		b.ReportMetric(secPerOp, "s/op")
	}
}
