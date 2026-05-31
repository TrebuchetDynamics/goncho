package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/timeutil"
)

var evalFeedbackDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_eval_candidates (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		benchmark_name TEXT NOT NULL,
		run_id TEXT NOT NULL DEFAULT '',
		question_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		status TEXT NOT NULL,
		query TEXT NOT NULL,
		failure_bucket TEXT NOT NULL DEFAULT '',
		rationale TEXT NOT NULL,
		evidence_ids_json TEXT NOT NULL DEFAULT '[]',
		expected_ids_json TEXT NOT NULL DEFAULT '[]',
		retrieved_ids_json TEXT NOT NULL DEFAULT '[]',
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_eval_candidates_status ON goncho_eval_candidates(workspace_id, benchmark_name, status, created_at DESC)`,
	`CREATE TABLE IF NOT EXISTS goncho_recall_feedback (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		peer_id TEXT NOT NULL DEFAULT '',
		session_key TEXT NOT NULL DEFAULT '',
		trace_id TEXT NOT NULL DEFAULT '',
		query TEXT NOT NULL DEFAULT '',
		label TEXT NOT NULL,
		memory_id TEXT NOT NULL DEFAULT '',
		reason TEXT NOT NULL DEFAULT '',
		submitted_by TEXT NOT NULL DEFAULT '',
		review_item_id TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_recall_feedback_trace ON goncho_recall_feedback(workspace_id, trace_id, created_at DESC)`,
}

type EvalCandidateKind string

const (
	EvalCandidateQueryExpansionHint       EvalCandidateKind = "query_expansion_hint"
	EvalCandidateGraphEdgeCandidate       EvalCandidateKind = "graph_edge_candidate"
	EvalCandidateExtractionGap            EvalCandidateKind = "extraction_gap"
	EvalCandidateStaleContradictoryMemory EvalCandidateKind = "stale_contradictory_memory"
	EvalCandidateScopeBug                 EvalCandidateKind = "scope_bug"
)

type EvalCandidateStatus string

const EvalCandidateOpen EvalCandidateStatus = "open"

type EvalRegistryInput struct {
	WorkspaceID   string        `json:"workspace_id,omitempty"`
	BenchmarkName string        `json:"benchmark_name"`
	RunID         string        `json:"run_id,omitempty"`
	Failures      []EvalFailure `json:"failures"`
}

type EvalFailure struct {
	QuestionID         string   `json:"question_id"`
	Category           string   `json:"category,omitempty"`
	Query              string   `json:"query"`
	ExpectedMemoryIDs  []string `json:"expected_memory_ids,omitempty"`
	RetrievedMemoryIDs []string `json:"retrieved_memory_ids,omitempty"`
	TopHitPreview      string   `json:"top_hit_preview,omitempty"`
	FailureBucket      string   `json:"failure_bucket,omitempty"`
}

type EvalRegistryResult struct {
	WorkspaceID   string                     `json:"workspace_id"`
	BenchmarkName string                     `json:"benchmark_name"`
	RunID         string                     `json:"run_id,omitempty"`
	Candidates    []EvalImprovementCandidate `json:"candidates"`
}

type EvalImprovementCandidate struct {
	ID                 string              `json:"id"`
	WorkspaceID        string              `json:"workspace_id"`
	BenchmarkName      string              `json:"benchmark_name"`
	RunID              string              `json:"run_id,omitempty"`
	QuestionID         string              `json:"question_id"`
	Kind               EvalCandidateKind   `json:"kind"`
	Status             EvalCandidateStatus `json:"status"`
	Query              string              `json:"query"`
	FailureBucket      string              `json:"failure_bucket,omitempty"`
	Rationale          string              `json:"rationale"`
	EvidenceIDs        []string            `json:"evidence_ids"`
	ExpectedMemoryIDs  []string            `json:"expected_memory_ids,omitempty"`
	RetrievedMemoryIDs []string            `json:"retrieved_memory_ids,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
}

type EvalCandidateQuery struct {
	WorkspaceID   string              `json:"workspace_id,omitempty"`
	BenchmarkName string              `json:"benchmark_name,omitempty"`
	Status        EvalCandidateStatus `json:"status,omitempty"`
	Limit         int                 `json:"limit,omitempty"`
}

type EvalCandidateList struct {
	Candidates []EvalImprovementCandidate `json:"candidates"`
}

type RecallFeedbackLabel string

const (
	RecallFeedbackUseful  RecallFeedbackLabel = "useful"
	RecallFeedbackWrong   RecallFeedbackLabel = "wrong"
	RecallFeedbackStale   RecallFeedbackLabel = "stale"
	RecallFeedbackUnsafe  RecallFeedbackLabel = "unsafe"
	RecallFeedbackMissing RecallFeedbackLabel = "missing"
)

type RecallFeedbackStatus string

const RecallFeedbackRecorded RecallFeedbackStatus = "recorded"

type RecallFeedbackParams struct {
	WorkspaceID string              `json:"workspace_id,omitempty"`
	Peer        string              `json:"peer_id,omitempty"`
	SessionKey  string              `json:"session_key,omitempty"`
	TraceID     string              `json:"trace_id,omitempty"`
	Query       string              `json:"query,omitempty"`
	Label       RecallFeedbackLabel `json:"label"`
	MemoryID    string              `json:"memory_id,omitempty"`
	Reason      string              `json:"reason"`
	SubmittedBy string              `json:"submitted_by,omitempty"`
}

type RecallFeedback struct {
	ID           string               `json:"id"`
	WorkspaceID  string               `json:"workspace_id"`
	Peer         string               `json:"peer_id,omitempty"`
	SessionKey   string               `json:"session_key,omitempty"`
	TraceID      string               `json:"trace_id,omitempty"`
	Query        string               `json:"query,omitempty"`
	Label        RecallFeedbackLabel  `json:"label"`
	MemoryID     string               `json:"memory_id,omitempty"`
	Reason       string               `json:"reason"`
	SubmittedBy  string               `json:"submitted_by,omitempty"`
	ReviewItemID string               `json:"review_item_id,omitempty"`
	Status       RecallFeedbackStatus `json:"status"`
	CreatedAt    time.Time            `json:"created_at"`
}

type RecallFeedbackQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	TraceID     string `json:"trace_id,omitempty"`
	Peer        string `json:"peer_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type RecallFeedbackList struct {
	Items []RecallFeedback `json:"items"`
}

type BenchmarkMetricComparison struct {
	Metric    string  `json:"metric"`
	Baseline  float64 `json:"baseline"`
	Current   float64 `json:"current"`
	Tolerance float64 `json:"tolerance"`
}

type BenchmarkTrendInput struct {
	BaselineID  string                      `json:"baseline_id"`
	CandidateID string                      `json:"candidate_id"`
	Metrics     []BenchmarkMetricComparison `json:"metrics"`
}

type BenchmarkTrendReport struct {
	BaselineID  string                 `json:"baseline_id"`
	CandidateID string                 `json:"candidate_id"`
	Status      string                 `json:"status"`
	Gates       []RegressionGateResult `json:"gates"`
}

type RegressionGateInput struct {
	Metric    string  `json:"metric"`
	Baseline  float64 `json:"baseline"`
	Current   float64 `json:"current"`
	Tolerance float64 `json:"tolerance"`
}

type RegressionGateResult struct {
	Metric    string  `json:"metric"`
	Baseline  float64 `json:"baseline"`
	Current   float64 `json:"current"`
	Tolerance float64 `json:"tolerance"`
	Drop      float64 `json:"drop"`
	Pass      bool    `json:"pass"`
	Reason    string  `json:"reason"`
}

func (s *Service) RecordEvalFailures(ctx context.Context, input EvalRegistryInput) (EvalRegistryResult, error) {
	workspaceID := serviceObservationWorkspace(s.workspaceID, input.WorkspaceID)
	benchmark := strings.TrimSpace(input.BenchmarkName)
	if benchmark == "" {
		return EvalRegistryResult{}, fmt.Errorf("goncho: eval registry requires benchmark_name")
	}
	out := EvalRegistryResult{WorkspaceID: workspaceID, BenchmarkName: benchmark, RunID: strings.TrimSpace(input.RunID), Candidates: []EvalImprovementCandidate{}}
	for _, failure := range input.Failures {
		candidate := buildEvalCandidate(workspaceID, benchmark, out.RunID, failure)
		if err := insertEvalCandidate(ctx, s.db, candidate); err != nil {
			return EvalRegistryResult{}, err
		}
		out.Candidates = append(out.Candidates, candidate)
	}
	return out, nil
}

func (s *Service) ListEvalCandidates(ctx context.Context, q EvalCandidateQuery) (EvalCandidateList, error) {
	workspaceID := serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	limit := limitutil.Default(q.Limit, 50)
	query := `SELECT id, workspace_id, benchmark_name, run_id, question_id, kind, status, query, failure_bucket, rationale, evidence_ids_json, expected_ids_json, retrieved_ids_json, created_at FROM goncho_eval_candidates WHERE workspace_id = ?`
	args := []any{workspaceID}
	if strings.TrimSpace(q.BenchmarkName) != "" {
		query += ` AND benchmark_name = ?`
		args = append(args, strings.TrimSpace(q.BenchmarkName))
	}
	if q.Status != "" {
		query += ` AND status = ?`
		args = append(args, string(q.Status))
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return EvalCandidateList{}, err
	}
	defer rows.Close()
	out := EvalCandidateList{Candidates: []EvalImprovementCandidate{}}
	for rows.Next() {
		c, err := scanEvalCandidate(rows)
		if err != nil {
			return EvalCandidateList{}, err
		}
		out.Candidates = append(out.Candidates, c)
	}
	return out, rows.Err()
}

func (s *Service) RecordRecallFeedback(ctx context.Context, params RecallFeedbackParams) (RecallFeedback, error) {
	workspaceID := serviceObservationWorkspace(s.workspaceID, params.WorkspaceID)
	label := RecallFeedbackLabel(strings.TrimSpace(string(params.Label)))
	if !validRecallFeedbackLabel(label) {
		return RecallFeedback{}, fmt.Errorf("goncho: invalid recall feedback label %q", params.Label)
	}
	reason := strings.TrimSpace(params.Reason)
	if reason == "" {
		return RecallFeedback{}, fmt.Errorf("goncho: recall feedback requires reason")
	}
	createdAt := time.Now().UTC()
	feedback := RecallFeedback{WorkspaceID: workspaceID, Peer: strings.TrimSpace(params.Peer), SessionKey: strings.TrimSpace(params.SessionKey), TraceID: strings.TrimSpace(params.TraceID), Query: strings.TrimSpace(params.Query), Label: label, MemoryID: strings.TrimSpace(params.MemoryID), Reason: reason, SubmittedBy: strings.TrimSpace(params.SubmittedBy), Status: RecallFeedbackRecorded, CreatedAt: createdAt}
	feedback.ID = feedbackID(feedback)
	if labelRequiresReview(label) {
		item, err := s.CreateReviewItem(ctx, ReviewItemCreateParams{Kind: feedbackReviewKind(label), WorkspaceID: workspaceID, PeerID: feedback.Peer, SessionKey: feedback.SessionKey, SubjectID: firstNonBlank(feedback.MemoryID, "trace:"+feedback.TraceID), Reason: fmt.Sprintf("recall feedback %s: %s", label, reason), EvidenceIDs: feedbackEvidenceIDs(feedback), CreatedAt: createdAt})
		if err != nil {
			return RecallFeedback{}, err
		}
		feedback.ReviewItemID = item.ID
	}
	if err := insertRecallFeedback(ctx, s.db, feedback); err != nil {
		return RecallFeedback{}, err
	}
	return feedback, nil
}

func (s *Service) ListRecallFeedback(ctx context.Context, q RecallFeedbackQuery) (RecallFeedbackList, error) {
	workspaceID := serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	limit := limitutil.Default(q.Limit, 50)
	query := `SELECT id, workspace_id, peer_id, session_key, trace_id, query, label, memory_id, reason, submitted_by, review_item_id, created_at FROM goncho_recall_feedback WHERE workspace_id = ?`
	args := []any{workspaceID}
	if strings.TrimSpace(q.TraceID) != "" {
		query += ` AND trace_id = ?`
		args = append(args, strings.TrimSpace(q.TraceID))
	}
	if strings.TrimSpace(q.Peer) != "" {
		query += ` AND peer_id = ?`
		args = append(args, strings.TrimSpace(q.Peer))
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return RecallFeedbackList{}, err
	}
	defer rows.Close()
	out := RecallFeedbackList{Items: []RecallFeedback{}}
	for rows.Next() {
		item, err := scanRecallFeedback(rows)
		if err != nil {
			return RecallFeedbackList{}, err
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

func BuildBenchmarkTrendReport(input BenchmarkTrendInput) BenchmarkTrendReport {
	report := BenchmarkTrendReport{BaselineID: input.BaselineID, CandidateID: input.CandidateID, Status: "ok", Gates: make([]RegressionGateResult, 0, len(input.Metrics))}
	for _, metric := range input.Metrics {
		gate := EvaluateRegressionGate(RegressionGateInput{Metric: metric.Metric, Baseline: metric.Baseline, Current: metric.Current, Tolerance: metric.Tolerance})
		if !gate.Pass {
			report.Status = "regressed"
		}
		report.Gates = append(report.Gates, gate)
	}
	return report
}

func EvaluateRegressionGate(input RegressionGateInput) RegressionGateResult {
	drop := roundRegression(input.Baseline - input.Current)
	pass := drop <= input.Tolerance
	reason := fmt.Sprintf("metric %s drop %.3f within tolerance %.3f", input.Metric, drop, input.Tolerance)
	if !pass {
		reason = fmt.Sprintf("metric %s drop %.3f exceeds tolerance %.3f", input.Metric, drop, input.Tolerance)
	}
	return RegressionGateResult{Metric: input.Metric, Baseline: input.Baseline, Current: input.Current, Tolerance: input.Tolerance, Drop: drop, Pass: pass, Reason: reason}
}

func buildEvalCandidate(workspaceID, benchmark, runID string, failure EvalFailure) EvalImprovementCandidate {
	kind := evalCandidateKindForFailure(failure)
	evidenceID := fmt.Sprintf("eval:%s:%s:%s", benchmark, runID, strings.TrimSpace(failure.QuestionID))
	rationale := fmt.Sprintf("%s miss in %s: expected %s, retrieved %s", strings.TrimSpace(failure.FailureBucket), strings.TrimSpace(failure.Category), strings.Join(failure.ExpectedMemoryIDs, ","), strings.Join(failure.RetrievedMemoryIDs, ","))
	if strings.TrimSpace(failure.TopHitPreview) != "" {
		rationale += "; top_hit=" + strings.TrimSpace(failure.TopHitPreview)
	}
	createdAt := time.Now().UTC()
	candidate := EvalImprovementCandidate{WorkspaceID: workspaceID, BenchmarkName: benchmark, RunID: runID, QuestionID: strings.TrimSpace(failure.QuestionID), Kind: kind, Status: EvalCandidateOpen, Query: strings.TrimSpace(failure.Query), FailureBucket: strings.TrimSpace(failure.FailureBucket), Rationale: rationale, EvidenceIDs: []string{evidenceID}, ExpectedMemoryIDs: cloneStrings(failure.ExpectedMemoryIDs), RetrievedMemoryIDs: cloneStrings(failure.RetrievedMemoryIDs), CreatedAt: createdAt}
	candidate.ID = evalCandidateID(candidate)
	return candidate
}

func evalCandidateKindForFailure(f EvalFailure) EvalCandidateKind {
	bucket := strings.TrimSpace(f.FailureBucket + " " + f.Category)
	switch {
	case textutil.ContainsAnySubstringFold(bucket, []string{"branch", "scope"}):
		return EvalCandidateScopeBug
	case textutil.ContainsAnySubstringFold(bucket, []string{"multi_hop", "multi-hop"}):
		return EvalCandidateGraphEdgeCandidate
	case textutil.ContainsAnySubstringFold(bucket, []string{"stale", "contradict"}):
		return EvalCandidateStaleContradictoryMemory
	case textutil.ContainsAnySubstringFold(bucket, []string{"missing", "extraction"}):
		return EvalCandidateExtractionGap
	default:
		return EvalCandidateQueryExpansionHint
	}
}

func insertEvalCandidate(ctx context.Context, db *sql.DB, c EvalImprovementCandidate) error {
	evidence, _ := json.Marshal(c.EvidenceIDs)
	expected, _ := json.Marshal(c.ExpectedMemoryIDs)
	retrieved, _ := json.Marshal(c.RetrievedMemoryIDs)
	_, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO goncho_eval_candidates(id, workspace_id, benchmark_name, run_id, question_id, kind, status, query, failure_bucket, rationale, evidence_ids_json, expected_ids_json, retrieved_ids_json, created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.ID, c.WorkspaceID, c.BenchmarkName, c.RunID, c.QuestionID, string(c.Kind), string(c.Status), c.Query, c.FailureBucket, c.Rationale, string(evidence), string(expected), string(retrieved), c.CreatedAt.UnixNano())
	return err
}

func scanEvalCandidate(scanner interface{ Scan(...any) error }) (EvalImprovementCandidate, error) {
	var c EvalImprovementCandidate
	var kind, status, evidence, expected, retrieved string
	var created int64
	if err := scanner.Scan(&c.ID, &c.WorkspaceID, &c.BenchmarkName, &c.RunID, &c.QuestionID, &kind, &status, &c.Query, &c.FailureBucket, &c.Rationale, &evidence, &expected, &retrieved, &created); err != nil {
		return c, err
	}
	c.Kind = EvalCandidateKind(kind)
	c.Status = EvalCandidateStatus(status)
	_ = json.Unmarshal([]byte(evidence), &c.EvidenceIDs)
	_ = json.Unmarshal([]byte(expected), &c.ExpectedMemoryIDs)
	_ = json.Unmarshal([]byte(retrieved), &c.RetrievedMemoryIDs)
	c.CreatedAt = timeutil.UnixNanoUTC(created)
	return c, nil
}

func insertRecallFeedback(ctx context.Context, db *sql.DB, f RecallFeedback) error {
	_, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO goncho_recall_feedback(id, workspace_id, peer_id, session_key, trace_id, query, label, memory_id, reason, submitted_by, review_item_id, created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, f.ID, f.WorkspaceID, f.Peer, f.SessionKey, f.TraceID, f.Query, string(f.Label), f.MemoryID, f.Reason, f.SubmittedBy, f.ReviewItemID, f.CreatedAt.UnixNano())
	return err
}

func scanRecallFeedback(scanner interface{ Scan(...any) error }) (RecallFeedback, error) {
	var f RecallFeedback
	var label string
	var created int64
	if err := scanner.Scan(&f.ID, &f.WorkspaceID, &f.Peer, &f.SessionKey, &f.TraceID, &f.Query, &label, &f.MemoryID, &f.Reason, &f.SubmittedBy, &f.ReviewItemID, &created); err != nil {
		return f, err
	}
	f.Label = RecallFeedbackLabel(label)
	f.Status = RecallFeedbackRecorded
	f.CreatedAt = timeutil.UnixNanoUTC(created)
	return f, nil
}

func validRecallFeedbackLabel(label RecallFeedbackLabel) bool {
	switch label {
	case RecallFeedbackUseful, RecallFeedbackWrong, RecallFeedbackStale, RecallFeedbackUnsafe, RecallFeedbackMissing:
		return true
	default:
		return false
	}
}
func labelRequiresReview(label RecallFeedbackLabel) bool {
	return label == RecallFeedbackWrong || label == RecallFeedbackStale || label == RecallFeedbackUnsafe || label == RecallFeedbackMissing
}
func feedbackReviewKind(label RecallFeedbackLabel) ReviewKind {
	if label == RecallFeedbackStale {
		return ReviewKindStale
	}
	return ReviewKindConflict
}
func feedbackEvidenceIDs(f RecallFeedback) []string {
	out := []string{}
	if f.TraceID != "" {
		out = append(out, "trace:"+f.TraceID)
	}
	if f.MemoryID != "" {
		out = append(out, f.MemoryID)
	}
	return out
}
func evalCandidateID(c EvalImprovementCandidate) string {
	return stableEvalID("eval", c.WorkspaceID, c.BenchmarkName, c.RunID, c.QuestionID, string(c.Kind), c.Query)
}
func feedbackID(f RecallFeedback) string {
	return stableEvalID("feedback", f.WorkspaceID, f.TraceID, f.MemoryID, string(f.Label), f.Reason)
}
func stableEvalID(parts ...string) string {
	var seed strings.Builder
	for _, p := range parts {
		seed.WriteString(strings.TrimSpace(p))
		seed.WriteByte(0)
	}
	return hashutil.SHA256HexStringPrefix(seed.String(), 12)
}
func roundRegression(v float64) float64 { return math.Round(v*1000) / 1000 }
