package kb

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zalaras/kb/internal/markers"
)

// freshRoot is the fixture after one gen — check must be green on it.
func freshRoot(t *testing.T) string {
	t.Helper()
	root := newKBRoot(t)
	runGen(t, root)
	return root
}

func TestCheck_PassesOnTheFixtureAfterGen(t *testing.T) {
	assert.Empty(t, runCheck(t, freshRoot(t)))
}

func TestCheck_PassesOnAnEmptyStore(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root)
	mustWriteFile(t, root, "go.mod", "module x\n")
	mustWriteFile(t, root, "docs/versions.txt", "1.2.0 2026-08-29 a\n")
	mustWriteFile(t, root, "internal/catalog/catalog.go", "package catalog\n")
	assert.Empty(t, runCheck(t, root), "the unowned-file rule waits for the first spec record")
	ix, _ := loadFixture(t, root)
	outs, findings, err := Outputs(ix)
	require.NoError(t, err)
	assert.Empty(t, findings)
	assert.Empty(t, outs, "nothing is generated until a record exists")
}

func TestCheck_FailsAFeatureTagWithNoSpecRecord(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/lessons/flush-twice.md", "roles: [impl]\n", "roles: [impl]\nfeatures: [sesions]\n")
	got := runCheck(t, root)
	assert.Contains(t, got, `docs/lessons/flush-twice.md: feature "sesions" has no docs/features/sesions/spec.md`)
}

