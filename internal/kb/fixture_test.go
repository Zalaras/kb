package kb

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixtureConfigCore is the kb.yaml every fixture starts from: the eight directories, a
// generic role and tag list, a version record, the two documents, and the tree, citation,
// ownership and scope settings the tests exercise. fixturePackConfig and fixtureRefsConfig
// carry the pack and refs sections so a test can replace one without a duplicate key.
const fixtureConfigCore = `dirs:
  rule: docs/rules
  decision: docs/adr
  spec: docs/features
  diagram: docs/diagrams
  fact: docs/facts
  lesson: docs/lessons
  runbook: docs/runbooks
  reference: docs/references
roles: [e2e, impl, ui-impl, tests, ui-tests, review, docs, orchestrator, retro, planner, review-ui, review-shape]
tags: [testing, ux, deps, security, revisit, never]
versions:
  file: docs/versions.txt
  ceiling_word: ceiling
paths:
  protocol: docs/protocol.md
  conventions: docs/conventions.md
tree:
  skip_dirs: [.git, node_modules, web/dist, dist, bin, internal/assets]
  keep_hidden_dirs: [.claude, .githooks, .github]
citations:
  exclude: ["plans/**", "docs/history/**", "docs/research/**", "web/dist/**", "dist/**", "**/node_modules/**", "web/e2e/**/*.spec.ts"]
ownership: [internal, web/src, web/e2e]
scope:
  roots: [cmd, internal, web/src, web/e2e]
  skip: ["**/CLAUDE.md", web/src/protocol.ts]
`

// fixturePackConfig scopes the pack per role: the ui roles read no facts, the test and
// docs roles read no accepted decisions, the two non-correctness reviewers read no
// contract, the planning roles read the system diagrams, and two roles read design docs.
const fixturePackConfig = `pack:
  touches_full_roles: [review]
  contractless_roles: [review-ui, review-shape]
  factless_roles: [ui-impl, ui-tests, review-ui, review-shape, docs]
  decisionless_roles: [tests, ui-tests, review-ui, docs]
  system_diagram_roles: [planner, review, orchestrator, review-shape]
  proposed_decision_roles: [review, planner]
  design_docs:
    ui-impl:
      - path: docs/design/design-system.md
      - path: docs/design/ux-flows.md
    review-ui:
      - path: docs/design/design-system.md
        sections: ["6.", "7."]
      - path: docs/design/ux-flows.md
  conventions_sections:
    impl: [Stack, Go, Composition roots, Design, Knowledge records]
    ui-impl: [Stack, TypeScript, Composition roots, Design, Knowledge records]
    tests: [Stack, Go, Design, Testing, Knowledge records]
    ui-tests: [Design, Testing, Knowledge records]
    e2e: [Testing, Knowledge records]
    review: [Stack, Go, TypeScript, Composition roots, Testing, Knowledge records]
    review-ui: [Stack, TypeScript, Testing, Knowledge records]
    review-shape: [Stack, Go, TypeScript, Composition roots, Design, Knowledge records]
`

// fixtureRefsConfig names the files the refs pass reads and the directories a cited path
// may open with; a test appends flags or a whitelist under it.
const fixtureRefsConfig = `refs:
  files: ["*.md", "*.go", "*.ts", "*.sh", "Makefile", ".githooks/*"]
  top_dirs: [docs, internal, web, cmd, scripts, tools, test, .claude, .githooks, .github]
`

// fixtureConfig is the whole fixture kb.yaml.
const fixtureConfig = fixtureConfigCore + fixturePackConfig + fixtureRefsConfig

// writeConfig writes the fixture kb.yaml into root.
func writeConfig(t *testing.T, root string) {
	t.Helper()
	mustWriteFile(t, root, ConfigPath, fixtureConfig)
}

