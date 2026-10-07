# kb

Go module `github.com/Zalaras/kb`: a library (root package `kb`) and a CLI (`cmd/kb`) that
index, check, generate from and bundle a repository's typed Markdown knowledge records.
Everything repository-specific comes from that repository's `kb.yaml`; nothing in this code
names a consumer. This repo keeps its own decision records under `docs/adr/` and runs the
tool over them.

## Commands

- `make check` — lint, tests, then `kb check` over this repo's own records. Run before a commit.
- `make test` / `go test ./...` — the tests build scratch repos under `t.TempDir()`; none reads this checkout.
- `go run ./cmd/kb <subcommand>` — the tool itself; `make gen-kb` regenerates `docs/INDEX.md`.
- `make hooks` once per clone arms the pre-commit identity guard.

## Rules

- No repository value in code. A directory name, role, tag, command string or path a consumer
  might spell differently is a `kb.yaml` key with a documented default (README § Configuration),
  and the default carries no repository's layout: optional passes default off, scoping lists empty.
- A change to the record model (types, fields, statuses, citation syntax, generated markers) or to
  a config key's meaning gets a decision record in `docs/adr/`, written like the ones there.
- `cmd/kb/main.go` only dispatches; logic and its table tests live in the root package.
- Never edit a generated file (`docs/INDEX.md` here); edit the records and run `make gen-kb`.
- Never run `git config user.*` in this checkout or any worktree of it; scratch repos in tests pass
  `-c user.name=… -c user.email=…` per command (`gitFx` in `fixture_test.go`).

## Gotchas

- `kb.yaml` is parsed strictly: an unknown key at any depth fails the load, so a misspelt section
  is an error, not a silently applied default.
- `kb gen` refuses to write while the sources have findings; fix the records first.
- `kb check` and `kb refs` need a git checkout with at least one commit (they read `git ls-files`
  and the branch); `Check` alone does not.
- The version record is read from disk on every run, never embedded, so a consumer's bump is
  visible without a rebuild.
- The fixtures (`fixture_test.go`) spell their `kb.yaml` out in three parts so a test can replace
  one section without a duplicate YAML key.
