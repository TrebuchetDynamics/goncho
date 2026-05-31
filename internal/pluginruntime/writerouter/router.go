package writerouter

import (
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/asyncwrite"
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/session"
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/writefrequency"
)

const (
	Flushed     = "goncho_write_flushed"
	Deferred    = "goncho_write_deferred"
	Queued      = "goncho_write_queued"
	FlushFailed = "goncho_write_flush_failed"
)

type Config struct {
	Frequency   writefrequency.Frequency
	Flusher     session.Flusher
	AsyncWriter *asyncwrite.Writer
}

type Router struct {
	frequency   writefrequency.Frequency
	flusher     session.Flusher
	asyncWriter *asyncwrite.Writer
	turn        int
}

func New(cfg Config) *Router {
	frequency := cfg.Frequency
	if frequency.Mode == "" || frequency.Mode == writefrequency.Invalid {
		frequency = writefrequency.Frequency{Mode: writefrequency.Async, Raw: "async"}
	}
	return &Router{frequency: frequency, flusher: cfg.Flusher, asyncWriter: cfg.AsyncWriter}
}

type Result struct {
	Code     string
	Evidence []string
}

func (r *Router) Save(session session.MemorySession) Result {
	if r == nil {
		return Result{Code: Deferred}
	}
	r.turn++
	switch r.frequency.Mode {
	case writefrequency.Async:
		if r.asyncWriter == nil {
			return Result{Code: Deferred}
		}
		result := r.asyncWriter.Enqueue(session)
		if result.Code == asyncwrite.Enqueued {
			return Result{Code: Queued}
		}
		return Result{Code: FlushFailed, Evidence: []string{result.Code}}
	case writefrequency.Turn:
		return r.flush(session)
	case writefrequency.Session:
		if r.asyncWriter != nil {
			r.asyncWriter.Cache(session)
		}
		return Result{Code: Deferred}
	case writefrequency.Every:
		if r.frequency.Every > 0 && r.turn%r.frequency.Every == 0 {
			return r.flush(session)
		}
		return Result{Code: Deferred}
	default:
		return Result{Code: Deferred}
	}
}

func (r *Router) flush(session session.MemorySession) Result {
	if r.flusher == nil {
		return Result{Code: FlushFailed}
	}
	if err := r.flusher.FlushPluginSession(session); err != nil {
		return Result{Code: FlushFailed, Evidence: []string{FlushFailed}}
	}
	return Result{Code: Flushed}
}
