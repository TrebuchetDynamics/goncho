package docs_test

import "testing"

func TestGitHubAndChatConnectorPlansDocumentControls(t *testing.T) {
	mustContainAllFold(t, mustReadGuardFile(t, "../integrations/github.md"), "github plan", []string{"status: plan", "issues", "pull requests", "discussions", "comments", "scoped observations", "rate-limit", "backfill", "preview"})
	mustContainAllFold(t, mustReadGuardFile(t, "../integrations/slack-discord.md"), "slack/discord plan", []string{"status: plan-after-server-acl", "slack", "discord", "team chats", "server-mode acls", "retention", "workspace/profile authorization", "preview"})
}
