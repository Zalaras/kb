package kb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// refsRoot is the kb fixture plus a Makefile, a flag-defining main, a .gitignore, a
// whitelist with one stale entry, and one citing file per comment style, committed on main.
func refsRoot(t *testing.T) string {
	t.Helper()
	root := newKBRoot(t)
	mustWriteFile(t, root, ConfigPath, fixtureConfig+"  flags:\n    binary: svc\n    source: cmd/svc\n  whitelist:\n    docs/wl.md: not written yet\n    internal/catalog/catalog_test.go: a stale entry\n")
	mustWriteFile(t, root, "Makefile", "build: ## Build\n\tgo build\n.PHONY: build\n")
	mustWriteFile(t, root, "cmd/svc/main.go", "package main\n\nimport \"flag\"\n\nfunc main() {\n\tfset := flag.NewFlagSet(\"svc\", 0)\n\tvar addr string\n\tfset.StringVar(&addr, \"addr\", \"\", \"\")\n}\n")
	mustWriteFile(t, root, "cmd/svc/main_test.go", "package main\n\n// fset.StringVar(&x, \"only-in-tests\", \"\", \"\")\n")
	mustWriteFile(t, root, ".gitignore", "web/dist\n")
	mustWriteFile(t, root, "docs/notes.md", strings.Join([]string{
		"`internal/catalog/catalog.go` exists",                  // 1
		"`docs/nope.md` is gone",                                // 2
		"`make build` and `make nope` and `make X=1`",           // 3
		"`svc -addr` then `svc --nope`",                         // 4
		"`internal/catalog.Manager` is a qualified name",        // 5
		"`catalog.go` resolves by basename; `other.go` nowhere", // 6
		"`web/dist` is gitignored",                              // 7
		"`docs/*.md` and `docs/{a,b}.md` carry skip chars",      // 8
		"`internal/catalog/catalog.go:12` and `internal/catalog/catalog.go.` and `./internal/catalog/catalog.go#why`", // 9
		"```",                         // 10
		"`docs/fenced-nope.md`",       // 11
		"```",                         // 12
		"`docs/wl.md` is whitelisted", // 13
		"",
	}, "\n"))
	mustWriteFile(t, root, "scripts/x.sh", "#!/bin/sh\n# see docs/nope2.md\necho docs/nope3.md\n")
	mustWriteFile(t, root, "internal/catalog/doc.go", "package catalog\n\n// See internal/nope/x.go and docs/adr/pin-order.md.\n")
	mustWriteFile(t, root, ".githooks/pre-commit", "#!/bin/sh\n# docs/hook-nope.md\n")
	newGitRepo(t, root)
	return root
}

func TestRefs_AllChecksEveryCitedPathMakeTargetAndFlag(t *testing.T) {
	root := refsRoot(t)
	ix, _ := loadFixture(t, root)
	res, err := Refs(ix, RefsOptions{All: true})
	require.NoError(t, err)
	assert.Equal(t, 16, res.Checked, "every resolvable or reportable token counts once; a whitelisted token never does")
	assert.Equal(t, 6, res.Missing)
	var lines []string
	for _, h := range res.Hits {
		lines = append(lines, h.String())
	}
	assert.ElementsMatch(t, []string{
		"docs/notes.md:2  docs/nope.md  missing (path)",
		"docs/notes.md:3  make nope  missing (make target)",
		"docs/notes.md:4  svc -nope  missing (flag)",
		"docs/notes.md:7  web/dist  ignored (gitignored by design)",
		"scripts/x.sh:2  docs/nope2.md  missing (path)",
		"internal/catalog/doc.go:3  internal/nope/x.go  missing (path)",
		".githooks/pre-commit:2  docs/hook-nope.md  missing (path)",
	}, lines)
	assert.Equal(t, []string{"internal/catalog/catalog_test.go"}, res.StaleWhitelist)
	assert.Equal(t, Finding{Path: "docs/notes.md", Line: 3, Msg: "reference `make nope` missing (make target)"}, res.Hits[0].Finding(), "make and flag hits report as they are met; path hits wait for the git-ignore batch")
}

