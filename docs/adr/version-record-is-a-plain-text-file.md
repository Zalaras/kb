---
id: version-record-is-a-plain-text-file
type: decision
status: accepted
date: 2026-10-08
summary: A fact's verified range is checked against a plain-text version record the config names, one version, date and note per row, plus a configurable ceiling word.
features: []
tags: [model, compat]
files: [internal/versions/versions.go, internal/kb/load.go, internal/kb/record.go, internal/kb/check.go]
tests: [TestParseRecord_ReadsRowsInAnyOrderAndSkipsCommentsAndBlanks, TestLoad_ReadsTheVersionRecordFromDisk, TestCheck_FailsAFactVerifiedAboveTheCeilingOrBelowTheFloorButAllowsALowerBoundBelowFloor]
refs: []
supersedes: []
---
**Context.** A fact record states what was measured on a range of versions of some external
tool, and the check that its upper bound does not exceed what the repository has actually
observed is what keeps facts honest. Which tool, and how its observed versions are recorded,
is the repository's business, not the model's; the tool only needs a floor and a ceiling.

**Options.** (A) A Go interface the consumer implements, which a standalone binary cannot
take. (B) A plain-text record file named in `kb.yaml`, in a fixed row format the tool parses
itself. (C) The range as two literal config values, edited by hand.

**Decision.** B. `versions.file` names the record: one `<x.y.z> <YYYY-MM-DD> <note>` row per
line, `#` comments and blank lines allowed, rows in any order; the floor and ceiling are its
minimum and maximum. A malformed row, a duplicate or an empty file is a load error, never a
silently narrowed range. `versions.ceiling_word` (default `ceiling`) is the word a fact's
upper bound may carry to track the ceiling as it moves. An absent `versions.file` turns the
range check off and a fact's bounds must then both be literal versions.

**Consequences.** A repository that already keeps such a record for its own purposes points
the tool at it and keeps its own parser; the format is small enough that two implementations
of it are not a maintenance burden. The error text names the configured path.
