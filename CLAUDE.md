# kb

Go module `github.com/Zalaras/kb`: one command, `kb`, that indexes, checks, generates from
and bundles a repository's typed Markdown knowledge records. It is a CLI, not a library:
every Go package sits under `internal/`, and the contract is the command line (subcommands,
flags, output lines, exit codes), not the Go API. Everything repository-specific comes from
the consuming repository's `kb.yaml`; nothing in this code names a consumer. This repo keeps
its own decision records under `docs/adr/` and runs the tool over them.

## Layout

- `cmd/kb/` — the command; `main.go` only dispatches.
- `internal/kb/` — the core: `Config`, `Record`, `Index`, `Load`, check, gen, pack, the queries, refs and scope.
- `internal/{frontmatter,glob,mermaid,markers,anchor,versions,tree,comments,gitx,finding}` — leaf packages
  with one job each and no knowledge of records; they import nothing from `internal/kb`.

## Commands

- `make check` — lint, tests, then `kb check` over this repo's own records. Run before a commit.
- `make test` / `go test ./...` — the tests build scratch repos under `t.TempDir()`; none reads this checkout.
- `go run ./cmd/kb <subcommand>` — the tool itself; `make gen-kb` regenerates `docs/INDEX.md`.
- `make release-check` — validate `.goreleaser.yaml` and build a snapshot into `dist/`.
- `make hooks` once per clone arms the commit-msg and pre-commit guards.

## Rules

- No repository value in code. A directory name, role, tag, command string or path a consumer
  might spell differently is a `kb.yaml` key with a documented default (README § Configuration),
  and the default carries no repository's layout: optional passes default off, scoping lists empty.
- A change to the record model (types, fields, statuses, citation syntax, generated markers) or to
  a config key's meaning gets a decision record in `docs/adr/`, written like the ones there.
- A leaf package stays a leaf: it may import `internal/finding` and other leaves, never `internal/kb`.
- Commits are conventional (`.githooks/commit-msg`): `feat` and `fix`/`perf`/`refactor` cut a
  release from `main` through `.github/workflows/release.yml`; keep a releasing subject under 72
  characters and use `test`, `docs`, `ci`, `chore` or `build` for anything that ships no binary.
- Never edit a generated file (`docs/INDEX.md` here); edit the records and run `make gen-kb`.
- Never run `git config user.*` in this checkout or any worktree of it; scratch repos in tests pass
  `-c user.name=… -c user.email=…` per command (`gitFx` in `internal/kb/fixture_test.go`).

## Gotchas

- `kb.yaml` is parsed strictly: an unknown key at any depth fails the load, so a misspelt section
  is an error, not a silently applied default.
- `kb gen` refuses to write while the sources have findings; fix the records first.
- `kb check` and `kb refs` need a git checkout with at least one commit (they read `git ls-files`
  and the branch); `kb.Check` alone does not.
- The version record is read from disk on every run, never embedded, so a consumer's bump is
  visible without a rebuild.
- The core fixtures (`internal/kb/fixture_test.go`) spell their `kb.yaml` out in three parts so a
  test can replace one section without a duplicate YAML key.
