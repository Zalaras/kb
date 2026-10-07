package kb

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// VersionRow is one row of a version record: a version, the date it was recorded and a
// free-text note.
type VersionRow struct {
	Version string
	Date    string
	Note    string
}

// versionRowRE matches one record row: "<major.minor.patch> <YYYY-MM-DD> <note>".
var versionRowRE = regexp.MustCompile(`^(\d+\.\d+\.\d+) (\d{4}-\d{2}-\d{2}) (.+)$`)

// ParseVersionRecord reads the plain-text version record the config's versions.file names
// (kb:adr/version-record-is-a-plain-text-file): one `<x.y.z> <YYYY-MM-DD> <note>` row per
// line, `#` comments and blank lines allowed, rows in any order. A malformed row, a
// duplicate version or an empty record is an error, so a corrupt file fails loudly rather
// than silently narrowing the range.
func ParseVersionRecord(r io.Reader) ([]VersionRow, error) {
	var rows []VersionRow
	seen := map[string]bool{}
	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := versionRowRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("line %d: malformed row %q (want <x.y.z> <YYYY-MM-DD> <note>)", lineNo, line)
		}
		if seen[m[1]] {
			return nil, fmt.Errorf("line %d: duplicate version %q", lineNo, m[1])
		}
		seen[m[1]] = true
		rows = append(rows, VersionRow{Version: m[1], Date: m[2], Note: m[3]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no version rows found")
	}
	return rows, nil
}

// VersionRecordRange returns the lowest and highest version in rows, or two empty strings
// for no rows.
func VersionRecordRange(rows []VersionRow) (floor, ceiling string) {
	if len(rows) == 0 {
		return "", ""
	}
	versions := make([]string, len(rows))
	for i, r := range rows {
		versions[i] = r.Version
	}
	sort.Slice(versions, func(i, j int) bool { return compareVersion(versions[i], versions[j]) < 0 })
	return versions[0], versions[len(versions)-1]
}
