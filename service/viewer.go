package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/idutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
)

// ViewerSnapshot is the read-only JSON model for Goncho's local viewer API.
type ViewerSnapshot struct {
	Status                     string                      `json:"status"`
	ReadOnly                   bool                        `json:"read_only"`
	WorkspaceID                string                      `json:"workspace_id"`
	ObserverPeerID             string                      `json:"observer_peer_id"`
	DB                         ViewerDBInfo                `json:"db"`
	Counts                     ViewerCounts                `json:"counts"`
	LatestObservations         []Observation               `json:"latest_observations"`
	LatestConclusions          []ViewerConclusion          `json:"latest_conclusions"`
	ReviewQueue                ViewerReviewQueue           `json:"review_queue"`
	NegativeEvidenceCandidates []NegativeEvidenceCandidate `json:"negative_evidence_candidates,omitempty"`
	GeneratedAt                time.Time                   `json:"generated_at"`
	UnavailableWarnings        []string                    `json:"unavailable_warnings,omitempty"`
}

type ViewerDBInfo struct {
	Path string `json:"path"`
}

type ViewerCounts struct {
	Workspaces     int `json:"workspaces"`
	Profiles       int `json:"profiles"`
	Sessions       int `json:"sessions"`
	Messages       int `json:"messages"`
	Observations   int `json:"observations"`
	Conclusions    int `json:"conclusions"`
	ReviewOpen     int `json:"review_open"`
	ReviewResolved int `json:"review_resolved"`
}

type ViewerConclusion struct {
	ID          int64     `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ProfileID   string    `json:"profile_id,omitempty"`
	PeerID      string    `json:"peer_id"`
	SessionKey  string    `json:"session_key,omitempty"`
	Content     string    `json:"content"`
	Scope       string    `json:"scope"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type ViewerReviewQueue struct {
	Open       int          `json:"open"`
	Resolved   int          `json:"resolved"`
	LatestOpen []ReviewItem `json:"latest_open"`
}

// ViewerRecallTrace is the read-only viewer wrapper around a recall trace.
type ViewerRecallTrace struct {
	Status      string      `json:"status"`
	ReadOnly    bool        `json:"read_only"`
	WorkspaceID string      `json:"workspace_id"`
	Peer        string      `json:"peer"`
	Query       string      `json:"query"`
	Trace       RecallTrace `json:"trace"`
	GeneratedAt time.Time   `json:"generated_at"`
}

type ViewerSessionTimeline struct {
	Status       string                `json:"status"`
	ReadOnly     bool                  `json:"read_only"`
	WorkspaceID  string                `json:"workspace_id"`
	SessionKey   string                `json:"session_key"`
	Messages     []MessageRecord       `json:"messages"`
	Observations []Observation         `json:"observations"`
	Summaries    []SessionSummary      `json:"summaries"`
	Events       []ViewerTimelineEvent `json:"events"`
	GeneratedAt  time.Time             `json:"generated_at"`
}

type ViewerTimelineEvent struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	AtUnix    int64  `json:"at_unix"`
	Sequence  int    `json:"sequence,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Role      string `json:"role,omitempty"`
	PeerID    string `json:"peer_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Source    string `json:"source"`
	Truncated bool   `json:"truncated,omitempty"`
}

