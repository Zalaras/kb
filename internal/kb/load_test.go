package kb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_BuildsFeaturesFromSpecRecordsAndIndexesRecordsByFeature(t *testing.T) {
	root := newKBRoot(t)
	ix, findings := loadFixture(t, root)
	require.Empty(t, findings)

	require.Len(t, ix.Features, 1)
	f := ix.Features[0]
	assert.Equal(t, "catalog", f.Name)
	assert.Equal(t, "Catalog list, pinning and ordering.", f.Summary)
	assert.Equal(t, []string{"internal/catalog/**"}, f.Go)
	assert.Equal(t, []string{"web/e2e/catalog.spec.ts"}, f.E2E)
	assert.Equal(t, []string{"catalog.pin", "catalog.order"}, f.Protocol)
	assert.Same(t, f.Spec, ix.ByID["catalog"])

	var ids []string
	for _, r := range ix.ByFeature["catalog"] {
		ids = append(ids, r.ID)
	}
	assert.ElementsMatch(t, []string{"catalog", "pin-order", "old-pin", "refresh-cadence", "catalog-state"}, ids)
	assert.Len(t, ix.Records, 7)
	assert.Equal(t, "docs/adr/old-pin.md", ix.Records[0].Path, "records are sorted by path")
	assert.Equal(t, []string{"catalog", "catalog.pin", "catalog.order"}, ix.AnchorOrder)
	assert.True(t, ix.InTree("internal/catalog/catalog.go"))
}

func TestLoad_ReportsEveryBrokenRecordInsteadOfStoppingAtTheFirst(t *testing.T) {
	root := newKBRoot(t)
	mustWriteFile(t, root, "docs/adr/broken-one.md", "no frontmatter at all\n")
	mustWriteFile(t, root, "docs/rules/broken-two.md", "---\nid: broken-two\ntype: rule\nstatus: bogus\ndate: 2026-08-30\nsummary: s\n---\n")
	mustWriteFile(t, root, "docs/runbooks/broken-three.md", "---\nid: broken-three\n")
	ix, findings := loadFixture(t, root)
	var msgs []string
	for _, f := range findings {
		msgs = append(msgs, f.String())
	}
	assert.ElementsMatch(t, []string{
		"docs/adr/broken-one.md:1: no frontmatter: file does not start with ---",
		`docs/rules/broken-two.md:4: status "bogus" is not valid for a rule (want active, draft or retired)`,
		"docs/runbooks/broken-three.md: frontmatter never closed (no second --- line)",
	}, msgs)
	assert.Len(t, ix.Records, 10, "broken records still occupy the index so ids resolve")
	assert.Equal(t, TypeDecision, ix.ByID["broken-one"].Type)
}

func TestLoad_FailsANonRecordMarkdownFileInsideARecordDirectory(t *testing.T) {
	root := newKBRoot(t)
	mustWriteFile(t, root, "docs/features/catalog/notes.md", "# stray\n")
	mustWriteFile(t, root, "docs/lessons/README.md", "# not a record\n")
	_, findings := loadFixture(t, root)
	var msgs []string
	for _, f := range findings {
		msgs = append(msgs, f.String())
	}
	assert.ElementsMatch(t, []string{
		"docs/features/catalog/notes.md: is not a record (a feature directory holds spec.md plus the generated INDEX.md and contract.md)",
		"docs/lessons/README.md:1: no frontmatter: file does not start with ---",
	}, msgs)
}

func TestLoad_IgnoresGeneratedBasenamesUnderFeaturesAndNonRecordDocs(t *testing.T) {
	root := newKBRoot(t)
	runGen(t, root)
	mustWriteFile(t, root, "docs/design/thing.md", "# design, not a record\n")
	mustWriteFile(t, root, "docs/overview.md", "# loose doc\n")
	mustWriteFile(t, root, "docs/adr/notes.txt", "not markdown\n")
	ix, findings := loadFixture(t, root)
	assert.Empty(t, findings)
	assert.Len(t, ix.Records, 7)
}

func TestLoad_ReadsTheVersionRecordFromDisk(t *testing.T) {
	root := newKBRoot(t)
	ix, _ := loadFixture(t, root)
	assert.Equal(t, "1.2.0", ix.Floor)
	assert.Equal(t, "1.4.0", ix.Verified)

	mustWriteFile(t, root, "docs/versions.txt", "1.4.0 2026-09-10 a\n1.1.5 2026-08-01 b\n1.5.0 2026-09-20 c\n")
	ix, _ = loadFixture(t, root)
	assert.Equal(t, "1.1.5", ix.Floor)
	assert.Equal(t, "1.5.0", ix.Verified)

	mustRemove(t, root, "docs/versions.txt")
	_, _, err := Load(root)
	require.Error(t, err, "a missing record file is an I/O error, not a finding")
}

func TestLoad_ReportsAFeatureDirectoryWithoutASpecAndANonSlugName(t *testing.T) {
	root := newKBRoot(t)
	mustWriteFile(t, root, "docs/features/old/INDEX.md", "leftover\n")
	mustWriteFile(t, root, "docs/features/Bad_Name/spec.md", fixtureSpec)
	_, findings := loadFixture(t, root)
	var msgs []string
	for _, f := range findings {
		msgs = append(msgs, f.String())
	}
	assert.Contains(t, msgs, "docs/features/old/: directory has no spec.md")
	assert.Contains(t, msgs, "docs/features/Bad_Name/: feature directory name is not a slug")
}

func TestLoad_FailsWithoutKbYaml(t *testing.T) {
	root := newKBRoot(t)
	mustRemove(t, root, ConfigPath)
	_, _, err := Load(root)
	require.ErrorContains(t, err, "kb.yaml is missing")
}

func TestLoad_SkipsTheVerifiedRangeWhenNoVersionsFileIsSet(t *testing.T) {
	root := newKBRoot(t)
	mustWriteFile(t, root, ConfigPath, strings.Replace(fixtureConfig, "versions:\n  file: docs/versions.txt\n  ceiling_word: ceiling\n", "", 1))
	mustRemove(t, root, "docs/versions.txt")
	edit(t, root, "docs/facts/refresh-cadence.md", "verified: 1.2.0..1.4.0", "verified: 1.2.0..9.9.9")
	runGen(t, root)
	ix, findings := loadFixture(t, root)
	assert.Empty(t, ix.Verified)
	all, err := Check(ix, findings)
	require.NoError(t, err)
	assert.Empty(t, all, "with no version record there is no ceiling to exceed")
}
