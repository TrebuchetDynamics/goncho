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
	"github.com/TrebuchetDynamics/goncho/toolmeta"
)

const (
	DefaultWorkspaceID = goncho.DefaultWorkspaceID
	DefaultObserverID  = goncho.DefaultObserverPeerID
)

type Config struct {
	DatabasePath       string
	ProfilesDirectory  string
	ProfileID          string
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
	RecallTool   *goncho.GonchoRecallTool
	RememberTool *goncho.GonchoRememberTool
	ReviewTool   *goncho.ReviewTool
	HandoffTool  *goncho.GonchoHandoffTool
	config       Config
}

type Status struct {
	Ready              bool                     `json:"ready"`
	WorkspaceID        string                   `json:"workspace_id"`
	ObserverID         string                   `json:"observer_id"`
	ProfileID          string                   `json:"profile_id,omitempty"`
	ProfilesDirectory  string                   `json:"profiles_directory,omitempty"`
	ProfileDirectory   string                   `json:"profile_directory,omitempty"`
	DatabasePath       string                   `json:"database_path"`
	MemoryMarkdownPath string                   `json:"memory_markdown_path,omitempty"`
	ToolNames          []string                 `json:"tool_names"`
	ToolSpecs          []toolmeta.OperationSpec `json:"tool_specs"`
}

func Open(ctx context.Context, cfg Config) (*Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg = cfg.withDefaults()
	if err := validateProfileDirectoryConfig(cfg); err != nil {
		return nil, err
	}
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
		RecallTool:   goncho.NewGonchoRecallTool(svc),
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
		Ready:              r.Service != nil && r.DB != nil,
		WorkspaceID:        r.config.WorkspaceID,
		ObserverID:         r.config.ObserverID,
		ProfileID:          r.config.ProfileID,
		ProfilesDirectory:  r.config.ProfilesDirectory,
		ProfileDirectory:   profileDirectory(r.config.ProfilesDirectory, r.config.ProfileID),
		DatabasePath:       r.config.DatabasePath,
		MemoryMarkdownPath: r.config.MemoryMarkdownPath,
		ToolNames:          []string{r.ContextTool.Name(), r.SearchTool.Name(), r.RecallTool.Name(), r.RememberTool.Name(), r.ReviewTool.Name(), r.HandoffTool.Name()},
		ToolSpecs:          []toolmeta.OperationSpec{r.ContextTool.Spec(), r.SearchTool.Spec(), r.RecallTool.Spec(), r.RememberTool.Spec(), r.ReviewTool.Spec(), r.HandoffTool.Spec()},
	}
}

func (c Config) withDefaults() Config {
	out := c
	out.ProfileID = strings.TrimSpace(out.ProfileID)
	out.ProfilesDirectory = strings.TrimSpace(out.ProfilesDirectory)
	if strings.TrimSpace(out.WorkspaceID) == "" {
		out.WorkspaceID = DefaultWorkspaceID
	}
	if strings.TrimSpace(out.ObserverID) == "" {
		out.ObserverID = DefaultObserverID
	}
	if out.RecentMessages <= 0 {
		out.RecentMessages = 8
	}
	if strings.TrimSpace(out.DatabasePath) == "" && out.ProfilesDirectory != "" && out.ProfileID != "" {
		out.DatabasePath = filepath.Join(profileDirectory(out.ProfilesDirectory, out.ProfileID), "goncho.db")
	}
	if strings.TrimSpace(out.MemoryMarkdownPath) == "" && strings.TrimSpace(out.DatabasePath) != "" {
		out.MemoryMarkdownPath = filepath.Join(filepath.Dir(out.DatabasePath), "GONCHO_MEMORY.md")
	}
	return out
}

func profileDirectory(profilesDirectory, profileID string) string {
	profilesDirectory = strings.TrimSpace(profilesDirectory)
	profileID = strings.TrimSpace(profileID)
	if profilesDirectory == "" || profileID == "" {
		return ""
	}
	return filepath.Join(profilesDirectory, profileID)
}

func validateProfileDirectoryConfig(cfg Config) error {
	if strings.TrimSpace(cfg.ProfilesDirectory) == "" && strings.TrimSpace(cfg.ProfileID) == "" {
		return nil
	}
	if strings.TrimSpace(cfg.ProfilesDirectory) == "" {
		return fmt.Errorf("gormes goncho: profiles directory is required when profile_id is set")
	}
	if strings.TrimSpace(cfg.ProfileID) == "" {
		return fmt.Errorf("gormes goncho: profile_id is required when profiles directory is set")
	}
	if strings.ContainsAny(cfg.ProfileID, `/\\`) || cfg.ProfileID == "." || cfg.ProfileID == ".." || strings.Contains(cfg.ProfileID, "..") {
		return fmt.Errorf("gormes goncho: unsafe profile_id %q", cfg.ProfileID)
	}
	return nil
}
