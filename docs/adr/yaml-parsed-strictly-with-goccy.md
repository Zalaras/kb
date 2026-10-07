---
id: yaml-parsed-strictly-with-goccy
type: decision
status: accepted
date: 2026-10-08
summary: kb.yaml is decoded by github.com/goccy/go-yaml in strict mode; record frontmatter keeps the hand-rolled flat scanner.
features: []
tags: [deps, config]
files: [config.go, frontmatter.go, go.mod]
tests: [TestLoadConfig_RejectsAnUnknownKeyAtAnyDepth, TestParseFrontmatter_ReadsScalarsInlineListsAndBlockListsInOnePass]
refs: [https://github.com/goccy/go-yaml]
supersedes: []
---
**Context.** Record frontmatter is deliberately flat: scalars and lists only, no nesting,
because a record's schema is fixed and strictness is the point, so a small hand-rolled
scanner with exact error positions serves it. `kb.yaml` is different: it carries maps (a
directory per type, documents per role, a whitelist of token to reason) and nested
sections, and a typo in a key must fail rather than silently take a default.

**Options.** (A) `gopkg.in/yaml.v3`, archived and unmaintained. (B) `go.yaml.in/yaml/v3`,
the same code under new ownership, frozen to security fixes. (C) `github.com/goccy/go-yaml`,
independently written, actively maintained, MIT, with unknown-field rejection as a
first-class decode option. (D) Extend the flat scanner with dotted keys.

**Decision.** C for `kb.yaml`, decoded with `yaml.Strict()` into a typed struct, defaults
applied to nil fields afterwards so an explicit empty list stays empty. The frontmatter
scanner stays as it is.

**Consequences.** One direct dependency besides the test library. The two parsers serve two
grammars on purpose: a record never gains nesting because the config parser could read it.
