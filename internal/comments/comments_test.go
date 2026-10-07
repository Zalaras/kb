package comments

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLines_UsesTheDeadRefsRuleSetPerFileType(t *testing.T) {
	md := "prose kb:adr/a\n```\nfenced kb:adr/b\n```\nafter\n~~~\nalso fenced\n~~~\ntail\n"
	var mdTexts []string
	for _, ln := range Lines("docs/x.md", md) {
		mdTexts = append(mdTexts, ln.Text)
	}
	assert.Equal(t, []string{"prose kb:adr/a", "after", "tail", ""}, mdTexts)

	goSrc := "package x\n\n// line comment\nfunc f() { s := \"kb:adr/in-code\" }\n/* block\n * inner\n */\n"
	var goLines []int
	for _, ln := range Lines("internal/x.go", goSrc) {
		goLines = append(goLines, ln.N)
	}
	assert.Equal(t, []int{3, 5, 6, 7}, goLines)

	ts := "const a = 1; // trailing is not a comment line\n  // indented comment\n"
	tsLines := Lines("web/src/x.ts", ts)
	require.Len(t, tsLines, 1)
	assert.Equal(t, 2, tsLines[0].N)

	shLines := Lines("scripts/x.sh", "# kb:adr/a\necho kb:adr/not-a-comment\n  # indented\n")
	require.Len(t, shLines, 2, "other file types carry citations in hash-comment lines only")
	assert.Equal(t, []int{1, 3}, []int{shLines[0].N, shLines[1].N})
	assert.Len(t, Lines("Makefile", "# kb:adr/a\nbuild:\n"), 1)
}
