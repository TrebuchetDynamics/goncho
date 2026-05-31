package pluginruntime

import (
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/asyncwrite"
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/session"
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/writefrequency"
	"github.com/TrebuchetDynamics/goncho/internal/pluginruntime/writerouter"
)

const (
	GonchoWriteFlushed     = writerouter.Flushed
	GonchoWriteDeferred    = writerouter.Deferred
	GonchoWriteQueued      = writerouter.Queued
	GonchoWriteFlushFailed = writerouter.FlushFailed

	GonchoAsyncEnqueued    = asyncwrite.Enqueued
	GonchoAsyncFlushed     = asyncwrite.Flushed
	GonchoAsyncRetry       = asyncwrite.Retry
	GonchoAsyncFlushFailed = asyncwrite.FlushFailed
	GonchoAsyncShutdown    = asyncwrite.Shutdown
	GonchoAsyncClosed      = asyncwrite.Closed
)

type WriteFrequencyMode = writefrequency.Mode

const (
	WriteFrequencyInvalid WriteFrequencyMode = writefrequency.Invalid
	WriteFrequencyAsync   WriteFrequencyMode = writefrequency.Async
	WriteFrequencyTurn    WriteFrequencyMode = writefrequency.Turn
	WriteFrequencySession WriteFrequencyMode = writefrequency.Session
	WriteFrequencyEvery   WriteFrequencyMode = writefrequency.Every
)

type PluginWriteFrequency = writefrequency.Frequency

func ParsePluginWriteFrequency(raw any) PluginWriteFrequency {
	return writefrequency.Parse(raw)
}

type PluginMemoryMessage = session.Message

type PluginMemorySession = session.MemorySession

type PluginSessionFlusher = session.Flusher

type PluginSessionFlusherFunc = session.FlusherFunc

type PluginWriteRouterConfig = writerouter.Config

type PluginWriteRouter = writerouter.Router

func NewPluginWriteRouter(cfg PluginWriteRouterConfig) *PluginWriteRouter {
	return writerouter.New(cfg)
}

type PluginWriteResult = writerouter.Result

type PluginAsyncWriter = asyncwrite.Writer

func NewPluginAsyncWriter(flusher PluginSessionFlusher) *PluginAsyncWriter {
	return asyncwrite.NewWriter(flusher)
}

type PluginAsyncResult = asyncwrite.Result
