package docs_test

import (
	"strings"
	"testing"
)

func TestGitHubAndChatConnectorPlansDocumentControls(t *testing.T) {
	github := strings.ToLower(mustReadGuardFile(t, "../integrations/github.md"))
	for _, want := range []string{"status: plan", "issues", "pull requests", "discussions", "comments", "scoped observations", "rate-limit", "backfill", "preview"} {
		if !strings.Contains(github, want) {
			t.Fatalf("github plan missing %q", want)
		}
	}
	chat := strings.ToLower(mustReadGuardFile(t, "../integrations/slack-discord.md"))
	for _, want := range []string{"status: plan-after-server-acl", "slack", "discord", "team chats", "server-mode acls", "retention", "workspace/profile authorization", "preview"} {
		if !strings.Contains(chat, want) {
			t.Fatalf("slack/discord plan missing %q", want)
		}
	}
}
