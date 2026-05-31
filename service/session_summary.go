package goncho

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

var (
	filePathRE = regexp.MustCompile(`(?:[\w./-]+/)?[\w.-]+\.(?:go|py|js|ts|tsx|jsx|rs|rb|sh|md|toml|yaml|yml|json|css|html)`)
	decisionRE = regexp.MustCompile(`(?i)(?:decided|chose|using|switched to|opted for|settled on|prefer)\s+(.+)`)
	skillRE    = regexp.MustCompile(`(?i)(?:skill|plugin|tool)\s+["']?([\w-]+)`)
)

func (s *Service) OnSessionEnd(ctx context.Context, sessionKey string, messages []Message) error {
	go func() {
		summary := extractStructuredSummary(messages)
		data, err := json.Marshal(summary)
		if err != nil {
			s.log.Warn("structured summary marshal failed", "err", err)
			return
		}
		if err := upsertSessionSummary(ctx, s.db, sessionSummaryRow{
			WorkspaceID: s.workspaceID,
			SessionKey:  sessionKey,
			SummaryType: "structured",
			Content:     string(data),
			TokenCount:  approxTokens(string(data)),
		}); err != nil {
			s.log.Warn("structured summary upsert failed", "err", err)
		}
	}()
	return nil
}

func extractStructuredSummary(messages []Message) *StructuredSummary {
	var summary StructuredSummary
	for _, msg := range messages {
		content := msg.Content
		if content == "" {
			continue
		}

		for _, m := range filePathRE.FindAllString(content, -1) {
			summary.FilesModified = append(summary.FilesModified, m)
		}

		for _, m := range decisionRE.FindAllStringSubmatch(content, -1) {
			if len(m) > 1 {
				summary.DecisionsMade = append(summary.DecisionsMade, m[0])
			}
		}

		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasSuffix(line, "?") && len(line) > 10 {
				summary.OpenQuestions = append(summary.OpenQuestions, line)
			}
			if textutil.HasAnyPrefixFold(line, "next:", "todo:", "still need") {
				summary.NextSteps = append(summary.NextSteps, line)
			}
		}

		for _, m := range skillRE.FindAllStringSubmatch(content, -1) {
			if len(m) > 1 {
				summary.SkillOutcomes = append(summary.SkillOutcomes, m[1])
			}
		}
	}

	summary.FilesModified = textutil.NormalizeUnique(summary.FilesModified, nil, false)
	summary.DecisionsMade = textutil.NormalizeUnique(summary.DecisionsMade, strings.TrimSpace, false)
	summary.OpenQuestions = textutil.NormalizeUnique(summary.OpenQuestions, strings.TrimSpace, false)
	summary.SkillOutcomes = textutil.NormalizeUnique(summary.SkillOutcomes, nil, false)
	summary.NextSteps = textutil.NormalizeUnique(summary.NextSteps, strings.TrimSpace, false)
	return &summary
}
