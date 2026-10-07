---
id: touches-pack-as-spec-and-contract
type: decision
status: accepted
date: 2026-10-08
summary: Touches names features a plan edits without changing; they pack as spec and contract only, except for configured roles, and kb scope can widen the header.
features: []
tags: [model]
files: [internal/kb/pack.go, internal/kb/scope.go]
tests: [TestPack_TouchedFeaturesPackSpecAndContractOnlyExceptForReview, TestScope_TouchWidensTheTouchesHeaderWithoutDuplicates]
refs: []
supersedes: []
---
**Context.** A plan names the features it changes in a `**Features**` header, and every
source file a branch touches must belong to one of them. Real work brushes other features:
a shared helper, a type both sides import. Naming each as a full feature packs its whole
record set into every role's context, and stopping the run to ask is a cost out of
proportion to a one-line edit.

**Options.** (A) Features only; a brushed feature is a stop. (B) A second header,
`**Touches**`, whose features pack as spec body and contract only, with roles the config
names (`pack.touches_full_roles`) reading them in full, and `kb scope --touch` appending a
missing owner to it instead of failing. (C) Infer touched features from the diff at pack time.

**Decision.** B. A feature in both headers is a Features one. The widening is explicit in
the plan file, so a reviewer sees what the branch reached into, and the roles that act as
the safety net for a settled decision a light edit breaks read the touched feature whole.

**Consequences.** Packs stay small for implementers; the scope gate widens without a stop and
promotion from Touches to Features is a judgement made on evidence, by hand.
