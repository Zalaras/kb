package comments

import (
	"regexp"
	"strings"
)

// Line is one numbered line of a file.
type Line struct {
	N    int
	Text string
}

var (
	fenceRE       = regexp.MustCompile("^\\s*(`{3,}|~{3,})")
	codeCommentRE = regexp.MustCompile(`^\s*(//|/\*|\*)`)
	hashCommentRE = regexp.MustCompile(`^\s*#`)
)

// Lines returns the lines of text that may carry a citation, by file type: every
// line outside a code fence for markdown, comment lines only for Go and TypeScript, and
// hash-comment lines for everything else (a shell script, a Makefile, a git hook).
func Lines(relpath, text string) []Line {
	if strings.HasSuffix(relpath, ".md") {
		return Prose(text)
	}
	re := hashCommentRE
	if strings.HasSuffix(relpath, ".go") || strings.HasSuffix(relpath, ".ts") {
		re = codeCommentRE
	}
	var out []Line
	for i, t := range strings.Split(text, "\n") {
		if re.MatchString(t) {
			out = append(out, Line{N: i + 1, Text: t})
		}
	}
	return out
}

// Prose returns the markdown lines outside code fences, fence lines excluded.
func Prose(text string) []Line {
	var out []Line
	fence := ""
	for i, t := range strings.Split(text, "\n") {
		if m := fenceRE.FindStringSubmatch(t); m != nil {
			switch {
			case fence == "":
				fence = m[1]
			case m[1][0] == fence[0] && len(m[1]) >= len(fence):
				fence = ""
			}
			continue
		}
		if fence == "" {
			out = append(out, Line{N: i + 1, Text: t})
		}
	}
	return out
}
