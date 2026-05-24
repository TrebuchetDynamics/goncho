package goncho

import "testing"

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
