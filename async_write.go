package goncho

import (
	"strconv"
	"strings"
	"sync"
)

const (
	GonchoWriteFlushed     = "goncho_write_flushed"
	GonchoWriteDeferred    = "goncho_write_deferred"
	GonchoWriteQueued      = "goncho_write_queued"
	GonchoWriteFlushFailed = "goncho_write_flush_failed"

	GonchoAsyncEnqueued    = "goncho_async_enqueued"
	GonchoAsyncFlushed     = "goncho_async_flushed"
	GonchoAsyncRetry       = "goncho_async_retry"
	GonchoAsyncFlushFailed = "goncho_async_flush_failed"
	GonchoAsyncShutdown    = "goncho_async_shutdown"
	GonchoAsyncClosed      = "goncho_async_closed"
)

type WriteFrequencyMode string

const (
	WriteFrequencyInvalid WriteFrequencyMode = "invalid"
	WriteFrequencyAsync   WriteFrequencyMode = "async"
	WriteFrequencyTurn    WriteFrequencyMode = "turn"
	WriteFrequencySession WriteFrequencyMode = "session"
	WriteFrequencyEvery   WriteFrequencyMode = "every"
)

type PluginWriteFrequency struct {
	Mode  WriteFrequencyMode
	Every int
	Raw   string
}

func ParsePluginWriteFrequency(raw any) PluginWriteFrequency {
	switch v := raw.(type) {
	case nil:
		return PluginWriteFrequency{Mode: WriteFrequencyAsync, Raw: "async"}
	case int:
		if v > 0 {
			return PluginWriteFrequency{Mode: WriteFrequencyEvery, Every: v, Raw: intToString(v)}
		}
	case int64:
		if v > 0 {
			return PluginWriteFrequency{Mode: WriteFrequencyEvery, Every: int(v), Raw: intToString(int(v))}
		}
	case float64:
		if v == float64(int(v)) && v > 0 {
			return PluginWriteFrequency{Mode: WriteFrequencyEvery, Every: int(v), Raw: intToString(int(v))}
		}
	case string:
		trimmed := stringsLowerTrim(v)
		switch trimmed {
		case "", "async":
			return PluginWriteFrequency{Mode: WriteFrequencyAsync, Raw: "async"}
		case "turn":
			return PluginWriteFrequency{Mode: WriteFrequencyTurn, Raw: "turn"}
		case "session":
			return PluginWriteFrequency{Mode: WriteFrequencySession, Raw: "session"}
		default:
			if n, ok := parsePositiveInt(trimmed); ok {
				return PluginWriteFrequency{Mode: WriteFrequencyEvery, Every: n, Raw: trimmed}
			}
		}
	}
	return PluginWriteFrequency{Mode: WriteFrequencyInvalid, Raw: ""}
}

type PluginMemoryMessage struct {
	Role    string
	Content string
	Synced  bool
}

type PluginMemorySession struct {
	Key             string
	UserPeerID      string
	AssistantPeerID string
	HonchoSessionID string
	Messages        []PluginMemoryMessage
}

type PluginSessionFlusher interface {
	FlushPluginSession(PluginMemorySession) error
}

type PluginSessionFlusherFunc func(PluginMemorySession) error

func (f PluginSessionFlusherFunc) FlushPluginSession(session PluginMemorySession) error {
	return f(session)
}

type PluginWriteRouterConfig struct {
	Frequency   PluginWriteFrequency
	Flusher     PluginSessionFlusher
	AsyncWriter *PluginAsyncWriter
}

type PluginWriteRouter struct {
	frequency   PluginWriteFrequency
	flusher     PluginSessionFlusher
	asyncWriter *PluginAsyncWriter
	turn        int
}

func NewPluginWriteRouter(cfg PluginWriteRouterConfig) *PluginWriteRouter {
	frequency := cfg.Frequency
	if frequency.Mode == "" || frequency.Mode == WriteFrequencyInvalid {
		frequency = PluginWriteFrequency{Mode: WriteFrequencyAsync, Raw: "async"}
	}
	return &PluginWriteRouter{frequency: frequency, flusher: cfg.Flusher, asyncWriter: cfg.AsyncWriter}
}

type PluginWriteResult struct {
	Code     string
	Evidence []string
}

