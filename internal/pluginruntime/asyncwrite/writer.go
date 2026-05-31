package asyncwrite

import (
	"sync"

	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/evidence"
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/session"
)

const (
	Enqueued    = "goncho_async_enqueued"
	Flushed     = "goncho_async_flushed"
	Retry       = "goncho_async_retry"
	FlushFailed = "goncho_async_flush_failed"
	Shutdown    = "goncho_async_shutdown"
	Closed      = "goncho_async_closed"
)

type Result struct {
	Code     string
	Flushed  int
	Pending  int
	Evidence []string
}

func (r Result) HasEvidence(code string) bool {
	return evidence.Has(r.Evidence, code)
}

type Writer struct {
	mu      sync.Mutex
	flusher session.Flusher
	queue   []session.MemorySession
	cache   map[string]session.MemorySession
	closed  bool
}

func NewWriter(flusher session.Flusher) *Writer {
	return &Writer{flusher: flusher, cache: map[string]session.MemorySession{}}
}

func (w *Writer) Enqueue(session session.MemorySession) Result {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return Result{Code: Closed, Pending: len(w.queue) + len(w.cache)}
	}
	w.queue = append(w.queue, session)
	return Result{Code: Enqueued, Pending: len(w.queue) + len(w.cache)}
}

func (w *Writer) Cache(item session.MemorySession) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cache == nil {
		w.cache = map[string]session.MemorySession{}
	}
	w.cache[item.Key] = item
}

func (w *Writer) FlushAll() Result {
	w.mu.Lock()
	items := append([]session.MemorySession(nil), w.queue...)
	w.queue = nil
	cacheKeys := make([]string, 0, len(w.cache))
	for key, session := range w.cache {
		cacheKeys = append(cacheKeys, key)
		items = append(items, session)
	}
	w.mu.Unlock()

	var result Result
	failedQueue := make([]session.MemorySession, 0)
	failedCache := map[string]session.MemorySession{}
	for i, session := range items {
		if err := w.flushWithRetry(session, &result); err != nil {
			if i < len(items)-len(cacheKeys) {
				failedQueue = append(failedQueue, session)
			} else {
				failedCache[session.Key] = session
			}
			continue
		}
		result.Flushed++
	}

	w.mu.Lock()
	if len(failedQueue) > 0 {
		w.queue = append(failedQueue, w.queue...)
	}
	for _, key := range cacheKeys {
		delete(w.cache, key)
	}
	for key, session := range failedCache {
		w.cache[key] = session
	}
	result.Pending = len(w.queue) + len(w.cache)
	w.mu.Unlock()

	if result.Pending > 0 {
		result.Code = FlushFailed
		result.Evidence = evidence.Append(result.Evidence, FlushFailed)
		return result
	}
	result.Code = Flushed
	return result
}

func (w *Writer) Shutdown() Result {
	result := w.FlushAll()
	w.mu.Lock()
	w.closed = true
	result.Code = Shutdown
	result.Pending = len(w.queue) + len(w.cache)
	w.mu.Unlock()
	return result
}

func (w *Writer) flushWithRetry(session session.MemorySession, result *Result) error {
	if w.flusher == nil {
		return errNoFlusher{}
	}
	if err := w.flusher.FlushPluginSession(session); err != nil {
		result.Evidence = evidence.Append(result.Evidence, Retry)
		return w.flusher.FlushPluginSession(session)
	}
	return nil
}

type errNoFlusher struct{}

func (errNoFlusher) Error() string { return "goncho async writer: no flusher configured" }
