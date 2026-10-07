---
id: config-in-kb-yaml-model-in-code
type: decision
status: accepted
date: 2026-10-08
summary: Repository values live in a strict kb.yaml at the root the tool walks up to; the record model and generated-file grammar are code.
features: []
tags: [config, model]
files: [kb.yaml, internal/kb/config.go, cmd/kb/main.go]
tests: [TestLoadConfig_RejectsAnUnknownKeyAtAnyDepth, TestLoadConfig_GenericDefaultsLeaveEveryOptionalPassOffOrUnscoped, TestRun_GenThenCheckIsGreenFromASubdirectory]
refs: []
supersedes: []
---
**Context.** The tool serves any repository that keeps typed records, but two kinds of value
are in play: the record model (eight types, their statuses and fields, the `kb:<type>/<id>`
citation syntax, the generated-file markers) and the repository's own layout (where each
type lives, its closed tag and role lists, which passes run and over which directories, how
it spells the command that regenerates files). A repository must be able to adopt the tool
without a fork, and a record written for one repository must mean the same in another.

**Options.** (A) Flags and environment variables per value. (B) One `kb.yaml` at the repo
root, parsed strictly, holding every repository value, with the model in code. (C) A
config that can also redefine the model (types, fields, citation syntax).

**Decision.** B. `kb.yaml` is required and the repository root is the nearest ancestor of
the working directory that holds one. `dirs`, `tags` and `roles` are required; every other
key has a default that carries no repository's layout: an optional pass is off until its
key is set (no protocol document, no conventions slice, no version record, no agents
directory, no ownership roots, no refs path check, no scope roots) and pack scoping lists
are empty, so an unlisted role packs every section in full. Unknown keys fail at any depth
(kb:adr/yaml-parsed-strictly-with-goccy). Generated text spells the repository's own
commands from `commands.cli` and `commands.regen`.

**Consequences.** Adopting the tool is a `kb.yaml` and a records directory; nothing in the
code names a repository. The fixtures spell their config out in full, which is what keeps
the defaults honest. A new repository value is a new config key with a documented default,
never a constant.
