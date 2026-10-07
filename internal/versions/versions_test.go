package versions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRecord_ReadsRowsInAnyOrderAndSkipsCommentsAndBlanks(t *testing.T) {
	rows, err := ParseRecord(strings.NewReader("# header\n\n1.4.0 2026-09-10 second\n  1.2.0 2026-08-29 first run  \n1.10.0 2026-09-20 third\n"))
	require.NoError(t, err)
	assert.Equal(t, []Row{
		{Version: "1.4.0", Date: "2026-09-10", Note: "second"},
		{Version: "1.2.0", Date: "2026-08-29", Note: "first run"},
		{Version: "1.10.0", Date: "2026-09-20", Note: "third"},
	}, rows)
	floor, ceiling := Range(rows)
	assert.Equal(t, "1.2.0", floor)
	assert.Equal(t, "1.10.0", ceiling, "numeric order, not lexical")
	floor, ceiling = Range(nil)
	assert.Empty(t, floor)
	assert.Empty(t, ceiling)
}

func TestParseRecord_RejectsAMalformedRowADuplicateAndAnEmptyRecord(t *testing.T) {
	_, err := ParseRecord(strings.NewReader("1.2.0 2026-08-29\n"))
	require.EqualError(t, err, `line 1: malformed row "1.2.0 2026-08-29" (want <x.y.z> <YYYY-MM-DD> <note>)`)
	_, err = ParseRecord(strings.NewReader("1.2.0 2026-08-29 a\n1.2.0 2026-08-30 b\n"))
	require.EqualError(t, err, `line 2: duplicate version "1.2.0"`)
	_, err = ParseRecord(strings.NewReader("# only a comment\n"))
	require.EqualError(t, err, "no version rows found")
}