// ViewerSnapshot returns a local-first, read-only overview for viewer clients.
func (s *Service) ViewerSnapshot(ctx context.Context) (ViewerSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return ViewerSnapshot{}, err
	}
	if s == nil || s.db == nil {
		return ViewerSnapshot{}, fmt.Errorf("goncho: nil service")
	}
	workspaceID := s.workspaceID
	snapshot := ViewerSnapshot{
		Status:         "ok",
		ReadOnly:       true,
		WorkspaceID:    workspaceID,
		ObserverPeerID: s.observer,
		DB:             ViewerDBInfo{Path: databasePath(ctx, s.db)},
		GeneratedAt:    time.Now().UTC(),
	}
	var warnings []string
	count := func(name, query string, args ...any) int {
		value, err := countScalar(ctx, s.db, query, args...)
		if err != nil {
			warnings = append(warnings, name+": "+err.Error())
			return 0
		}
		return value
	}
	snapshot.Counts = ViewerCounts{
		Workspaces:     count("workspaces", `SELECT COUNT(*) FROM (SELECT ? AS workspace_id UNION SELECT workspace_id FROM goncho_observations UNION SELECT workspace_id FROM goncho_conclusions UNION SELECT workspace_id FROM goncho_review_items)`, workspaceID),
		Profiles:       count("profiles", `SELECT COUNT(DISTINCT profile_id) FROM goncho_peer_cards WHERE workspace_id = ? AND profile_id != ''`, workspaceID),
		Sessions:       count("sessions", viewerTurnsCountSQL(`COUNT(DISTINCT session_id)`), workspaceID, workspaceID),
		Messages:       count("messages", viewerTurnsCountSQL(`COUNT(*)`), workspaceID, workspaceID),
		Observations:   count("observations", `SELECT COUNT(*) FROM goncho_observations WHERE workspace_id = ?`, workspaceID),
		Conclusions:    count("conclusions", `SELECT COUNT(*) FROM goncho_conclusions WHERE workspace_id = ? AND observer_peer_id = ?`, workspaceID, s.observer),
		ReviewOpen:     count("review_open", `SELECT COUNT(*) FROM goncho_review_items WHERE workspace_id = ? AND status = 'open'`, workspaceID),
		ReviewResolved: count("review_resolved", `SELECT COUNT(*) FROM goncho_review_items WHERE workspace_id = ? AND status = 'resolved'`, workspaceID),
	}
	observations, err := s.ListObservations(ctx, ObservationQuery{WorkspaceID: workspaceID, Limit: 10})
	if err != nil {
		warnings = append(warnings, "latest_observations: "+err.Error())
	} else {
		snapshot.LatestObservations = observations.Observations
	}
	conclusions, err := latestViewerConclusions(ctx, s.db, workspaceID, s.observer, 10)
	if err != nil {
		warnings = append(warnings, "latest_conclusions: "+err.Error())
	} else {
		snapshot.LatestConclusions = conclusions
	}
	openReviews, err := s.ListReviewItems(ctx, ReviewQuery{WorkspaceID: workspaceID, Status: ReviewStatusOpen, Limit: 10})
	if err != nil {
		warnings = append(warnings, "review_queue: "+err.Error())
	} else {
		snapshot.ReviewQueue.LatestOpen = openReviews.Items
	}
	candidates, err := s.NegativeEvidenceCandidates(ctx, ObservationQuery{WorkspaceID: workspaceID, Limit: 500})
	if err != nil {
		warnings = append(warnings, "negative_evidence_candidates: "+err.Error())
	} else {
		snapshot.NegativeEvidenceCandidates = candidates
	}
	snapshot.ReviewQueue.Open = snapshot.Counts.ReviewOpen
	snapshot.ReviewQueue.Resolved = snapshot.Counts.ReviewResolved
	warnings = append(warnings, providerWarnings(s.ProviderHealthDiagnostics())...)
	snapshot.UnavailableWarnings = warnings
	return snapshot, nil
}