func (r *PluginWriteRouter) Save(session PluginMemorySession) PluginWriteResult {
	if r == nil {
		return PluginWriteResult{Code: GonchoWriteDeferred}
	}
	r.turn++
	switch r.frequency.Mode {
	case WriteFrequencyAsync:
		if r.asyncWriter == nil {
			return PluginWriteResult{Code: GonchoWriteDeferred}
		}
		result := r.asyncWriter.Enqueue(session)
		if result.Code == GonchoAsyncEnqueued {
			return PluginWriteResult{Code: GonchoWriteQueued}
		}
		return PluginWriteResult{Code: GonchoWriteFlushFailed, Evidence: []string{result.Code}}
	case WriteFrequencyTurn:
		return r.flush(session)
	case WriteFrequencySession:
		if r.asyncWriter != nil {
			r.asyncWriter.Cache(session)
		}
		return PluginWriteResult{Code: GonchoWriteDeferred}
	case WriteFrequencyEvery:
		if r.frequency.Every > 0 && r.turn%r.frequency.Every == 0 {
			return r.flush(session)
		}
		return PluginWriteResult{Code: GonchoWriteDeferred}
	default:
		return PluginWriteResult{Code: GonchoWriteDeferred}
	}
}

func (r *PluginWriteRouter) flush(session PluginMemorySession) PluginWriteResult {
	if r.flusher == nil {
		return PluginWriteResult{Code: GonchoWriteFlushFailed}
	}
	if err := r.flusher.FlushPluginSession(session); err != nil {
		return PluginWriteResult{Code: GonchoWriteFlushFailed, Evidence: []string{GonchoWriteFlushFailed}}
	}
	return PluginWriteResult{Code: GonchoWriteFlushed}
}

type PluginAsyncWriter struct {
	mu      sync.Mutex
	flusher PluginSessionFlusher
	queue   []PluginMemorySession
	cache   map[string]PluginMemorySession
	closed  bool
}

func NewPluginAsyncWriter(flusher PluginSessionFlusher) *PluginAsyncWriter {
	return &PluginAsyncWriter{flusher: flusher, cache: map[string]PluginMemorySession{}}
}

type PluginAsyncResult struct {
	Code     string
	Flushed  int
	Pending  int
	Evidence []string
}

func (r PluginAsyncResult) HasEvidence(code string) bool {
	for _, item := range r.Evidence {
		if item == code {
			return true
		}
	}
	return false
}

func (w *PluginAsyncWriter) Enqueue(session PluginMemorySession) PluginAsyncResult {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return PluginAsyncResult{Code: GonchoAsyncClosed, Pending: len(w.queue) + len(w.cache)}
	}
	w.queue = append(w.queue, session)
	return PluginAsyncResult{Code: GonchoAsyncEnqueued, Pending: len(w.queue) + len(w.cache)}
}

func (w *PluginAsyncWriter) Cache(session PluginMemorySession) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cache == nil {
		w.cache = map[string]PluginMemorySession{}
	}
	w.cache[session.Key] = session
}

func (w *PluginAsyncWriter) FlushAll() PluginAsyncResult {
	w.mu.Lock()
	items := append([]PluginMemorySession(nil), w.queue...)
	w.queue = nil
	cacheKeys := make([]string, 0, len(w.cache))
	for key, session := range w.cache {
		cacheKeys = append(cacheKeys, key)
		items = append(items, session)
	}
	w.mu.Unlock()

	var result PluginAsyncResult
	failedQueue := make([]PluginMemorySession, 0)
	failedCache := map[string]PluginMemorySession{}
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
		result.Code = GonchoAsyncFlushFailed
		result.Evidence = appendEvidence(result.Evidence, GonchoAsyncFlushFailed)
		return result
	}
	result.Code = GonchoAsyncFlushed
	return result
}

func (w *PluginAsyncWriter) Shutdown() PluginAsyncResult {
	result := w.FlushAll()
	w.mu.Lock()
	w.closed = true
	result.Code = GonchoAsyncShutdown
	result.Pending = len(w.queue) + len(w.cache)
	w.mu.Unlock()
	return result
}

func (w *PluginAsyncWriter) flushWithRetry(session PluginMemorySession, result *PluginAsyncResult) error {
	if w.flusher == nil {
		return errPluginAsyncNoFlusher{}
	}
	if err := w.flusher.FlushPluginSession(session); err != nil {
		result.Evidence = appendEvidence(result.Evidence, GonchoAsyncRetry)
		return w.flusher.FlushPluginSession(session)
	}
	return nil
}

type errPluginAsyncNoFlusher struct{}

func (errPluginAsyncNoFlusher) Error() string { return "goncho async writer: no flusher configured" }

func appendEvidence(items []string, code string) []string {
	for _, item := range items {
		if item == code {
			return items
		}
	}
	return append(items, code)
}

func stringsLowerTrim(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parsePositiveInt(value string) (int, bool) {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func intToString(n int) string {
	return strconv.Itoa(n)
}
