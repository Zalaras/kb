package kb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configFrom loads a kb.yaml with the given text from a scratch root.
func configFrom(t *testing.T, text string) (*Config, error) {
	t.Helper()
	root := t.TempDir()
	mustWriteFile(t, root, ConfigPath, text)
	return LoadConfig(root)
}

func TestLoadConfig_FailsWhenTheFileIsAbsent(t *testing.T) {
	_, err := LoadConfig(t.TempDir())
	require.ErrorContains(t, err, "kb.yaml is missing")
}

func TestLoadConfig_RejectsAnUnknownKeyAtAnyDepth(t *testing.T) {
	_, err := configFrom(t, fixtureConfig+"colour: blue\n")
	require.ErrorContains(t, err, "parsing kb.yaml")
	_, err = configFrom(t, fixtureConfigCore+fixtureRefsConfig+"pack:\n  nope: [a]\n")
	require.ErrorContains(t, err, "parsing kb.yaml")
	_, err = configFrom(t, fixtureConfig+"budgets:\n  body_words: many\n")
	require.ErrorContains(t, err, "parsing kb.yaml")
}

func TestLoadConfig_RequiresEveryTypeDirAndNonEmptyTagsAndRoles(t *testing.T) {
	_, err := configFrom(t, strings.Replace(fixtureConfig, "  lesson: docs/lessons\n", "", 1))
	require.EqualError(t, err, `kb.yaml: dirs is missing type "lesson"`)
	_, err = configFrom(t, strings.Replace(fixtureConfig, "  lesson: docs/lessons\n", "  lesson: docs/lessons\n  note: docs/notes\n", 1))
	require.ErrorContains(t, err, `dirs names unknown type "note"`)
	_, err = configFrom(t, strings.Replace(fixtureConfig, "tags: [", "tags: [] #", 1))
	require.EqualError(t, err, "kb.yaml: tags must list at least one tag")
	_, err = configFrom(t, strings.Replace(fixtureConfig, "roles: [", "roles: [] #", 1))
	require.EqualError(t, err, "kb.yaml: roles must list at least one role")
}

func TestLoadConfig_DefaultsAbsentKeysAndKeepsExplicitValues(t *testing.T) {
	cfg, err := configFrom(t, fixtureConfig)
	require.NoError(t, err)
	assert.Equal(t, defaultBudgets, cfg.Budgets)
	assert.Equal(t, "docs/protocol.md", cfg.Paths.Protocol, "an explicit path is kept")
	assert.Equal(t, ".claude/rules", cfg.Paths.RulesDir)
	assert.Equal(t, "plans", cfg.Paths.PlansDir)
	assert.Equal(t, "main", cfg.BaseBranch)
	assert.Equal(t, CommandsConfig{CLI: "kb", Regen: "kb gen"}, cfg.Commands)
	assert.Equal(t, "ceiling", cfg.Versions.CeilingWord)
	assert.Equal(t, []string{"tests", "ui-tests", "review-ui", "docs"}, cfg.Pack.DecisionlessRoles)
	assert.Equal(t, []string{"review", "planner"}, cfg.Pack.ProposedDecisionRoles)
	assert.Equal(t, []string{"internal", "web/src", "web/e2e"}, cfg.Ownership)
	assert.Empty(t, cfg.Refs.Flags.Binary, "the fixture names no binary, so the flag check is off")
	assert.Empty(t, cfg.Agents.Dir, "the fixture names no agents directory, so that check is off")
	assert.Equal(t, "docs/features", cfg.SpecDir())
	assert.Equal(t, "plans/x/plan.md", cfg.PlanPath("x"))

	_, err = configFrom(t, strings.Replace(fixtureConfig, "ownership: [internal, web/src, web/e2e]\n", "ownership: [cmd/]\n", 1)+"budgets:\n  spec_words: 900\npaths_x: 1\n")
	require.Error(t, err, "a misspelt section is unknown, not ignored")
	cfg, err = configFrom(t, strings.Replace(fixtureConfigCore, "ownership: [internal, web/src, web/e2e]\n", "ownership: [cmd/]\n", 1)+fixtureRefsConfig+"budgets:\n  spec_words: 900\npack:\n  decisionless_roles: []\n")
	require.NoError(t, err)
	assert.Equal(t, 900, cfg.Budgets.SpecWords)
	assert.Equal(t, defaultBudgets.BodyWords, cfg.Budgets.BodyWords, "the other budgets keep their defaults")
	assert.Equal(t, []string{"cmd"}, cfg.Ownership, "a trailing slash is trimmed")
	assert.Empty(t, cfg.Pack.DecisionlessRoles, "an explicit empty list stays empty")
	assert.Equal(t, 900, cfg.bodyBudget(TypeSpec))
	assert.Equal(t, defaultBudgets.RunbookWords, cfg.bodyBudget(TypeRunbook))
	assert.Equal(t, defaultBudgets.BodyWords, cfg.bodyBudget(TypeFact))
}