func (s *Service) ViewerRecallTrace(ctx context.Context, peer, query string, limit int) (ViewerRecallTrace, error) {
	if err := ctx.Err(); err != nil {
		return ViewerRecallTrace{}, err
	}
	if s == nil || s.db == nil {
		return ViewerRecallTrace{}, fmt.Errorf("goncho: nil service")
	}
	if strings.TrimSpace(peer) == "" {
		return ViewerRecallTrace{}, fmt.Errorf("goncho: peer is required")
	}
	if strings.TrimSpace(query) == "" {
		return ViewerRecallTrace{}, fmt.Errorf("goncho: query is required")
	}
	limit = limitutil.DefaultClamped(limit, 5, 50)
	trace, err := s.Recall(ctx, RecallQuery{
		WorkspaceID: s.workspaceID,
		Peer:        strings.TrimSpace(peer),
		Query:       strings.TrimSpace(query),
		Limit:       limit,
	})
	if err != nil {
		return ViewerRecallTrace{}, err
	}
	return ViewerRecallTrace{
		Status:      "ok",
		ReadOnly:    true,
		WorkspaceID: s.workspaceID,
		Peer:        strings.TrimSpace(peer),
		Query:       strings.TrimSpace(query),
		Trace:       trace,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (s *Service) ViewerSessionTimeline(ctx context.Context, sessionKey string) (ViewerSessionTimeline, error) {
	if err := ctx.Err(); err != nil {
		return ViewerSessionTimeline{}, err
	}
	if s == nil || s.db == nil {
		return ViewerSessionTimeline{}, fmt.Errorf("goncho: nil service")
	}
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return ViewerSessionTimeline{}, fmt.Errorf("goncho: session_key is required")
	}
	workspaceID := s.workspaceID
	messages, err := listLifecycleMessages(ctx, s.db, workspaceID, sessionKey)
	if err != nil {
		return ViewerSessionTimeline{}, err
	}
	observations, err := s.ListObservations(ctx, ObservationQuery{WorkspaceID: workspaceID, SessionKey: sessionKey, Limit: 500})
	if err != nil {
		return ViewerSessionTimeline{}, err
	}
	shortSummary, longSummary, err := getSessionSummaries(ctx, s.db, workspaceID, sessionKey)
	if err != nil {
		return ViewerSessionTimeline{}, err
	}
	summaries := []SessionSummary{}
	if shortSummary != nil {
		summaries = append(summaries, *shortSummary)
	}
	if longSummary != nil {
		summaries = append(summaries, *longSummary)
	}
	timeline := ViewerSessionTimeline{
		Status:       "ok",
		ReadOnly:     true,
		WorkspaceID:  workspaceID,
		SessionKey:   sessionKey,
		Messages:     messages,
		Observations: observations.Observations,
		Summaries:    summaries,
		GeneratedAt:  time.Now().UTC(),
	}
	timeline.Events = viewerTimelineEvents(messages, observations.Observations, summaries)
	return timeline, nil
}

func viewerTimelineEvents(messages []MessageRecord, observations []Observation, summaries []SessionSummary) []ViewerTimelineEvent {
	events := make([]ViewerTimelineEvent, 0, len(messages)+len(observations)+len(summaries))
	for _, message := range messages {
		events = append(events, ViewerTimelineEvent{
			Type:     "message",
			ID:       idutil.Prefixed("message:", message.ID),
			AtUnix:   message.CreatedAt,
			Sequence: message.Sequence,
			Role:     message.Role,
			PeerID:   message.Peer,
			Content:  message.Content,
			Source:   "turns",
		})
	}
	for _, observation := range observations {
		content := strings.TrimSpace(observation.Input)
		if content == "" {
			content = strings.TrimSpace(observation.Output)
		}
		events = append(events, ViewerTimelineEvent{
			Type:      "observation",
			ID:        "observation:" + observation.ID,
			AtUnix:    observation.ObservedAt.Unix(),
			Kind:      string(observation.Kind),
			PeerID:    observation.PeerID,
			Content:   content,
			Source:    "goncho_observations",
			Truncated: observation.InputTruncated || observation.OutputTruncated,
		})
	}
	for _, summary := range summaries {
		events = append(events, ViewerTimelineEvent{
			Type:    "summary",
			ID:      "summary:" + summary.SummaryType,
			AtUnix:  summary.CreatedAt,
			Kind:    summary.SummaryType,
			Content: summary.Content,
			Source:  "goncho_session_summaries",
		})
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].AtUnix != events[j].AtUnix {
			return events[i].AtUnix < events[j].AtUnix
		}
		if events[i].Sequence != events[j].Sequence {
			return events[i].Sequence < events[j].Sequence
		}
		return events[i].ID < events[j].ID
	})
	return events
}

func viewerTurnsCountSQL(expr string) string {
	return `SELECT ` + expr + ` FROM turns WHERE session_id != '' AND COALESCE(CASE WHEN json_valid(COALESCE(meta_json, '')) THEN json_extract(meta_json, '$.goncho.workspace_id') END, ?) = ?`
}

func databasePath(ctx context.Context, db *sql.DB) string {
	rows, err := db.QueryContext(ctx, `PRAGMA database_list`)
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var seq int
		var name string
		var file string
		if err := rows.Scan(&seq, &name, &file); err == nil && name == "main" {
			return file
		}
	}
	return ""
}

func countScalar(ctx context.Context, db *sql.DB, query string, args ...any) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func latestViewerConclusions(ctx context.Context, db *sql.DB, workspaceID, observer string, limit int) ([]ViewerConclusion, error) {
	limit = limitutil.Default(limit, 10)
	rows, err := db.QueryContext(ctx, `
		SELECT id, workspace_id, profile_id, peer_id, session_key, content, scope, status, created_at
		FROM goncho_conclusions
		WHERE workspace_id = ? AND observer_peer_id = ?
		ORDER BY updated_at DESC, id DESC
		LIMIT ?
	`, workspaceID, observer, limit)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return []ViewerConclusion{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := []ViewerConclusion{}
	for rows.Next() {
		var item ViewerConclusion
		var sessionKey sql.NullString
		var createdAt int64
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.ProfileID, &item.PeerID, &sessionKey, &item.Content, &item.Scope, &item.Status, &createdAt); err != nil {
			return nil, err
		}
		item.SessionKey = sessionKey.String
		if createdAt > 0 {
			item.CreatedAt = time.Unix(createdAt, 0).UTC()
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
