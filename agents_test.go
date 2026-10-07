package kb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// agentsRoot is the fixture with an agents directory: one file per role except the
// session roles, one aliased, and one agent that reads no pack.
func agentsRoot(t *testing.T) string {
	t.Helper()
	root := freshRoot(t)
	mustWriteFile(t, root, ConfigPath, fixtureConfig+"agents:\n  dir: .claude/agents\n  aliases: {review: review-work}\n  session_roles: [orchestrator, planner, retro]\n  agentless: [judge]\n")
	for _, a := range []string{"e2e", "impl", "ui-impl", "tests", "ui-tests", "review-work", "docs", "review-ui", "review-shape", "judge"} {
		mustWriteFile(t, root, ".claude/agents/"+a+".md", "# "+a+"\n")
	}
	return root
}

func TestCheck_TiesTheRoleListToTheAgentsDirectoryWhenConfigured(t *testing.T) {
	root := agentsRoot(t)
	assert.Empty(t, runCheck(t, root), "every role has its agent file and every agent its role")

	mustRemove(t, root, ".claude/agents/ui-tests.md")
	mustWriteFile(t, root, ".claude/agents/stray.md", "# stray\n")
	got := runCheck(t, root)
	assert.Contains(t, got, `kb.yaml: role "ui-tests" packs for no agent file .claude/agents/ui-tests.md (list it under agents.session_roles if that is by design)`)
	assert.Contains(t, got, `.claude/agents/stray.md: agent "stray" has no role in kb.yaml roles, so no lesson can reach it (list it under agents.agentless if it reads no pack)`)
	assert.NotContains(t, strings.Join(got, "\n"), `"review"`, "an aliased role is matched through its alias")
	assert.NotContains(t, strings.Join(got, "\n"), `"judge"`, "an agentless file needs no role")

	mustWriteFile(t, root, ConfigPath, fixtureConfig)
	assert.Empty(t, runCheck(t, root), "with no agents.dir the check is off, whatever the directory holds")
}