// testConfig loads the fixture config from a scratch directory, for tests that need a
// Config but no tree.
func testConfig(t *testing.T) *Config {
	t.Helper()
	root := t.TempDir()
	writeConfig(t, root)
	cfg, err := LoadConfig(root)
	require.NoError(t, err)
	return cfg
}

// gitFx runs git in dir; a commit carries a scratch identity per command, never a
// repo-local config (worktrees share .git/config).
func gitFx(t *testing.T, dir string, args ...string) string {
	t.Helper()
	if len(args) > 0 && args[0] == "commit" {
		args = append([]string{"-c", "user.name=kb", "-c", "user.email=kb@example.invalid"}, args...)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return string(out)
}

// newGitRepo turns root into a git checkout on main with two commits (so HEAD~1 resolves),
// the second holding everything written so far.
func newGitRepo(t *testing.T, root string) {
	t.Helper()
	gitFx(t, root, "init", "-q")
	gitFx(t, root, "symbolic-ref", "HEAD", "refs/heads/main")
	gitFx(t, root, "commit", "-q", "--allow-empty", "-m", "initial")
	gitFx(t, root, "add", "-A")
	gitFx(t, root, "commit", "-q", "-m", "fixture")
}

func mustWriteFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
}

func mustReadFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	require.NoError(t, err)
	return string(b)
}

// edit replaces old with new in one fixture file, failing if old is absent.
func edit(t *testing.T, root, rel, oldText, newText string) {
	t.Helper()
	cur := mustReadFile(t, root, rel)
	require.Contains(t, cur, oldText, "edit target must exist in %s", rel)
	mustWriteFile(t, root, rel, strings.Replace(cur, oldText, newText, 1))
}

func mustRemove(t *testing.T, root, rel string) {
	t.Helper()
	require.NoError(t, os.RemoveAll(filepath.Join(root, filepath.FromSlash(rel))))
}

const fixtureProtocol = `# Protocol

## 1. Conventions

Intro text.

<!-- kb:anchor catalog -->
## 3. HTTP endpoints — UI

Overview of the endpoints.

<!-- kb:anchor catalog.pin -->
### 3.10 ` + "`PUT /api/items/{id}/pin`" + `

Pin body line.

` + "```" + `
## not a heading
` + "```" + `

<!-- kb:anchor catalog.order -->
### 3.11 Order

Order body line.

## 4. WebSocket

Socket text.
`

const fixtureSpec = `---
id: catalog
type: spec
status: active
date: 2026-08-30
summary: Catalog list, pinning and ordering.
features: [catalog]
go: [internal/catalog/**]
e2e: [web/e2e/catalog.spec.ts]
protocol: [catalog.pin, catalog.order]
---
# Catalog

The catalog feature keeps the list ordered.
`

const fixturePinOrder = `---
id: pin-order
type: decision
status: accepted
date: 2026-08-30
summary: Pinned catalog keep their relative order.
features: [catalog]
tags: [ux]
files: [internal/catalog/**]
tests: [TestCatalog_PinKeepsOrder, web/e2e/catalog.spec.ts]
refs: ["#12", https://example.invalid/pin]
supersedes: [old-pin]
---
Pinning must never reorder the list.
`

const fixtureOldPin = `---
id: old-pin
type: decision
status: superseded
date: 2026-08-01
summary: Pinned catalog float to the top.
features: [catalog]
---
Superseded by the ordering decision.
`

const fixtureFact = `---
id: refresh-cadence
type: fact
status: active
date: 2026-08-30
summary: The index refreshes about every 300 ms while a scan is running.
features: [catalog]
files: [internal/catalog/catalog.go]
verified: 1.2.0..1.4.0
guard: TestCatalog_PinKeepsOrder
---
Measured on the test bench.
`

const fixtureLesson = `---
id: flush-twice
type: lesson
status: active
date: 2026-08-30
summary: Flush the cache and the index together or the two drift.
roles: [impl]
---
Call both, in that order.
`

