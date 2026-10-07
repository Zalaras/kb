package kb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchGlob_DoubleStarSpansZeroOrMoreSegmentsAndStarNeverCrossesASlash(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"internal/catalog/**", "internal/catalog/catalog.go", true},
		{"internal/catalog/**", "internal/catalog/deep/er/file.go", true},
		{"internal/catalog/**", "internal/catalog", true},
		{"internal/**/catalog.go", "internal/catalog.go", true},
		{"internal/**/catalog.go", "internal/a/b/catalog.go", true},
		{"internal/*", "internal/catalog/catalog.go", false},
		{"internal/*/*.go", "internal/catalog/catalog.go", true},
		{"**/*.ts", "web/src/a/b.ts", true},
		{"**/*.ts", "b.ts", true},
		{"web/e2e/catalog.spec.ts", "web/e2e/catalog.spec.ts", true},
		{"web/e2e/catalog.spec.ts", "web/e2e/catalog.spec.tsx", false},
		{"internal/c?talog/**", "internal/catalog/x.go", true},
		{"internal/[st]ess/**", "internal/tess/x.go", true},
		{"**", "anything/at/all", true},
		{"internal/**", "web/x.ts", false},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+" vs "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, MatchGlob(tc.pattern, tc.path))
		})
	}
	assert.Equal(t, []string{"a/x.go", "a/y.go"}, Expand([]string{"a/x.go", "a/y.go", "b/z.go"}, "a/*.go"))
}

func TestMatchGlob_IsByteExactSoAMiscasedPatternMatchesNothing(t *testing.T) {
	assert.False(t, MatchGlob("Internal/catalog/**", "internal/catalog/catalog.go"))
	assert.False(t, MatchGlob("internal/catalog/Sess.go", "internal/catalog/catalog.go"))
}
