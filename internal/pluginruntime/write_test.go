package pluginruntime_test

import (
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime"
)

func TestGonchoAsyncWriterLifecycle(t *testing.T) {
	attempts := 0
	writer := pluginruntime.NewPluginAsyncWriter(pluginruntime.PluginSessionFlusherFunc(func(session pluginruntime.PluginMemorySession) error {
		attempts++
		if session.Key == "queued" && attempts == 1 {
			return errors.New("temporary store failure with /tmp/private/path")
		}
		return nil
	}))

	queued := pluginruntime.PluginMemorySession{Key: "queued", Messages: []pluginruntime.PluginMemoryMessage{{Role: "user", Content: "hello"}}}
	cached := pluginruntime.PluginMemorySession{Key: "cached", Messages: []pluginruntime.PluginMemoryMessage{{Role: "assistant", Content: "hi"}}}
	if got := writer.Enqueue(queued); got.Code != pluginruntime.GonchoAsyncEnqueued {
		t.Fatalf("Enqueue code = %q, want %q", got.Code, pluginruntime.GonchoAsyncEnqueued)
	}
	writer.Cache(cached)

	result := writer.FlushAll()
	if result.Code != pluginruntime.GonchoAsyncFlushed {
		t.Fatalf("FlushAll code = %q, want %q", result.Code, pluginruntime.GonchoAsyncFlushed)
	}
	if result.Flushed != 2 || result.Pending != 0 {
		t.Fatalf("FlushAll flushed/pending = %d/%d, want 2/0", result.Flushed, result.Pending)
	}
	if !result.HasEvidence(pluginruntime.GonchoAsyncRetry) {
		t.Fatalf("FlushAll evidence = %+v, want %s", result.Evidence, pluginruntime.GonchoAsyncRetry)
	}
	if attempts != 3 {
		t.Fatalf("flush attempts = %d, want queued retry plus cached flush", attempts)
	}

	shutdown := writer.Shutdown()
	if shutdown.Code != pluginruntime.GonchoAsyncShutdown || shutdown.Pending != 0 {
		t.Fatalf("Shutdown = %+v, want shutdown with no pending writes", shutdown)
	}
	if got := writer.Enqueue(pluginruntime.PluginMemorySession{Key: "after-shutdown"}); got.Code != pluginruntime.GonchoAsyncClosed {
		t.Fatalf("Enqueue after shutdown = %+v, want %s", got, pluginruntime.GonchoAsyncClosed)
	}
}

func TestGonchoAsyncWriterFlushAllKeepsFailedItems(t *testing.T) {
	writer := pluginruntime.NewPluginAsyncWriter(pluginruntime.PluginSessionFlusherFunc(func(pluginruntime.PluginMemorySession) error {
		return errors.New("still unavailable")
	}))
	_ = writer.Enqueue(pluginruntime.PluginMemorySession{Key: "queued"})

	result := writer.FlushAll()
	if result.Code != pluginruntime.GonchoAsyncFlushFailed {
		t.Fatalf("FlushAll code = %q, want %q", result.Code, pluginruntime.GonchoAsyncFlushFailed)
	}
	if result.Pending != 1 {
		t.Fatalf("Pending = %d, want failed write retained", result.Pending)
	}
	if !result.HasEvidence(pluginruntime.GonchoAsyncFlushFailed) {
		t.Fatalf("Evidence = %+v, want %s", result.Evidence, pluginruntime.GonchoAsyncFlushFailed)
	}
}
