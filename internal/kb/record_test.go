package kb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zalaras/kb/internal/finding"
	"github.com/Zalaras/kb/internal/frontmatter"
)

// parseRecord runs the scanner and the record layer on src as if it sat at relpath.
func parseRecord(t *testing.T, relpath, src string) (*Record, []string) {
	t.Helper()
	fields, body, bodyLine, err := frontmatter.Parse([]byte(src))
	require.NoError(t, err)
	r, findings := RecordFromFields(testConfig(t), relpath, fields, body, bodyLine)
	finding.Sort(findings)
	var msgs []string
	for _, f := range findings {
		msgs = append(msgs, f.String())
	}
	return r, msgs
}

func TestRecordFromFields_EnforcesPerTypeRequiredAndForbiddenFields(t *testing.T) {
	base := "id: %s\ntype: %s\nstatus: %s\ndate: 2026-08-30\nsummary: s\n"
	cases := []struct {
		name  string
		path  string
		front string
		want  []string
	}{
		{"fact needs verified", "docs/facts/x.md", "id: x\ntype: fact\nstatus: active\ndate: 2026-08-30\nsummary: s\n",
			[]string{`docs/facts/x.md: missing required field "verified" (fact records must carry one)`}},
		{"lesson needs roles", "docs/lessons/x.md", "id: x\ntype: lesson\nstatus: active\ndate: 2026-08-30\nsummary: s\n",
			[]string{`docs/lessons/x.md: missing required field "roles" (lesson records must name at least one role)`}},
		{"roles only on lesson", "docs/adr/x.md", "id: x\ntype: decision\nstatus: accepted\ndate: 2026-08-30\nsummary: s\nroles: [review]\n",
			[]string{`docs/adr/x.md:7: field "roles" is only valid on lesson records`}},
		{"verified only on fact", "docs/rules/x.md", "id: x\ntype: rule\nstatus: active\ndate: 2026-08-30\nsummary: s\nverified: 1.2.0..1.4.0\n",
			[]string{`docs/rules/x.md:7: field "verified" is only valid on fact records`}},
		{"supersedes only on decision", "docs/runbooks/x.md", "id: x\ntype: runbook\nstatus: active\ndate: 2026-08-30\nsummary: s\nsupersedes: [y]\n",
			[]string{`docs/runbooks/x.md:7: field "supersedes" is only valid on decision records`}},
		{"go only on spec", "docs/references/x.md", "id: x\ntype: reference\nstatus: active\ndate: 2026-08-30\nsummary: s\ngo: [internal/**]\n",
			[]string{`docs/references/x.md:7: field "go" is only valid on spec records`}},
		{"unknown field", "docs/adr/x.md", "id: x\ntype: decision\nstatus: accepted\ndate: 2026-08-30\nsummary: s\nfoo: bar\n",
			[]string{`docs/adr/x.md:7: unknown field "foo"`}},
		{"missing required", "docs/adr/x.md", "id: x\ntype: decision\n",
			[]string{`docs/adr/x.md: missing required field "date"`, `docs/adr/x.md: missing required field "status"`, `docs/adr/x.md: missing required field "summary"`}},
		{"list where scalar", "docs/adr/x.md", "id: [x]\ntype: decision\nstatus: accepted\ndate: 2026-08-30\nsummary: s\n",
			[]string{`docs/adr/x.md:2: field "id" takes a single value, not a list`}},
		{"scalar where list", "docs/adr/x.md", "id: x\ntype: decision\nstatus: accepted\ndate: 2026-08-30\nsummary: s\ntags: ux\n",
			[]string{`docs/adr/x.md:7: field "tags" takes a list (write tags: [a, b])`}},
		{"unknown tag", "docs/adr/x.md", "id: x\ntype: decision\nstatus: accepted\ndate: 2026-08-30\nsummary: s\ntags: [nope]\n",
			[]string{`docs/adr/x.md:7: unknown tag "nope" (want one of: ` + strings.Join(testConfig(t).Tags, ", ") + ")"}},
		{"unknown role", "docs/lessons/x.md", "id: x\ntype: lesson\nstatus: active\ndate: 2026-08-30\nsummary: s\nroles: [ceo]\n",
			[]string{`docs/lessons/x.md:7: unknown role "ceo" (want one of: ` + strings.Join(testConfig(t).Roles, ", ") + ")"}},
		{"bad guard", "docs/facts/x.md", "id: x\ntype: fact\nstatus: active\ndate: 2026-08-30\nsummary: s\nverified: 1.2.0..ceiling\nguard: notATest\n",
			[]string{`docs/facts/x.md:8: guard "notATest" is not a Go test name (or the word none)`}},
		{"clean rule", "docs/rules/x.md", "id: x\ntype: rule\nstatus: active\ndate: 2026-08-30\nsummary: s\n", nil},
		{"clean fact with the ceiling word and none", "docs/facts/x.md", "id: x\ntype: fact\nstatus: draft\ndate: 2026-08-30\nsummary: s\nverified: 1.2.0..ceiling\nguard: none\n", nil},
	}
	_ = base
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := parseRecord(t, tc.path, "---\n"+tc.front+"---\nbody\n")
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRecordFromFields_RejectsAStatusFromAnotherTypesEnum(t *testing.T) {
	_, got := parseRecord(t, "docs/adr/x.md", "---\nid: x\ntype: decision\nstatus: active\ndate: 2026-08-30\nsummary: s\n---\n")
	assert.Equal(t, []string{`docs/adr/x.md:4: status "active" is not valid for a decision (want accepted, proposed, superseded or rejected)`}, got)

	_, got = parseRecord(t, "docs/facts/x.md", "---\nid: x\ntype: fact\nstatus: accepted\ndate: 2026-08-30\nsummary: s\nverified: 1.2.0..ceiling\n---\n")
	assert.Equal(t, []string{`docs/facts/x.md:4: status "accepted" is not valid for a fact (want active, draft or retired)`}, got)
}

func TestRecordFromFields_ReportsTypeDirIdDateAndSummaryProblemsOnTheirLines(t *testing.T) {
	long := strings.Repeat("s", 190)
	_, got := parseRecord(t, "docs/adr/x.md", "---\nid: y\ntype: fact\nstatus: accepted\ndate: 2026-9-1\nsummary: "+long+"\n---\n")
	assert.Equal(t, []string{
		`docs/adr/x.md:2: id "y" does not match the filename slug "x"`,
		`docs/adr/x.md:3: type "fact" does not belong in docs/adr/ (that directory holds decision records)`,
		`docs/adr/x.md:5: date "2026-9-1" is not YYYY-MM-DD`,
		`docs/adr/x.md:6: summary is 190 chars (budget 160)`,
	}, got)

	_, got = parseRecord(t, "docs/adr/x.md", "---\nid: x\ntype: note\nstatus: accepted\ndate: 2026-08-30\nsummary: s\n---\n")
	assert.Equal(t, []string{`docs/adr/x.md:3: unknown type "note" (want rule, spec, diagram, decision, fact, lesson, runbook or reference)`}, got)
}

func TestRecordFromFields_DerivesIDFromTheFeatureDirectoryForSpecRecords(t *testing.T) {
	r, got := parseRecord(t, "docs/features/catalog/spec.md", "---\nid: catalog\ntype: spec\nstatus: active\ndate: 2026-08-30\nsummary: s\nfeatures: [catalog]\n---\n")
	assert.Empty(t, got)
	assert.Equal(t, TypeSpec, r.Type)
	assert.Equal(t, "catalog", r.ID)

	_, got = parseRecord(t, "docs/features/catalog/spec.md", "---\nid: items\ntype: spec\nstatus: active\ndate: 2026-08-30\nsummary: s\nfeatures: [other]\n---\n")
	assert.Equal(t, []string{
		`docs/features/catalog/spec.md: a spec record's features must be exactly [catalog]`,
		`docs/features/catalog/spec.md:2: id "items" does not match the feature directory "catalog"`,
	}, got)
}

func TestRecordFromFields_CountsBodyWordsIncludingFences(t *testing.T) {
	r, got := parseRecord(t, "docs/rules/x.md", "---\nid: x\ntype: rule\nstatus: active\ndate: 2026-08-30\nsummary: s\n---\none two\n\n```go\nthree four five\n```\n")
	assert.Empty(t, got)
	assert.Equal(t, 7, r.BodyWords, "fence markers and their contents all count")
	assert.Equal(t, 8, r.BodyLine)
}

func TestParseVersionRange_AcceptsOrderedRangesAndRejectsTheRest(t *testing.T) {
	cases := []struct {
		in     string
		want   VersionRange
		errSub string
	}{
		{"1.2.0..1.4.0", VersionRange{Lo: "1.2.0", Hi: "1.4.0"}, ""},
		{"1.2.0..1.2.0", VersionRange{Lo: "1.2.0", Hi: "1.2.0"}, ""},
		{"2.1.9..2.1.10", VersionRange{Lo: "2.1.9", Hi: "2.1.10"}, ""},
		{"1.2.0..ceiling", VersionRange{Lo: "1.2.0", Hi: "ceiling", HiCeiling: true}, ""},
		{"1.3.0..1.1.5", VersionRange{}, "lower bound exceeds upper"},
		{"1.2.0", VersionRange{}, "is not <x.y.z>..<x.y.z> or <x.y.z>..ceiling"},
		{"2.1..1.4.0", VersionRange{}, "is not"},
		{"ceiling..1.4.0", VersionRange{}, "is not"},
		{"1.2.0..latest", VersionRange{}, "is not"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseVersionRange(tc.in, "ceiling")
			if tc.errSub != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSub)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRecordFromFields_DiagramsRequireAKnownKindAndKindIsScopedToThem(t *testing.T) {
	head := "---\nid: x\ntype: diagram\nstatus: active\ndate: 2026-09-15\nsummary: s\n"
	_, got := parseRecord(t, "docs/diagrams/x.md", head+"---\n")
	assert.Equal(t, []string{`docs/diagrams/x.md: missing required field "kind" (diagram records must name one of: component, container, context, domain, er, flow, sequence, state)`}, got)

	_, got = parseRecord(t, "docs/diagrams/x.md", head+"kind: mindmap\n---\n")
	assert.Equal(t, []string{`docs/diagrams/x.md:7: unknown kind "mindmap" (want one of: component, container, context, domain, er, flow, sequence, state)`}, got)

	r, got := parseRecord(t, "docs/diagrams/x.md", head+"kind: er\n---\n")
	assert.Empty(t, got)
	assert.Equal(t, "er", r.Kind)
	assert.Equal(t, "kb:diagram/x", r.Token())

	_, got = parseRecord(t, "docs/facts/f.md", "---\nid: f\ntype: fact\nstatus: active\ndate: 2026-09-15\nsummary: s\nverified: 1.2.0..ceiling\nkind: er\n---\n")
	assert.Equal(t, []string{`docs/facts/f.md:8: field "kind" is only valid on diagram records`}, got)
}

func TestRecordFromFields_DoesNotBudgetMermaidFenceSource(t *testing.T) {
	r, got := parseRecord(t, "docs/features/catalog/spec.md", "---\nid: catalog\ntype: spec\nstatus: active\ndate: 2026-09-15\nsummary: s\nfeatures: [catalog]\n---\none two\n\n```mermaid\nsequenceDiagram\n  A->>B: hello there friend\n```\n")
	assert.Empty(t, got)
	assert.Equal(t, 4, r.BodyWords, "prose and the two fence markers count; the diagram source does not")
}
