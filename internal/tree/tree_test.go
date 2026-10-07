package tree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_SkipsGitNodeModulesDistAndBinAndReturnsSlashPaths(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"go.mod", "internal/a/a.go", ".git/HEAD", "web/node_modules/x/index.js", "web/dist/app.js",
		"dist/svc", "bin/tool", "internal/assets/app.js", ".claude/rules/x.md",
		".hidden/secret.md", ".github/workflows/ci.yml", ".gitignore",
	} {
		writeFile(t, root, rel)
	}
	got, err := Walk(root, []string{".git", "node_modules", "web/dist", "dist", "bin", "internal/assets"}, []string{".claude", ".githooks", ".github"})
	require.NoError(t, err)
	assert.Equal(t, []string{".claude/rules/x.md", ".github/workflows/ci.yml", ".gitignore", "go.mod", "internal/a/a.go"}, got)
}

func writeFile(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
}
