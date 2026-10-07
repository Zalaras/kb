package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// checkAgents ties the config's role list to its agents directory
// (kb:adr/roles-checked-against-an-agents-directory): a role that packs for no agent file
// reaches nobody, and an agent file with no role can be addressed by no lesson. Off when
// agents.dir is empty.
func (c *checker) checkAgents() error {
	cfg := c.ix.Config
	if cfg.Agents.Dir == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(c.ix.Root, filepath.FromSlash(cfg.Agents.Dir)))
	if err != nil {
		return fmt.Errorf("reading agents.dir %s: %w", cfg.Agents.Dir, err)
	}
	agents := map[string]bool{}
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".md"); ok && !e.IsDir() {
			agents[name] = true
		}
	}
	c.checkRolesHaveAgents(agents)
	c.checkAgentsHaveRoles(agents)
	return nil
}

// checkRolesHaveAgents reports a role whose agent file (through its alias) is absent.
func (c *checker) checkRolesHaveAgents(agents map[string]bool) {
	cfg := c.ix.Config
	for _, role := range cfg.Roles {
		if contains(cfg.Agents.SessionRoles, role) {
			continue
		}
		name := role
		if a, ok := cfg.Agents.Aliases[role]; ok {
			name = a
		}
		if !agents[name] {
			c.fail(ConfigPath, 0, "role %q packs for no agent file %s/%s.md (list it under agents.session_roles if that is by design)", role, cfg.Agents.Dir, name)
		}
	}
}

// checkAgentsHaveRoles reports an agent file (through its alias) that no role names.
func (c *checker) checkAgentsHaveRoles(agents map[string]bool) {
	cfg := c.ix.Config
	roleOf := map[string]string{}
	for r, a := range cfg.Agents.Aliases {
		roleOf[a] = r
	}
	names := make([]string, 0, len(agents))
	for a := range agents {
		names = append(names, a)
	}
	sort.Strings(names)
	for _, agent := range names {
		if contains(cfg.Agents.Agentless, agent) {
			continue
		}
		role := agent
		if r, ok := roleOf[agent]; ok {
			role = r
		}
		if !contains(cfg.Roles, role) {
			c.fail(cfg.Agents.Dir+"/"+agent+".md", 0, "agent %q has no role in kb.yaml roles, so no lesson can reach it (list it under agents.agentless if it reads no pack)", agent)
		}
	}
}
