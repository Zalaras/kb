# kb

[![release](https://img.shields.io/github/v/release/Zalaras/kb)](https://github.com/Zalaras/kb/releases/latest) [![ci](https://github.com/Zalaras/kb/actions/workflows/ci.yml/badge.svg)](https://github.com/Zalaras/kb/actions/workflows/ci.yml)

`kb` keeps a repository's knowledge as typed Markdown records with strict frontmatter, and
gates them the way a compiler gates code: every id, status, tag, citation, glob, test name
and version range is checked, the index files are generated rather than written, and an
agent or a person can ask for exactly the records a piece of work needs instead of reading
whole documents. It is one command, `kb`, built in Go; everything specific to a repository
lives in that repository's `kb.yaml`.

## Install

Pick one:

```sh
# A released binary, verified against the release's SHA-256 (macOS or Linux, amd64 or arm64)
curl -fsSL https://raw.githubusercontent.com/Zalaras/kb/main/scripts/install.sh | sh

# Pinned by a Go module (Go 1.24+), so every clone and CI run uses the same version
go get -tool github.com/Zalaras/kb/cmd/kb@latest
go tool kb check

# From source, anywhere Go runs
go install github.com/Zalaras/kb/cmd/kb@latest
```

The installer puts `kb` in `~/.local/bin` (`--bin-dir`, `--version`, `--arch`; `--help` lists
them). It downloads from the GitHub release and refuses an archive whose checksum does not
match `checksums.txt`. Prefer to read it first:

```sh
curl -fsSL -o install.sh https://raw.githubusercontent.com/Zalaras/kb/main/scripts/install.sh
less install.sh
sh install.sh
```

`kb version` prints the release a binary was built from.

## Quick start

1. Put a `kb.yaml` at the repository root. Three keys are required:

   ```yaml
   dirs:
     rule: docs/rules
     decision: docs/adr
     spec: docs/features
     diagram: docs/diagrams
     fact: docs/facts
     lesson: docs/lessons
     runbook: docs/runbooks
     reference: docs/references
   roles: [impl, review]
   tags: [testing, ux, deps]
   ```

2. Write a record. A decision at `docs/adr/errors-are-wrapped.md`:

   ```markdown
   ---
   id: errors-are-wrapped
   type: decision
   status: accepted
   date: 2026-10-08
   summary: Every error crossing a package boundary is wrapped with the operation that failed.
   features: []
   tags: [testing]
   files: [internal/**]
   tests: []
   refs: []
   supersedes: []
   ---
   **Context.** … **Options.** … **Decision.** … **Consequences.** …
   ```

3. Check and generate:

   ```sh
   kb check      # every invariant; exit 1 listing each finding
   kb gen        # writes docs/INDEX.md and the other generated files
   kb show errors-are-wrapped
   ```

   Cite the record from code or prose as `kb:adr/errors-are-wrapped`; `kb check` fails on a
   citation that resolves to nothing.

`kb` finds the repository root by walking up from the working directory to the nearest
`kb.yaml`, so it runs from any subdirectory. `kb check` and `kb refs` read `git ls-files`
and need a checkout with at least one commit.

## Records

Eight record types, each in its own directory. A record is one Markdown file with a strict,
flat frontmatter block (scalars and lists; no nesting) and a prose body under a word budget.

| type | directory key | cited as | type-specific fields |
|---|---|---|---|
| `rule` | `dirs.rule` | `kb:rule/<id>` | — |
| `decision` | `dirs.decision` | `kb:adr/<id>` | `supersedes: [<id>]`; statuses `accepted`, `proposed`, `superseded`, `rejected` |
| `spec` | `dirs.spec/<feature>/spec.md` | `kb:spec/<id>` | `go`, `web`, `e2e` (file globs the feature owns), `protocol` (anchor ids) |
| `diagram` | `dirs.diagram` | `kb:diagram/<id>` | `kind`: one of `context`, `container`, `component`, `domain`, `state`, `sequence`, `er`, `flow`; exactly one mermaid fence opening with that kind's keyword |
| `fact` | `dirs.fact` | `kb:fact/<id>` | `verified: <lo>..<hi>` (a version range), `guard: <TestName> \| none` |
| `lesson` | `dirs.lesson` | `kb:lesson/<id>` | `roles: [<role>]` from `roles` |
| `runbook` | `dirs.runbook` | `kb:runbook/<id>` | — |
| `reference` | `dirs.reference` | `kb:ref/<id>` | — |

Common fields: `id` (must equal the filename slug, or the feature directory for a spec),
`type`, `status` (`active`, `draft`, `retired`; decisions use their own four), `date`
(`YYYY-MM-DD`), `summary` (one line, 160 characters), `features`, `tags` (from the closed
list), `files` (globs that must match something), `tests` (Go test names that must exist),
`refs` (provenance: `plan:<name>`, `#<issue>`, a URL or a repo path).

**Features.** A spec's `go`, `web` and `e2e` globs are the feature registry: with
`ownership` roots configured, every source file under them must be owned by some feature,
and `kb for <path>` names the features and records covering a path.

**Anchors.** One document (`paths.protocol`) may carry `<!-- kb:anchor <id> -->` comments
before headings. A spec lists the anchors it depends on under `protocol`, the generated
`contract.md` slices those sections out, and `kb:anchor/<id>` cites one.

**Version ranges.** A fact's `verified` range is checked against a version record
(`versions.file`): a text file of `<x.y.z> <YYYY-MM-DD> <note>` rows whose minimum and
maximum are the floor and ceiling. The upper bound may be the configured `ceiling_word` to
track the ceiling as it moves.

## Generated files

`kb gen` renders, and `kb check` verifies, each of these; never edit them by hand:

- `paths.root_index` (default `docs/INDEX.md`): every live record by type, retired ones at the end.
- `<dirs.spec>/<feature>/INDEX.md` and `contract.md`: a feature's records and its anchored sections.
- `<paths.rules_dir>/<feature>.md` (default `.claude/rules/`): a per-feature rules file with a
  `paths:` frontmatter of the feature's globs, truncated to `budgets.rule_file_lines`.
- The `<!-- kb:features -->` and `<!-- kb:trailer -->` blocks inside any `CLAUDE.md` that carries them.

Each generated file opens with a header naming its content hash and the command that
regenerates it (`commands.regen`). A file whose body no longer matches its hash is reported
as hand-edited; one whose sources moved on is reported as stale.

## Commands

| command | does | exit |
|---|---|---|
| `kb check` | every invariant, plus the refs pass over every tracked file | 1 listing each finding |
| `kb gen` | regenerate; writes only changed files; refuses while sources have findings | 1 |
| `kb pack --plan NAME --role ROLE [--features a,b] [--touches c,d]` | one context bundle: rules, design docs, the plan's feature specs and contracts, diagrams, decisions, facts, lessons, runbooks, scoped per role; reports its word count | 0 (warns over `budgets.pack_words`) |
| `kb for PATH` / `kb why PATH` | the features and records covering a path / the decisions and facts explaining it | 0 |
| `kb show ID` / `kb cite ID` | one record rendered / its citation token and path | 1 if unknown |
| `kb find WORD...` | records whose text carries every word | 0 |
| `kb ls [--type T] [--feature F] [--status S] [--role R] [--guard none]` | filtered listing | 0 |
| `kb fences FILE...` | every mermaid fence in the files opens with an allowed keyword; needs no index | 1 |
| `kb refs [--all \| --changed \| FILE...]` | every repo path, `make` target and `<binary> -flag` cited in a comment or markdown line exists | 1 if any is missing |
| `kb scope --plan NAME [--touch]` | every changed source file's feature is in the plan's `**Features**` or `**Touches**` header; `--touch` widens Touches | 1 per unplaced file |
| `kb owners PATH...` | `path<TAB>feature[,feature]` per path, for scripts | 0 |
| `kb version` | the release this binary was built from; needs no repository | 0 |

A usage mistake exits 2. Findings print as `path:line: message`.

## Configuration

`kb.yaml` is parsed strictly: an unknown key at any depth is an error. Only `dirs`, `tags`
and `roles` are required. Every other key has a default that carries no repository's
layout; an optional pass stays off until its key is set.

| key | default | meaning |
|---|---|---|
| `dirs.<type>` | required | the directory each record type lives in; a spec is `<dirs.spec>/<feature>/spec.md` |
| `tags` | required | the closed tag list |
| `roles` | required | the roles a lesson may address and a pack may be requested as |
| `base_branch` | `main` | the branch `kb refs --changed` and `kb scope` diff against |
| `paths.protocol` | off | the one document carrying `kb:anchor` comments |
| `paths.conventions` | off | a document packs slice per role (`pack.conventions_sections`) |
| `paths.root_index` | `docs/INDEX.md` | the generated store index |
| `paths.rules_dir` | `.claude/rules` | where the generated per-feature rules files go |
| `paths.plans_dir` | `plans` | `plan:<name>` refs resolve to `<plans_dir>/<name>/plan.md` |
| `commands.cli` | `kb` | how generated text spells the command (for example `go tool kb`) |
| `commands.regen` | `kb gen` | the command the generated-file header names (for example `make gen-kb`) |
| `versions.file` | off | the version record a fact's `verified` range is checked against |
| `versions.ceiling_word` | `ceiling` | the word a fact's upper bound may carry to track the record's maximum |
| `agents.dir` | off | a directory of `<name>.md` agent files the role list must match, both ways |
| `agents.aliases` | `{}` | role → agent file name, when they differ |
| `agents.session_roles` | `[]` | roles with no agent file by design |
| `agents.agentless` | `[]` | agent files that read no pack and so have no role |
| `budgets.*` | 300 / 800 / 600 words (body / spec / runbook), 160 summary chars, 150 root CLAUDE.md lines, 400 nested CLAUDE.md words, 60 rule-file lines, 30000 pack words | hard limits `check` enforces (pack only warns) |
| `tree.skip_dirs` | `[.git, node_modules, dist, bin]` | directories the walk never enters |
| `tree.keep_hidden_dirs` | `[.claude, .github]` | the dot-directories it does enter |
| `citations.exclude` | `[]` | globs no citation scan, comment check or refs pass reads |
| `ownership` | off | source roots under which every `.go` and `.ts` file must be owned by a feature |
| `pack.touches_full_roles` | `[]` | roles that read a Touches feature in full |
| `pack.contractless_roles`, `pack.factless_roles`, `pack.decisionless_roles` | `[]` | roles whose pack replaces that section with a one-line pointer |
| `pack.system_diagram_roles` | `[]` | roles that read the feature-less diagrams |
| `pack.proposed_decision_roles` | `[]` | roles that read the plan's `proposed` decisions |
| `pack.design_docs.<role>` | `{}` | documents a role packs, whole or by level-two heading prefix (`sections`) |
| `pack.conventions_sections.<role>` | `{}` | the level-two sections of `paths.conventions` a role reads; an unlisted role reads it whole |
| `refs.files` | `["*.md", "*.go"]` | git pathspecs the refs pass reads |
| `refs.top_dirs` | off | a cited token is a repo path only if it opens with one of these; empty checks make targets and flags only |
| `refs.makefile` | `Makefile` | where `make <target>` tokens are checked |
| `refs.flags.binary`, `refs.flags.source` | off | a backticked `<binary> -flag` token is checked against the flags that directory's non-test Go sources define |
| `refs.whitelist` | `{}` | token → reason, for tokens that look like paths but are not; a whitelisted path that now exists is reported |
| `scope.roots` | off | source roots whose changed files need an owner in the plan's headers |
| `scope.skip` | `["**/CLAUDE.md"]` | files judged by what they mirror, not by their owner |

This repository's own `kb.yaml` is the smallest real example; the fixtures in
`internal/kb/fixture_test.go` spell out every key.

## Layout

`kb` is a command, not a library: every Go package is under `internal/`, so the contract
is the command line above, not a Go API. `cmd/kb` dispatches; `internal/kb` holds the
record model, the index and the subcommands; the leaf packages (`frontmatter`, `glob`,
`mermaid`, `markers`, `anchor`, `versions`, `tree`, `comments`, `gitx`) each parse or
walk one thing and know nothing about records.

## Development

`make check` runs the linter, the tests and `kb check` over this repository's own records in
`docs/adr/`, which is where its decisions are written up. `make hooks` arms the commit
guards once per clone: commits are conventional, and a `feat`, `fix`, `perf` or `refactor`
pushed to `main` cuts a GitHub release through GoReleaser (`.github/workflows/release.yml`),
with `make release-check` building the same archives locally into `dist/`.

## Licence

MIT. See `LICENSE`.
