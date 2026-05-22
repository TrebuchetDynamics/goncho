package goncho

import "database/sql"

type lifecycleModule struct {
	db             *sql.DB
	workspaceID    string
	maxMessageSize int
}

func (s *Service) lifecycle() lifecycleModule {
	return lifecycleModule{
		db:             s.db,
		workspaceID:    s.workspaceID,
		maxMessageSize: s.maxMessageSize,
	}
}
