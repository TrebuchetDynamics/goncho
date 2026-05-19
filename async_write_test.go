package goncho

import (
	"errors"
	"testing"
)

func TestGonchoAsyncWriterLifecycle(t *testing.T) {
	attempts := 0
	writer := NewPluginAsyncWriter(PluginSessionFlusherFunc(func(session PluginMemorySession) error {
		attempts++
		if session.Key == "queued" && attempts == 1 {
			return errors.New("temporary store failure with /tmp/private/path")
		}
		return nil
	}))

	queued := PluginMemorySession{Key: "queued", Messages: []PluginMemoryMessage{{Role: "user", Content: "hello"}}}
	cached := PluginMemorySession{Key: "cached", Messages: []PluginMemoryMessage{{Role: "assistant", Content: "hi"}}}
	if got := writer.Enqueue(queued); got.Code != GonchoAsyncEnqueued {
		t.Fatalf("Enqueue code = %q, want %q", got.Code, GonchoAsyncEnqueued)
	}
	writer.Cache(cached)

	result := writer.FlushAll()
	if result.Code != GonchoAsyncFlushed {
		t.Fatalf("FlushAll code = %q, want %q", result.Code, GonchoAsyncFlushed)
	}
	if result.Flushed != 2 || result.Pending != 0 {
		t.Fatalf("FlushAll flushed/pending = %d/%d, want 2/0", result.Flushed, result.Pending)
	}
	if !result.HasEvidence(GonchoAsyncRetry) {
		t.Fatalf("FlushAll evidence = %+v, want %s", result.Evidence, GonchoAsyncRetry)
	}
	if attempts != 3 {
		t.Fatalf("flush attempts = %d, want queued retry plus cached flush", attempts)
	}

	shutdown := writer.Shutdown()
	if shutdown.Code != GonchoAsyncShutdown || shutdown.Pending != 0 {
		t.Fatalf("Shutdown = %+v, want shutdown with no pending writes", shutdown)
	}
	if got := writer.Enqueue(PluginMemorySession{Key: "after-shutdown"}); got.Code != GonchoAsyncClosed {
		t.Fatalf("Enqueue after shutdown = %+v, want %s", got, GonchoAsyncClosed)
	}
}

func TestGonchoAsyncWriterFlushAllKeepsFailedItems(t *testing.T) {
	writer := NewPluginAsyncWriter(PluginSessionFlusherFunc(func(PluginMemorySession) error {
		return errors.New("still unavailable")
	}))
	_ = writer.Enqueue(PluginMemorySession{Key: "queued"})

	result := writer.FlushAll()
	if result.Code != GonchoAsyncFlushFailed {
		t.Fatalf("FlushAll code = %q, want %q", result.Code, GonchoAsyncFlushFailed)
	}
	if result.Pending != 1 {
		t.Fatalf("Pending = %d, want failed write retained", result.Pending)
	}
	if !result.HasEvidence(GonchoAsyncFlushFailed) {
		t.Fatalf("Evidence = %+v, want %s", result.Evidence, GonchoAsyncFlushFailed)
	}
}
