package goncho

import pluginruntime "github.com/TrebuchetDynamics/goncho/internal/pluginruntime"

const (
	GonchoWriteFlushed     = pluginruntime.GonchoWriteFlushed
	GonchoWriteDeferred    = pluginruntime.GonchoWriteDeferred
	GonchoWriteQueued      = pluginruntime.GonchoWriteQueued
	GonchoWriteFlushFailed = pluginruntime.GonchoWriteFlushFailed

	GonchoAsyncEnqueued    = pluginruntime.GonchoAsyncEnqueued
	GonchoAsyncFlushed     = pluginruntime.GonchoAsyncFlushed
	GonchoAsyncRetry       = pluginruntime.GonchoAsyncRetry
	GonchoAsyncFlushFailed = pluginruntime.GonchoAsyncFlushFailed
	GonchoAsyncShutdown    = pluginruntime.GonchoAsyncShutdown
	GonchoAsyncClosed      = pluginruntime.GonchoAsyncClosed
)

type WriteFrequencyMode = pluginruntime.WriteFrequencyMode

const (
	WriteFrequencyInvalid WriteFrequencyMode = pluginruntime.WriteFrequencyInvalid
	WriteFrequencyAsync   WriteFrequencyMode = pluginruntime.WriteFrequencyAsync
	WriteFrequencyTurn    WriteFrequencyMode = pluginruntime.WriteFrequencyTurn
	WriteFrequencySession WriteFrequencyMode = pluginruntime.WriteFrequencySession
	WriteFrequencyEvery   WriteFrequencyMode = pluginruntime.WriteFrequencyEvery
)

type PluginWriteFrequency = pluginruntime.PluginWriteFrequency

type PluginMemoryMessage = pluginruntime.PluginMemoryMessage

type PluginMemorySession = pluginruntime.PluginMemorySession

type PluginSessionFlusher = pluginruntime.PluginSessionFlusher

type PluginSessionFlusherFunc = pluginruntime.PluginSessionFlusherFunc

type PluginWriteRouterConfig = pluginruntime.PluginWriteRouterConfig

type PluginWriteRouter = pluginruntime.PluginWriteRouter

type PluginWriteResult = pluginruntime.PluginWriteResult

type PluginAsyncWriter = pluginruntime.PluginAsyncWriter

type PluginAsyncResult = pluginruntime.PluginAsyncResult

func ParsePluginWriteFrequency(raw any) PluginWriteFrequency {
	return pluginruntime.ParsePluginWriteFrequency(raw)
}

func NewPluginWriteRouter(cfg PluginWriteRouterConfig) *PluginWriteRouter {
	return pluginruntime.NewPluginWriteRouter(cfg)
}

func NewPluginAsyncWriter(flusher PluginSessionFlusher) *PluginAsyncWriter {
	return pluginruntime.NewPluginAsyncWriter(flusher)
}
