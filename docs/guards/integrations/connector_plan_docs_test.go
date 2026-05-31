package integrations_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestGitHubAndChatConnectorPlansDocumentControls(t *testing.T) {
	guardtest.ContainsAllFold(t, guardtest.ReadRepoFile(t, "docs/integrations/planned/github.md"), "github plan", []string{"status: plan", "issues", "pull requests", "discussions", "comments", "scoped observations", "rate-limit", "backfill", "preview"})
	guardtest.ContainsAllFold(t, guardtest.ReadRepoFile(t, "docs/integrations/planned/slack-discord.md"), "slack/discord plan", []string{"status: plan-after-server-acl", "slack", "discord", "team chats", "server-mode acls", "retention", "workspace/profile authorization", "preview"})
}