func TestConfig_ExcludedMatchesTheGlobList(t *testing.T) {
	cfg := testConfig(t)
	for rel, want := range map[string]bool{
		"plans/x/plan.md":             true,
		"docs/history/h.md":           true,
		"docs/research/r.md":          true,
		"web/e2e/a.spec.ts":           true,
		"web/e2e/sub/a.spec.ts":       true,
		"web/node_modules/x/y.js":     true,
		"dist/x":                      true,
		"web/e2e/helpers/fixtures.ts": false,
		"docs/adr/x.md":               false,
		"internal/catalog/catalog.go": false,
	} {
		assert.Equal(t, want, cfg.Excluded(rel), rel)
	}
}

// The defaults carry no repository's layout: a kb.yaml with only the required keys turns
// every optional pass off or leaves it unscoped, and opts in key by key.
func TestLoadConfig_GenericDefaultsLeaveEveryOptionalPassOffOrUnscoped(t *testing.T) {
	cfg, err := configFrom(t, "dirs:\n  rule: d/rules\n  decision: d/adr\n  spec: d/features\n  diagram: d/diagrams\n  fact: d/facts\n  lesson: d/lessons\n  runbook: d/runbooks\n  reference: d/references\nroles: [dev]\ntags: [misc]\n")
	require.NoError(t, err)
	assert.Empty(t, cfg.Paths.Protocol, "no anchored document")
	assert.Empty(t, cfg.Paths.Conventions, "no conventions slice")
	assert.Empty(t, cfg.Versions.File, "no verified-range check")
	assert.Empty(t, cfg.Agents.Dir)
	assert.Equal(t, []string{".git", "node_modules", "dist", "bin"}, cfg.Tree.SkipDirs)
	assert.Equal(t, []string{".claude", ".github"}, cfg.Tree.KeepHiddenDirs)
	assert.Empty(t, cfg.Citations.Exclude)
	assert.Empty(t, cfg.Ownership, "no file needs an owning feature")
	assert.Empty(t, cfg.Refs.TopDirs, "no token is a repo path")
	assert.Equal(t, []string{"*.md", "*.go"}, cfg.Refs.Files)
	assert.Empty(t, cfg.Scope.Roots, "kb scope is not configured")
	assert.Equal(t, []string{"**/CLAUDE.md"}, cfg.Scope.Skip)
	for name, list := range map[string][]string{
		"touches_full": cfg.Pack.TouchesFullRoles, "contractless": cfg.Pack.ContractlessRoles, "factless": cfg.Pack.FactlessRoles,
		"decisionless": cfg.Pack.DecisionlessRoles, "system_diagram": cfg.Pack.SystemDiagramRoles, "proposed_decision": cfg.Pack.ProposedDecisionRoles,
	} {
		assert.Empty(t, list, name)
	}
	assert.Empty(t, cfg.Pack.DesignDocs)
	assert.Empty(t, cfg.Pack.ConventionsSections)
	assert.Equal(t, DefaultConfig().Tree, cfg.Tree, "DefaultConfig applies the same defaults")
}