const fixtureStateDiagram = `---
id: catalog-state
type: diagram
status: active
date: 2026-09-15
summary: The catalog state machine.
features: [catalog]
files: [internal/catalog/**]
kind: state
---
Every item moves through these states.

` + "```mermaid" + `
stateDiagram-v2
  [*] --> starting
  starting --> running
  running --> ended
` + "```" + `
`

const fixtureContainerDiagram = `---
id: system-container
type: diagram
status: active
date: 2026-09-15
summary: The service, the dashboard and the store.
kind: container
---
The three containers and their wires.

` + "```mermaid" + `
%% comment first
C4Container
  Person(u, "Bob")
  Container(d, "service", "Go")
` + "```" + `
`

// newKBRoot builds the shared fixture: one feature, two decisions, one fact, one
// lesson, two diagrams, three protocol anchors, one code citation and two CLAUDE.md
// fragments.
func newKBRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeConfig(t, root)
	mustWriteFile(t, root, "go.mod", "module example.invalid/fixture\n\ngo 1.27\n")
	mustWriteFile(t, root, "docs/versions.txt",
		"1.2.0 2026-08-29 first\n1.4.0 2026-09-10 second\n")
	mustWriteFile(t, root, "docs/protocol.md", fixtureProtocol)
	mustWriteFile(t, root, "docs/features/catalog/spec.md", fixtureSpec)
	mustWriteFile(t, root, "internal/catalog/catalog.go",
		"package catalog\n\n// Pin keeps the order (kb:adr/pin-order).\nfunc Pin() {}\n")
	mustWriteFile(t, root, "internal/catalog/catalog_test.go",
		"package catalog\n\nimport \"testing\"\n\nfunc TestCatalog_PinKeepsOrder(t *testing.T) {}\n")
	mustWriteFile(t, root, "web/e2e/catalog.spec.ts", "// e2e\n")
	mustWriteFile(t, root, "docs/adr/pin-order.md", fixturePinOrder)
	mustWriteFile(t, root, "docs/adr/old-pin.md", fixtureOldPin)
	mustWriteFile(t, root, "docs/facts/refresh-cadence.md", fixtureFact)
	mustWriteFile(t, root, "docs/lessons/flush-twice.md", fixtureLesson)
	mustWriteFile(t, root, "docs/diagrams/catalog-state.md", fixtureStateDiagram)
	mustWriteFile(t, root, "docs/diagrams/system-container.md", fixtureContainerDiagram)
	mustWriteFile(t, root, "CLAUDE.md",
		"# Fixture\n\nRules.\n\n<!-- kb:features -->\n<!-- /kb:features -->\n\nMore rules.\n")
	mustWriteFile(t, root, "internal/catalog/CLAUDE.md",
		"# catalog\n\nPackage notes.\n\n<!-- kb:trailer -->\n<!-- /kb:trailer -->\n")
	return root
}

// loadFixture loads root and fails on an I/O error.
func loadFixture(t *testing.T, root string) (*Index, []Finding) {
	t.Helper()
	ix, findings, err := Load(root)
	require.NoError(t, err)
	return ix, findings
}

// runGen regenerates every generated file in root and returns the paths written.
func runGen(t *testing.T, root string) []string {
	t.Helper()
	ix, findings := loadFixture(t, root)
	require.Empty(t, findings, "fixture must load clean before gen")
	outs, fragFindings, err := Outputs(ix)
	require.NoError(t, err)
	require.Empty(t, fragFindings)
	written, err := Apply(root, outs)
	require.NoError(t, err)
	return written
}

// runCheck loads and checks root, returning the findings as strings.
func runCheck(t *testing.T, root string) []string {
	t.Helper()
	ix, findings, err := Load(root)
	require.NoError(t, err)
	all, err := Check(ix, findings)
	require.NoError(t, err)
	out := make([]string, 0, len(all))
	for _, f := range all {
		out = append(out, f.String())
	}
	return out
}
