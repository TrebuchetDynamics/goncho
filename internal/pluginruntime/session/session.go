package session

// Message is a plugin memory message awaiting persistence.
type Message struct {
	Role    string
	Content string
	Synced  bool
}

// MemorySession is the plugin-side session payload passed to write flushers.
type MemorySession struct {
	Key             string
	UserPeerID      string
	AssistantPeerID string
	HonchoSessionID string
	Messages        []Message
}

// Flusher persists a plugin memory session.
type Flusher interface {
	FlushPluginSession(MemorySession) error
}

// FlusherFunc adapts a function into a Flusher.
type FlusherFunc func(MemorySession) error

func (f FlusherFunc) FlushPluginSession(session MemorySession) error {
	return f(session)
}