func TestRefs_ExplicitFilesScanOnlyThoseFiles(t *testing.T) {
	root := refsRoot(t)
	ix, _ := loadFixture(t, root)
	res, err := Refs(ix, RefsOptions{Files: []string{"docs/notes.md", "docs/missing-file.md", "plans/x/plan.md"}})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Files, "a file that does not exist or is out of scope is dropped")
	assert.Equal(t, 3, res.Missing)
}

func TestRefs_ChangedModeFindsACitationOfAMovedPathAnywhereInTheTree(t *testing.T) {
	root := newKBRoot(t)
	mustWriteFile(t, root, "internal/catalog/old.go", "package catalog\n")
	mustWriteFile(t, root, "docs/cite-old.md", "the `catalog/old.go` file, see also internal/catalog/old.go\n")
	newGitRepo(t, root)
	gitFx(t, root, "checkout", "-q", "-b", "plan/x")
	gitFx(t, root, "rm", "-q", "internal/catalog/old.go")
	gitFx(t, root, "commit", "-q", "-m", "drop old")
	ix, _ := loadFixture(t, root)
	res, err := Refs(ix, RefsOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Files, "the branch changed no file still in scope")
	require.Len(t, res.Hits, 1)
	assert.Equal(t, "docs/cite-old.md:1  catalog/old.go  missing (moved: internal/catalog/old.go -> deleted)", res.Hits[0].String())
	assert.Equal(t, 1, res.Missing)
}

func TestRefs_ChangedModeOnMainReadsTheLastCommit(t *testing.T) {
	root := newKBRoot(t)
	newGitRepo(t, root)
	mustWriteFile(t, root, "docs/late.md", "`docs/nope.md`\n")
	gitFx(t, root, "add", "-A")
	gitFx(t, root, "commit", "-q", "-m", "late")
	mustWriteFile(t, root, "docs/uncommitted.md", "`docs/nope-too.md`\n")
	ix, _ := loadFixture(t, root)
	res, err := Refs(ix, RefsOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Files, "HEAD~1 against the working tree: the last commit's files, not untracked ones")
	assert.Equal(t, 1, res.Missing)
}

func TestCheckRepo_FoldsMissingReferencesIntoTheFindings(t *testing.T) {
	root := refsRoot(t)
	runGen(t, root)
	gitFx(t, root, "add", "-A")
	gitFx(t, root, "commit", "-q", "-m", "gen")
	ix, findings := loadFixture(t, root)
	all, err := CheckRepo(ix, findings)
	require.NoError(t, err)
	var got []string
	for _, f := range all {
		got = append(got, f.String())
	}
	assert.Contains(t, got, "docs/notes.md:2: reference `docs/nope.md` missing (path)")
	assert.Contains(t, got, "docs/notes.md:3: reference `make nope` missing (make target)")
	assert.Contains(t, got, `kb.yaml: refs whitelist entry "internal/catalog/catalog_test.go" now exists — remove it`)
	assert.NotContains(t, strings.Join(got, "\n"), "web/dist", "an ignored path is reported by kb refs, never a check finding")
	plain, err := Check(ix, findings)
	require.NoError(t, err)
	assert.Empty(t, plain, "Check itself needs no git and reports no reference")
}

func TestRefs_WithNoTopDirsChecksMakeTargetsAndFlagsOnly(t *testing.T) {
	root := refsRoot(t)
	mustWriteFile(t, root, ConfigPath, fixtureConfigCore+fixturePackConfig+"refs:\n  files: [\"*.md\"]\n  flags:\n    binary: svc\n    source: cmd/svc\n")
	ix, _ := loadFixture(t, root)
	res, err := Refs(ix, RefsOptions{All: true})
	require.NoError(t, err)
	var lines []string
	for _, h := range res.Hits {
		lines = append(lines, h.String())
	}
	assert.ElementsMatch(t, []string{
		"docs/notes.md:3  make nope  missing (make target)",
		"docs/notes.md:4  svc -nope  missing (flag)",
	}, lines, "no token is judged a repo path when refs.top_dirs is empty")
}
