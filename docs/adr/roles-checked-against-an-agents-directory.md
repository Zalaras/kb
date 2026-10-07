---
id: roles-checked-against-an-agents-directory
type: decision
status: accepted
date: 2026-10-08
summary: With agents.dir set, check ties the role list to the agent files, one <name>.md each, through aliases, session roles and agentless files named in the config.
features: []
tags: [config, model]
files: [agents.go, config.go]
tests: [TestCheck_TiesTheRoleListToTheAgentsDirectoryWhenConfigured]
refs: []
supersedes: []
---
**Context.** A lesson record addresses roles, and a pack is requested as a role. In a
repository that spawns agents from files, a role that matches no agent file is a lesson
nobody reads, and an agent file with no role is an agent no lesson can reach. The role list
was once seeded from pipeline step names rather than from what is spawned, and the drift
went unnoticed until a lesson failed to arrive.

**Options.** (A) Leave the correspondence to the consumer's own test. (B) An optional
`agents` section: the directory, the roles that pack under a different file name, the
roles the main session plays with no file, and the files that read no pack; `check`
reports both directions. (C) Derive the role list from the directory and drop `roles`.

**Decision.** B. `roles` stays the explicit closed list every record is validated against,
and the directory check is an opt-in that `kb check` runs with everything else, so it holds
on every gate rather than in one test file.

**Consequences.** A repository with agent files lists four small keys and gains a finding
whenever the two lists diverge; one without them sets nothing and the check is off.
