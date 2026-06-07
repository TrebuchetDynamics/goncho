package plugins

import (
	"fmt"
	"sort"
	"sync"
	"testing"
)

func TestPluginRuntimePublicFacadeRoutesAsyncWrites(t *testing.T) {
	flushed := 0
	writer := NewPluginAsyncWriter(PluginSessionFlusherFunc(func(session PluginMemorySession) error {
		flushed++
		if session.Key != "session-1" {
			t.Fatalf("session key = %q, want session-1", session.Key)
		}
		return nil
	}))

	router := NewPluginWriteRouter(PluginWriteRouterConfig{
		Frequency:   ParsePluginWriteFrequency("async"),
		AsyncWriter: writer,
	})
	if got := router.Save(PluginMemorySession{Key: "session-1", Messages: []PluginMemoryMessage{{Role: "user", Content: "hello"}}}); got.Code != GonchoWriteQueued {
		t.Fatalf("Save code = %q, want %q", got.Code, GonchoWriteQueued)
	}
	if got := writer.FlushAll(); got.Code != GonchoAsyncFlushed || got.Flushed != 1 || flushed != 1 {
		t.Fatalf("FlushAll = %+v flushed=%d, want one flushed async write", got, flushed)
	}
}

func TestPluginAsyncWriterFlushesQueuedSessionsFIFO(t *testing.T) {
	var flushed []string
	writer := NewPluginAsyncWriter(PluginSessionFlusherFunc(func(session PluginMemorySession) error {
		flushed = append(flushed, session.Key)
		return nil
	}))

	for _, key := range []string{"session-1", "session-2", "session-3"} {
		if got := writer.Enqueue(PluginMemorySession{Key: key}); got.Code != GonchoAsyncEnqueued {
			t.Fatalf("Enqueue(%q) = %+v, want enqueued", key, got)
		}
	}

	if got := writer.FlushAll(); got.Code != GonchoAsyncFlushed || got.Flushed != 3 || got.Pending != 0 {
		t.Fatalf("FlushAll = %+v, want three flushed and none pending", got)
	}
	want := []string{"session-1", "session-2", "session-3"}
	if fmt.Sprint(flushed) != fmt.Sprint(want) {
		t.Fatalf("flush order = %v, want FIFO %v", flushed, want)
	}
	if got := writer.FlushAll(); got.Code != GonchoAsyncFlushed || got.Flushed != 0 || got.Pending != 0 {
		t.Fatalf("second FlushAll = %+v, want idempotent empty flush", got)
	}
}

func TestPluginAsyncWriterConcurrentEnqueueFlushesEachSessionOnce(t *testing.T) {
	const sessions = 32
	var mu sync.Mutex
	flushed := map[string]int{}
	writer := NewPluginAsyncWriter(PluginSessionFlusherFunc(func(session PluginMemorySession) error {
		mu.Lock()
		defer mu.Unlock()
		flushed[session.Key]++
		return nil
	}))

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < sessions; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			writer.Enqueue(PluginMemorySession{Key: fmt.Sprintf("session-%02d", i)})
		}()
	}
	close(start)
	wg.Wait()

	if got := writer.FlushAll(); got.Code != GonchoAsyncFlushed || got.Flushed != sessions || got.Pending != 0 {
		t.Fatalf("FlushAll = %+v, want every concurrent enqueue flushed once", got)
	}
	if len(flushed) != sessions {
		keys := make([]string, 0, len(flushed))
		for key := range flushed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		t.Fatalf("flushed keys len = %d, want %d: %v", len(flushed), sessions, keys)
	}
	for i := 0; i < sessions; i++ {
		key := fmt.Sprintf("session-%02d", i)
		if flushed[key] != 1 {
			t.Fatalf("flushed[%s] = %d, want exactly once", key, flushed[key])
		}
	}
}
