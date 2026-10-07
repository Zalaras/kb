package versions

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Row is one row of a version record: a version, the date it was recorded and a
// free-text note.
type Row struct {
	Version string
	Date    string
	Note    string
}

// rowRE matches one record row: "<major.minor.patch> <YYYY-MM-DD> <note>".
var rowRE = regexp.MustCompile(`^(\d+\.\d+\.\d+) (\d{4}-\d{2}-\d{2}) (.+)$`)

// ParseRecord reads the plain-text version record the config's versions.file names
// (kb:adr/version-record-is-a-plain-text-file): one `<x.y.z> <YYYY-MM-DD> <note>` row per
// line, `#` comments and blank lines allowed, rows in any order. A malformed row, a
// duplicate version or an empty record is an error, so a corrupt file fails loudly rather
// than silently narrowing the range.
func ParseRecord(r io.Reader) ([]Row, error) {
	var rows []Row
	seen := map[string]bool{}
	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := rowRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("line %d: malformed row %q (want <x.y.z> <YYYY-MM-DD> <note>)", lineNo, line)
		}
		if seen[m[1]] {
			return nil, fmt.Errorf("line %d: duplicate version %q", lineNo, m[1])
		}
		seen[m[1]] = true
		rows = append(rows, Row{Version: m[1], Date: m[2], Note: m[3]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no version rows found")
	}
	return rows, nil
}

// Range returns the lowest and highest version in rows, or two empty strings
// for no rows.
func Range(rows []Row) (floor, ceiling string) {
	if len(rows) == 0 {
		return "", ""
	}
	versions := make([]string, len(rows))
	for i, r := range rows {
		versions[i] = r.Version
	}
	sort.Slice(versions, func(i, j int) bool { return Compare(versions[i], versions[j]) < 0 })
	return versions[0], versions[len(versions)-1]
}

// Compare orders two x.y.z strings numerically, component by component.
func Compare(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3 && i < len(pa) && i < len(pb); i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
