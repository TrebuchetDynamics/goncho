package gormes

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/goncho/memory"
)

const (
	DefaultWorkspaceID = "gormes"
	DefaultObserverID  = "gormes"
)

type Config struct {
	DatabasePath       string
	WorkspaceID        string
	ObserverID         string
	RecentMessages     int
	MemoryMarkdownPath string
	Logger             *slog.Logger
}

type Runtime struct {
	Store        *memory.SqliteStore
	DB           *sql.DB
	Service      *goncho.Service
	ContextTool  *goncho.GonchoContextTool
	SearchTool   *goncho.GonchoSearchTool
	RememberTool *goncho.GonchoRememberTool
	ReviewTool   *goncho.ReviewTool
	HandoffTool  *goncho.GonchoHandoffTool
	config       Config
}

type Status struct {
	Ready        bool     `json:"ready"`
	WorkspaceID  string   `json:"workspace_id"`
	ObserverID   string   `json:"observer_id"`
	DatabasePath string   `json:"database_path"`
	ToolNames    []string `json:"tool_names"`
}

func Open(ctx context.Context, cfg Config) (*Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg = cfg.withDefaults()
	if strings.TrimSpace(cfg.DatabasePath) == "" {
		return nil, fmt.Errorf("gormes goncho: database path is required")
	}
	store, err := memory.OpenSqlite(cfg.DatabasePath, 0, cfg.Logger)
	if err != nil {
		return nil, err
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		_ = store.Close(ctx)
		return nil, fmt.Errorf("gormes goncho: run migrations: %w", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{
		WorkspaceID:    cfg.WorkspaceID,
		ObserverPeerID: cfg.ObserverID,
		RecentMessages: cfg.RecentMessages,
	}, cfg.Logger)
	handoffStore := goncho.NewLocalMarkdownMemoryStore(store.DB(), goncho.LocalMarkdownMemoryConfig{
		Path:        cfg.MemoryMarkdownPath,
		AgentID:     "agent:" + cfg.ObserverID,
		WorkspaceID: cfg.WorkspaceID,
		PeerID:      "operator",
		SessionID:   "startup",
	})
	return &Runtime{
		Store:        store,
		DB:           store.DB(),
		Service:      svc,
		ContextTool:  goncho.NewGonchoContextTool(svc),
		SearchTool:   goncho.NewGonchoSearchTool(svc),
		RememberTool: goncho.NewGonchoRememberTool(svc),
		ReviewTool:   goncho.NewReviewTool(svc),
		HandoffTool:  goncho.NewGonchoHandoffTool(handoffStore),
		config:       cfg,
	}, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil || r.Store == nil {
		return nil
	}
	return r.Store.Close(ctx)
}

func (r *Runtime) Status() Status {
	if r == nil {
		return Status{}
	}
	return Status{
		Ready:        r.Service != nil && r.DB != nil,
		WorkspaceID:  r.config.WorkspaceID,
		ObserverID:   r.config.ObserverID,
		DatabasePath: r.config.DatabasePath,
		ToolNames:    []string{r.ContextTool.Name(), r.SearchTool.Name(), r.RememberTool.Name(), r.ReviewTool.Name(), r.HandoffTool.Name()},
	}
}

func (c Config) withDefaults() Config {
	out := c
	if strings.TrimSpace(out.WorkspaceID) == "" {
		out.WorkspaceID = DefaultWorkspaceID
	}
	if strings.TrimSpace(out.ObserverID) == "" {
		out.ObserverID = DefaultObserverID
	}
	if out.RecentMessages <= 0 {
		out.RecentMessages = 8
	}
	if strings.TrimSpace(out.MemoryMarkdownPath) == "" && strings.TrimSpace(out.DatabasePath) != "" {
		out.MemoryMarkdownPath = filepath.Join(filepath.Dir(out.DatabasePath), "GONCHO_MEMORY.md")
	}
	return out
}