func TestCheck_FailsAFilesGlobThatMatchesNothing(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/adr/pin-order.md", "files: [internal/catalog/**]", "files: [internal/sesion/**]")
	edit(t, root, "docs/features/catalog/spec.md", "go: [internal/catalog/**]", "go: [internal/catalog/**, internal/Sess/**]")
	got := runCheck(t, root)
	assert.Contains(t, got, `docs/adr/pin-order.md: files entry "internal/sesion/**" matches no file`)
	assert.Contains(t, got, `docs/features/catalog/spec.md: go entry "internal/Sess/**" matches no file`)
}

func TestCheck_FailsATestsEntryWhoseGoFuncOrSpecPathIsMissing(t *testing.T) {
	cases := []struct{ entry, want string }{
		{"TestFoo", `docs/adr/pin-order.md: tests entry "TestFoo" — no func TestFoo( in any *_test.go`},
		{"web/e2e/foo.spec.ts", `docs/adr/pin-order.md: tests entry "web/e2e/foo.spec.ts" does not exist`},
		{"foo", `docs/adr/pin-order.md: tests entry "foo" is neither a Go test name nor a *.spec.ts path`},
	}
	for _, tc := range cases {
		t.Run(tc.entry, func(t *testing.T) {
			root := freshRoot(t)
			edit(t, root, "docs/adr/pin-order.md", "tests: [TestCatalog_PinKeepsOrder, web/e2e/catalog.spec.ts]", "tests: ["+tc.entry+"]")
			assert.Contains(t, runCheck(t, root), tc.want)
		})
	}
}

func TestCheck_FailsARefsEntryThatIsNeitherURLPlanIssueTokenNorPath(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "plans/real/plan.md", "# plan\n")
	edit(t, root, "docs/adr/pin-order.md", `refs: ["#12", https://example.invalid/pin]`,
		`refs: ["#12", https://example.invalid/pin, plan:real, plan:nope, "issue 4", kb:fact/refresh-cadence, kb:fact/pin-order, kb:adr/nope, docs/protocol.md, docs/missing.md]`)
	got := runCheck(t, root)
	assert.Contains(t, got, `docs/adr/pin-order.md: refs entry "plan:nope" — plans/nope/plan.md does not exist`)
	assert.Contains(t, got, `docs/adr/pin-order.md: refs entry "issue 4" is not a URL, plan:<name>, #N, kb:<type>/<id> or an existing repo path`)
	assert.Contains(t, got, `docs/adr/pin-order.md: refs entry "kb:fact/pin-order" — names a decision (want kb:adr/pin-order)`)
	assert.Contains(t, got, `docs/adr/pin-order.md: refs entry "kb:adr/nope" — resolves to no record`)
	assert.Contains(t, got, `docs/adr/pin-order.md: refs entry "docs/missing.md" is not a URL, plan:<name>, #N, kb:<type>/<id> or an existing repo path`)
	for _, ok := range []string{`"plan:real"`, `"#12"`, `"https://`, `"kb:fact/refresh-cadence"`, `"docs/protocol.md"`} {
		assert.NotContains(t, strings.Join(got, "\n"), "refs entry "+ok, "a valid ref must not be reported")
	}
}

func TestCheck_FailsAFactVerifiedAboveTheCeilingOrBelowTheFloorButAllowsALowerBoundBelowFloor(t *testing.T) {
	cases := []struct {
		verified string
		want     []string
	}{
		{"1.2.0..1.4.0", nil},
		{"1.0.0..1.3.0", nil},
		{"1.2.0..ceiling", nil},
		{"1.2.0..1.5.0", []string{"docs/facts/refresh-cadence.md: verified upper bound 1.5.0 exceeds the version record ceiling 1.4.0 (docs/versions.txt)"}},
		{"1.0.0..1.1.0", []string{"docs/facts/refresh-cadence.md: verified upper bound 1.1.0 is below the version record floor 1.2.0 — re-verify and raise it, or set status: retired"}},
		{"1.3.0..1.1.5", []string{`docs/facts/refresh-cadence.md:9: verified "1.3.0..1.1.5" — lower bound exceeds upper`}},
	}
	for _, tc := range cases {
		t.Run(tc.verified, func(t *testing.T) {
			root := freshRoot(t)
			edit(t, root, "docs/facts/refresh-cadence.md", "verified: 1.2.0..1.4.0", "verified: "+tc.verified)
			runGenIfClean(t, root)
			var got []string
			for _, f := range runCheck(t, root) {
				if strings.Contains(f, "verified") {
					got = append(got, f)
				}
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

// runGenIfClean regenerates when the sources load clean, so a test that only changed a
// source field is not also reported as stale.
func runGenIfClean(t *testing.T, root string) {
	t.Helper()
	ix, findings, err := Load(root)
	require.NoError(t, err)
	if len(findings) > 0 {
		return
	}
	outs, _, err := Outputs(ix)
	require.NoError(t, err)
	_, err = Apply(root, outs)
	require.NoError(t, err)
}

func TestCheck_FailsASupersededDecisionWithNoAcceptedSuccessor(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/adr/pin-order.md", "supersedes: [old-pin]\n", "")
	assert.Contains(t, runCheck(t, root), "docs/adr/old-pin.md: status superseded but no accepted or proposed decision lists it in supersedes")
}

// A supersede is a pair — the successor goes accepted as the target goes superseded — and
// both halves land in one commit. A plan branch holds its decisions as `proposed` until
// Completion flips them, so the two pre-flip shapes below must both check clean, or a
// branch cannot represent a supersede at all.
func TestCheck_AllowsAProposedSuccessorBeforeTheCompletionFlip(t *testing.T) {
	root := freshRoot(t)
	// Pre-flip: successor still proposed, target still accepted.
	edit(t, root, "docs/adr/pin-order.md", "status: accepted", "status: proposed")
	edit(t, root, "docs/adr/old-pin.md", "status: superseded", "status: accepted")
	assert.NotContains(t, runCheck(t, root), "supersedes")

	// Mid-flip the other way: target already superseded, successor not yet accepted.
	root2 := freshRoot(t)
	edit(t, root2, "docs/adr/pin-order.md", "status: accepted", "status: proposed")
	assert.NotContains(t, runCheck(t, root2), "supersedes")
}

// The strict half survives: once the successor is accepted, its target must be superseded,
// which is what keeps a half-applied flip off main.
func TestCheck_StillFailsAnAcceptedSuccessorWhoseTargetIsNotSuperseded(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/adr/old-pin.md", "status: superseded", "status: accepted")
	assert.Contains(t, runCheck(t, root), `docs/adr/pin-order.md: supersedes "old-pin" but docs/adr/old-pin.md has status accepted (want superseded)`)
}

func TestCheck_FailsSupersedesPointingAtANonSupersededOrMissingDecision(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/adr/old-pin.md", "status: superseded", "status: accepted")
	edit(t, root, "docs/adr/pin-order.md", "supersedes: [old-pin]", "supersedes: [old-pin, nope, refresh-cadence]")
	got := runCheck(t, root)
	assert.Contains(t, got, `docs/adr/pin-order.md: supersedes "old-pin" but docs/adr/old-pin.md has status accepted (want superseded)`)
	assert.Contains(t, got, `docs/adr/pin-order.md: supersedes "nope" — no such decision`)
	assert.Contains(t, got, `docs/adr/pin-order.md: supersedes "refresh-cadence" — no such decision`)
}

func TestCheck_FailsAnUnresolvedCitationInAGoCommentButIgnoresOneInCode(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "internal/catalog/extra.go",
		"package catalog\n\n// See kb:adr/hook-lifetim for why.\nvar s = \"kb:adr/also-missing\"\n")
	got := runCheck(t, root)
	assert.Contains(t, got, "internal/catalog/extra.go:3: citation kb:adr/hook-lifetim resolves to no record")
	assert.NotContains(t, strings.Join(got, "\n"), "also-missing")
}

func TestCheck_FailsACitationWhosePrefixNamesTheWrongTypeOrAnUnknownAnchor(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "web/src/pin.ts", "// kb:fact/pin-order and kb:anchor/catalog.pinn and kb:anchor/catalog.pin\n")
	edit(t, root, "docs/features/catalog/spec.md", "e2e: [web/e2e/catalog.spec.ts]", "e2e: [web/e2e/catalog.spec.ts]\nweb: [web/src/**]")
	got := runCheck(t, root)
	assert.Contains(t, got, "web/src/pin.ts:1: citation kb:fact/pin-order names a decision (want kb:adr/pin-order)")
	assert.Contains(t, got, "web/src/pin.ts:1: citation kb:anchor/catalog.pinn names no kb:anchor in docs/protocol.md")
	assert.NotContains(t, strings.Join(got, "\n"), "kb:anchor/catalog.pin ")
}

func TestCheck_IgnoresCitationsInPlansHistoryResearchE2ESpecsAndInsideFences(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "plans/x/plan.md", "kb:adr/nope-plan\n")
	mustWriteFile(t, root, "docs/history/h.md", "kb:adr/nope-history\n")
	mustWriteFile(t, root, "docs/research/r.md", "kb:adr/nope-research\n")
	mustWriteFile(t, root, "web/e2e/catalog.spec.ts", "// kb:adr/nope-spec\n")
	mustWriteFile(t, root, "docs/design/d.md", "prose\n\n```\nkb:adr/nope-fenced\n```\n")
	got := strings.Join(runCheck(t, root), "\n")
	assert.NotContains(t, got, "nope-")
}

func TestCheck_FailsAProtocolEntryNamingAnUnknownAnchor(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/features/catalog/spec.md", "protocol: [catalog.pin, catalog.order]", "protocol: [catalog.pinn, catalog.order]")
	assert.Contains(t, runCheck(t, root), `docs/features/catalog/spec.md: protocol entry "catalog.pinn" — no kb:anchor with that id in docs/protocol.md`)
}

func TestCheck_FailsBodyAndCLAUDEmdBudgets(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/adr/pin-order.md", "Pinning must never reorder the list.\n", strings.Repeat("word ", 301)+"\n")
	edit(t, root, "docs/runbooks/../features/catalog/spec.md", "The catalog feature keeps the list ordered.\n", strings.Repeat("word ", 801)+"\n")
	mustWriteFile(t, root, "docs/runbooks/long.md", "---\nid: long\ntype: runbook\nstatus: active\ndate: 2026-08-30\nsummary: s\n---\n"+strings.Repeat("w ", 601))
	edit(t, root, "CLAUDE.md", "More rules.\n", strings.Repeat("line\n", 150))
	frag := mustReadFile(t, root, "internal/catalog/CLAUDE.md")
	mustWriteFile(t, root, "internal/catalog/CLAUDE.md", strings.Repeat("prose ", 401)+"\n"+frag)
	got := runCheck(t, root)
	assert.Contains(t, got, "docs/adr/pin-order.md: body is 301 words (budget 300 for a decision)")
	assert.Contains(t, got, "docs/features/catalog/spec.md: body is 803 words (budget 800 for a spec)")
	assert.Contains(t, got, "docs/runbooks/long.md: body is 601 words (budget 600 for a runbook)")
	assert.Contains(t, got, "CLAUDE.md: 161 lines (budget 150)")
	assert.Contains(t, got, "internal/catalog/CLAUDE.md: 405 words outside kb fragments (budget 400)")

	root = freshRoot(t)
	frag = mustReadFile(t, root, "internal/catalog/CLAUDE.md")
	mustWriteFile(t, root, "internal/catalog/CLAUDE.md", strings.Repeat("prose ", 390)+"\n"+frag)
	assert.NotContains(t, strings.Join(runCheck(t, root), "\n"), "outside kb fragments", "fragment rows are not charged to the budget")
}

func TestCheck_ReportsStaleAndHandEditedGeneratedFilesWithDifferentMessages(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/adr/pin-order.md", "summary: Pinned catalog keep their relative order.", "summary: Pinned catalog keep order.")
	edit(t, root, "docs/features/catalog/contract.md", "Pin body line.", "Pin body line, edited by hand.")
	edit(t, root, "internal/catalog/CLAUDE.md", "records name files in this directory", "records were hand-edited here")
	got := runCheck(t, root)
	assert.Contains(t, got, "docs/INDEX.md: stale — regenerate with kb gen")
	assert.Contains(t, got, "docs/features/catalog/INDEX.md: stale — regenerate with kb gen")
	assert.Contains(t, got, ".claude/rules/catalog.md: stale — regenerate with kb gen")
	assert.Contains(t, got, "docs/features/catalog/contract.md: hand-edited — its body no longer matches its kb:generated hash; revert and change the sources instead")
	assert.Contains(t, got, "internal/catalog/CLAUDE.md: hand-edited — its body no longer matches its kb:generated hash; revert and change the sources instead")
	assert.NotContains(t, strings.Join(got, "\n"), "CLAUDE.md: stale", "the root fragment does not depend on the edited summary")

	runGen(t, root)
	assert.Empty(t, runCheck(t, root), "gen overwrites the hand edit and check is green again")
}

func TestCheck_ReportsAMissingGeneratedFileAndAnOrphanRulesFile(t *testing.T) {
	root := freshRoot(t)
	mustRemove(t, root, "docs/features/catalog/INDEX.md")
	mustWriteFile(t, root, ".claude/rules/oldfeature.md", markers.WrapGenerated("kb gen", "# old\n"))
	mustWriteFile(t, root, ".claude/rules/hand-written.md", "---\npaths:\n  - \"web/**\"\n---\nHand-written rule, not kb's business.\n")
	got := runCheck(t, root)
	assert.Contains(t, got, "docs/features/catalog/INDEX.md: missing — regenerate with kb gen")
	assert.Contains(t, got, ".claude/rules/oldfeature.md: generated file has no feature — delete it")
	assert.NotContains(t, strings.Join(got, "\n"), "hand-written.md")
}

func TestCheck_FailsAnUnrecognisedKBComment(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/protocol.md", "## 1. Conventions\n", "## 1. Conventions\n\n<!-- kb:anchors -->\n")
	mustWriteFile(t, root, "docs/design/d.md", "<!-- kb:anchor elsewhere -->\n## Heading\n\n```\n<!-- kb:fenced-is-fine -->\n```\n")
	got := runCheck(t, root)
	assert.Contains(t, got, `docs/protocol.md:5: unrecognised kb comment "<!-- kb:anchors -->" (want kb:anchor ID, kb:generated HASH, kb:NAME or /kb:NAME)`)
	assert.Contains(t, got, "docs/design/d.md:1: kb:anchor belongs in docs/protocol.md only")
	assert.NotContains(t, strings.Join(got, "\n"), "fenced-is-fine")
}

func TestCheck_ReportsFragmentErrorsInHandWrittenFiles(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "web/CLAUDE.md", "# web\n\n<!-- kb:index -->\n<!-- /kb:index -->\n")
	mustWriteFile(t, root, "cmd/CLAUDE.md", "# cmd\n\n<!-- kb:trailer -->\nnever closed\n")
	got := runCheck(t, root)
	assert.Contains(t, got, `web/CLAUDE.md: unknown fragment name "index" at offset 7 (want features or trailer)`)
	assert.Contains(t, got, `cmd/CLAUDE.md: fragment "trailer" at offset 7 has no matching closing marker`)
}

func TestCheck_ReportsADuplicateIDOnTheLaterPathOnly(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "docs/lessons/pin-order.md", "---\nid: pin-order\ntype: lesson\nstatus: active\ndate: 2026-08-30\nsummary: s\nroles: [review]\n---\n")
	got := runCheck(t, root)
	assert.Contains(t, got, `docs/lessons/pin-order.md: duplicate id "pin-order" (also docs/adr/pin-order.md)`)
	for _, f := range got {
		assert.False(t, strings.HasPrefix(f, "docs/adr/pin-order.md: duplicate"), f)
	}
}

func TestCheck_FailsCodeFilesOwnedByNoFeatureOnlyOnceAFeatureExists(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "internal/other/other.go", "package other\n")
	mustWriteFile(t, root, "web/src/main.ts", "export {}\n")
	mustWriteFile(t, root, "cmd/svc/main.go", "package main\n")
	mustWriteFile(t, root, "internal/other/README.md", "# not code\n")
	got := runCheck(t, root)
	assert.Contains(t, got, "internal/other/other.go: owned by no feature (add it to a docs/features/<name>/spec.md glob)")
	assert.Contains(t, got, "web/src/main.ts: owned by no feature (add it to a docs/features/<name>/spec.md glob)")
	joined := strings.Join(got, "\n")
	assert.NotContains(t, joined, "cmd/svc")
	assert.NotContains(t, joined, "README.md")
	assert.NotContains(t, joined, "internal/catalog/catalog.go")
}

// A feature with more live records than its rules file holds is bounded by truncation,
// never reported as a source defect: failing it would push authors to file records under
// the wrong feature to dodge the budget (Track 3 batch 5 did exactly that once).
func TestCheck_PassesWhenARulesFileIsTruncatedToBudget(t *testing.T) {
	root := newKBRoot(t)
	for i := range 60 {
		id := "r" + strings.Repeat("x", i%3) + string(rune('a'+i/3))
		mustWriteFile(t, root, "docs/rules/"+id+".md", "---\nid: "+id+"\ntype: rule\nstatus: active\ndate: 2026-08-30\nsummary: s\nfeatures: [catalog]\n---\n")
	}
	runGen(t, root)
	got := runCheck(t, root)
	assert.Empty(t, got)
	rules := mustReadFile(t, root, ".claude/rules/catalog.md")
	assert.LessOrEqual(t, strings.Count(rules, "\n"), defaultBudgets.RuleFileLines)
	assert.Contains(t, rules, "more: see `docs/features/catalog/INDEX.md`")
}

func TestCheck_SortsFindingsByPathThenLine(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "internal/catalog/z.go", "package catalog\n// kb:adr/z-two\n// kb:adr/z-one\n")
	mustWriteFile(t, root, "internal/catalog/a.go", "package catalog\n// kb:adr/a-one\n")
	edit(t, root, "docs/adr/pin-order.md", "files: [internal/catalog/**]", "files: [internal/nope/**]")
	got := runCheck(t, root)
	require.GreaterOrEqual(t, len(got), 4)
	assert.True(t, sort.StringsAreSorted(got), "%v", got)
	idxA := indexOf(got, "internal/catalog/a.go:2:")
	idxZ2 := indexOf(got, "internal/catalog/z.go:2:")
	idxZ3 := indexOf(got, "internal/catalog/z.go:3:")
	assert.True(t, idxA < idxZ2 && idxZ2 < idxZ3, "%v", got)
}

func indexOf(list []string, prefix string) int {
	for i, s := range list {
		if strings.HasPrefix(s, prefix) {
			return i
		}
	}
	return -1
}

func TestCheck_HoldsADiagramToOneFenceWhoseKeywordMatchesItsKind(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/diagrams/catalog-state.md", "stateDiagram-v2\n", "sequenceDiagram\n")
	assert.Contains(t, runCheck(t, root), `docs/diagrams/catalog-state.md:13: kind "state" wants a stateDiagram-v2 fence, found "sequenceDiagram"`)

	root = freshRoot(t)
	edit(t, root, "docs/diagrams/catalog-state.md", "Every item moves through these states.\n", "Prose only.\n")
	edit(t, root, "docs/diagrams/catalog-state.md", "```mermaid\nstateDiagram-v2\n  [*] --> starting\n  starting --> running\n  running --> ended\n```\n", "")
	assert.Contains(t, runCheck(t, root), "docs/diagrams/catalog-state.md: a diagram record holds exactly one mermaid fence (found 0)")

	root = freshRoot(t)
	edit(t, root, "docs/diagrams/catalog-state.md", "Every item moves through these states.\n", "Two.\n\n```mermaid\nstateDiagram-v2\n  x --> y\n```\n")
	assert.Contains(t, runCheck(t, root), "docs/diagrams/catalog-state.md: a diagram record holds exactly one mermaid fence (found 2)")
}

func TestCheck_ValidatesMermaidFencesInEveryRecordAgainstTheClosedKeywordList(t *testing.T) {
	root := freshRoot(t)
	edit(t, root, "docs/features/catalog/spec.md", "The catalog feature keeps the list ordered.\n",
		"The catalog feature keeps the list ordered.\n\n```mermaid\nerDiagram\n  SESSION ||--o{ EVENT : logs\n```\n")
	assert.Empty(t, runCheck(t, root), "an inline diagram of an allowed kind passes")

	edit(t, root, "docs/features/catalog/spec.md", "erDiagram\n", "pie title Catalog\n")
	got := runCheck(t, root)
	assert.Contains(t, got, `docs/features/catalog/spec.md:16: mermaid fence opens with "pie" (want one of: C4Component, C4Container, C4Context, classDiagram, erDiagram, flowchart, sequenceDiagram, stateDiagram-v2)`)
	assert.NotContains(t, strings.Join(got, "\n"), "exactly one mermaid fence", "the one-fence rule is for diagram records only")
}

func TestCheck_ResolvesDiagramCitationsAndNamesTheDiagramPrefixOnAMismatch(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, "internal/catalog/state.go", "package catalog\n\n// See kb:diagram/catalog-state and kb:adr/catalog-state and kb:diagram/nope.\n")
	got := runCheck(t, root)
	assert.Contains(t, got, "internal/catalog/state.go:3: citation kb:adr/catalog-state names a diagram (want kb:diagram/catalog-state)")
	assert.Contains(t, got, "internal/catalog/state.go:3: citation kb:diagram/nope resolves to no record")
	assert.NotContains(t, strings.Join(got, "\n"), "kb:diagram/catalog-state resolves")
}

func TestCheck_OwnershipRootsComeFromTheConfig(t *testing.T) {
	root := freshRoot(t)
	mustWriteFile(t, root, ConfigPath, strings.Replace(fixtureConfig, "ownership: [internal, web/src, web/e2e]\n", "ownership: [cmd]\n", 1))
	mustWriteFile(t, root, "internal/other/other.go", "package other\n")
	mustWriteFile(t, root, "cmd/svc/main.go", "package main\n")
	got := runCheck(t, root)
	assert.Contains(t, got, "cmd/svc/main.go: owned by no feature (add it to a docs/features/<name>/spec.md glob)")
	assert.NotContains(t, strings.Join(got, "\n"), "internal/other/other.go")
}
