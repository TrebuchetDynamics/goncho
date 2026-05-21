package goncho

import "database/sql"

type retrievalModule struct {
	db              *sql.DB
	workspaceID     string
	observer        string
	recentLimit     int
	peerCardEnabled bool
	dreamEnabled    bool
	sessions        SessionDirectory
}

func (s *Service) retrieval() retrievalModule {
	return retrievalModule{
		db:              s.db,
		workspaceID:     s.workspaceID,
		observer:        s.observer,
		recentLimit:     s.recentLimit,
		peerCardEnabled: s.peerCardEnabled,
		dreamEnabled:    s.dreamEnabled,
		sessions:        s.sessions,
	}
}
