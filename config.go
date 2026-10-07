package kb

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// ConfigPath is the repo-root file that carries every repo-specific value the tool reads:
// the record directories, the closed tag and role lists, the paths it writes, the budgets,
// the pack scoping per role and the refs and scope settings. Load fails without it. The
// record model itself — types, statuses, fields, citation syntax, generated markers — is
// code, not configuration (kb:adr/config-in-kb-yaml-model-in-code).
const ConfigPath = "kb.yaml"

// Config is the parsed kb.yaml. Unknown keys fail at any depth; dirs, tags and roles are
// required; every other key defaults to the value documented beside its field, and an
// explicit empty list stays empty.
type Config struct {
	Dirs       map[Type]string `yaml:"dirs"`
	Tags       []string        `yaml:"tags"`
	Roles      []string        `yaml:"roles"`
	BaseBranch string          `yaml:"base_branch"`
	Paths      PathsConfig     `yaml:"paths"`
	Commands   CommandsConfig  `yaml:"commands"`
	Versions   VersionsConfig  `yaml:"versions"`
	Agents     AgentsConfig    `yaml:"agents"`
	Budgets    Budgets         `yaml:"budgets"`
	Tree       TreeConfig      `yaml:"tree"`
	Citations  CitationsConfig `yaml:"citations"`
	Ownership  []string        `yaml:"ownership"`
	Pack       PackConfig      `yaml:"pack"`
	Refs       RefsConfig      `yaml:"refs"`
	Scope      ScopeConfig     `yaml:"scope"`
}

// PathsConfig names the files and directories the tool reads and writes outside the record
// directories. Protocol is the one document whose sections carry kb:anchor comments and
// Conventions the document packs slice per role; either may be empty, which turns that
// part off. RootIndex, RulesDir and PlansDir have defaults.
type PathsConfig struct {
	Protocol    string `yaml:"protocol"`
	RootIndex   string `yaml:"root_index"`
	RulesDir    string `yaml:"rules_dir"`
	Conventions string `yaml:"conventions"`
	PlansDir    string `yaml:"plans_dir"`
}

// CommandsConfig is how the repo spells the tool's own invocations in generated text: CLI
// is what a reader types to run a subcommand (default "kb"), Regen the command that
// regenerates the generated files (default "kb gen").
type CommandsConfig struct {
	CLI   string `yaml:"cli"`
	Regen string `yaml:"regen"`
}

// VersionsConfig names the version record a fact's verified range is checked against:
// File is a text file of `<x.y.z> <YYYY-MM-DD> <note>` rows whose minimum and maximum are
// the floor and ceiling (kb:adr/version-record-is-a-plain-text-file), and CeilingWord the
// word a verified upper bound may carry to track that ceiling (default "ceiling"). An
// empty File turns the range check off.
type VersionsConfig struct {
	File        string `yaml:"file"`
	CeilingWord string `yaml:"ceiling_word"`
}

// AgentsConfig ties the role list to a directory of agent files, one <name>.md per agent
// (kb:adr/roles-checked-against-an-agents-directory). With Dir set, check reports a role
// with no agent file and an agent file with no role. Aliases maps a role to the agent file
// it packs for when the names differ; SessionRoles are roles with no agent file by design;
// Agentless are agent files that read no pack and so have no role.
type AgentsConfig struct {
	Dir          string            `yaml:"dir"`
	Aliases      map[string]string `yaml:"aliases"`
	SessionRoles []string          `yaml:"session_roles"`
	Agentless    []string          `yaml:"agentless"`
}

// TreeConfig shapes the walk: directories never entered, and the hidden directories that are.
type TreeConfig struct {
	SkipDirs       []string `yaml:"skip_dirs"`
	KeepHiddenDirs []string `yaml:"keep_hidden_dirs"`
}

// CitationsConfig lists the globs no citation scan, kb comment check or refs pass reads.
type CitationsConfig struct {
	Exclude []string `yaml:"exclude"`
}

// DesignDoc is one document a role packs, whole or by its level-two sections.
type DesignDoc struct {
	Path     string   `yaml:"path"`
	Sections []string `yaml:"sections,omitempty"`
}

// PackConfig scopes a pack per role: a role named in a *less list has that section replaced
// by a one-line pointer, a role absent from every list and map packs everything.
type PackConfig struct {
	TouchesFullRoles      []string               `yaml:"touches_full_roles"`
	ContractlessRoles     []string               `yaml:"contractless_roles"`
	FactlessRoles         []string               `yaml:"factless_roles"`
	DecisionlessRoles     []string               `yaml:"decisionless_roles"`
	SystemDiagramRoles    []string               `yaml:"system_diagram_roles"`
	ProposedDecisionRoles []string               `yaml:"proposed_decision_roles"`
	DesignDocs            map[string][]DesignDoc `yaml:"design_docs"`
	ConventionsSections   map[string][]string    `yaml:"conventions_sections"`
}

// RefsConfig shapes kb refs: the tracked files it reads, the directories a cited path may
// open with (empty turns the path check off, leaving make targets and flags), the make
// file and flag source it checks tokens against, and the tokens that look like paths but
// are not.
type RefsConfig struct {
	Files     []string          `yaml:"files"`
	TopDirs   []string          `yaml:"top_dirs"`
	Makefile  string            `yaml:"makefile"`
	Flags     FlagsConfig       `yaml:"flags"`
	Whitelist map[string]string `yaml:"whitelist"`
}

// FlagsConfig names the binary whose flags a backticked `binary -flag` token is checked
// against and the directory its non-test Go sources define them in. An empty binary turns
// the check off.
type FlagsConfig struct {
	Binary string `yaml:"binary"`
	Source string `yaml:"source"`
}

// ScopeConfig shapes kb scope: the source roots whose changed files need an owner in the
// plan's headers (empty means scope is not configured), and the globs judged by what they
// mirror rather than by their owner.
type ScopeConfig struct {
	Roots []string `yaml:"roots"`
	Skip  []string `yaml:"skip"`
}

// LoadConfig reads and validates root/kb.yaml, applying the documented defaults to every
// absent optional key.
func LoadConfig(root string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(root, ConfigPath))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s is missing: the kb tool reads its repo settings from it", ConfigPath)
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ConfigPath, err)
	}
	cfg := &Config{}
	if err := yaml.UnmarshalWithOptions(data, cfg, yaml.Strict()); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ConfigPath, err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigPath, err)
	}
	return cfg, nil
}

// DefaultConfig returns the values the optional keys take when absent, plus one directory
// per type under docs/; a real repo spells dirs, tags and roles out in its kb.yaml.
func DefaultConfig() *Config {
	cfg := &Config{
		Dirs: map[Type]string{
			TypeRule: "docs/rules", TypeDecision: "docs/adr", TypeSpec: "docs/features", TypeDiagram: "docs/diagrams",
			TypeFact: "docs/facts", TypeLesson: "docs/lessons", TypeRunbook: "docs/runbooks", TypeReference: "docs/references",
		},
	}
	cfg.applyDefaults()
	return cfg
}

func (c *Config) applyDefaults() {
	c.BaseBranch = orDefault(c.BaseBranch, "main")
	c.Paths.RootIndex = orDefault(c.Paths.RootIndex, "docs/INDEX.md")
	c.Paths.RulesDir = orDefault(c.Paths.RulesDir, ".claude/rules")
	c.Paths.PlansDir = orDefault(c.Paths.PlansDir, "plans")
	c.Commands.CLI = orDefault(c.Commands.CLI, "kb")
	c.Commands.Regen = orDefault(c.Commands.Regen, "kb gen")
	c.Versions.CeilingWord = orDefault(c.Versions.CeilingWord, "ceiling")
	c.Budgets.applyDefaults()
	c.Tree.SkipDirs = orList(c.Tree.SkipDirs, ".git", "node_modules", "dist", "bin")
	c.Tree.KeepHiddenDirs = orList(c.Tree.KeepHiddenDirs, ".claude", ".github")
	c.Citations.Exclude = orList(c.Citations.Exclude)
	c.Ownership = orList(c.Ownership)
	c.Pack.applyDefaults()
	c.Refs.Files = orList(c.Refs.Files, "*.md", "*.go")
	c.Refs.TopDirs = orList(c.Refs.TopDirs)
	c.Refs.Makefile = orDefault(c.Refs.Makefile, "Makefile")
	c.Scope.Roots = orList(c.Scope.Roots)
	c.Scope.Skip = orList(c.Scope.Skip, "**/CLAUDE.md")
	for i, d := range c.Ownership {
		c.Ownership[i] = strings.TrimSuffix(d, "/")
	}
	for i, d := range c.Scope.Roots {
		c.Scope.Roots[i] = strings.TrimSuffix(d, "/")
	}
	for t, d := range c.Dirs {
		c.Dirs[t] = strings.TrimSuffix(d, "/")
	}
	c.Paths.RulesDir = strings.TrimSuffix(c.Paths.RulesDir, "/")
	c.Paths.PlansDir = strings.TrimSuffix(c.Paths.PlansDir, "/")
	c.Agents.Dir = strings.TrimSuffix(c.Agents.Dir, "/")
}

// applyDefaults leaves every scoping list empty: with no role named, each role packs every
// section in full, and a repo opts into scoping list by list.
func (p *PackConfig) applyDefaults() {
	p.TouchesFullRoles = orList(p.TouchesFullRoles)
	p.ContractlessRoles = orList(p.ContractlessRoles)
	p.FactlessRoles = orList(p.FactlessRoles)
	p.DecisionlessRoles = orList(p.DecisionlessRoles)
	p.SystemDiagramRoles = orList(p.SystemDiagramRoles)
	p.ProposedDecisionRoles = orList(p.ProposedDecisionRoles)
	if p.DesignDocs == nil {
		p.DesignDocs = map[string][]DesignDoc{}
	}
	if p.ConventionsSections == nil {
		p.ConventionsSections = map[string][]string{}
	}
}

// validate holds the required keys to their contract: a directory for every type, and a
// non-empty tag and role list.
func (c *Config) validate() error {
	for _, t := range typeOrder {
		if c.Dirs[t] == "" {
			return fmt.Errorf("dirs is missing type %q", t)
		}
	}
	for t := range c.Dirs {
		if !contains(typeNames(), string(t)) {
			return fmt.Errorf("dirs names unknown type %q (want %s)", t, joinOr(typeNames()))
		}
	}
	if len(c.Tags) == 0 {
		return errors.New("tags must list at least one tag")
	}
	if len(c.Roles) == 0 {
		return errors.New("roles must list at least one role")
	}
	return nil
}

// bodyBudget is the word budget for a record type.
func (c *Config) bodyBudget(t Type) int {
	switch t {
	case TypeSpec:
		return c.Budgets.SpecWords
	case TypeRunbook:
		return c.Budgets.RunbookWords
	default:
		return c.Budgets.BodyWords
	}
}

// Excluded reports whether a citations.exclude glob matches rel.
func (c *Config) Excluded(rel string) bool {
	for _, g := range c.Citations.Exclude {
		if MatchGlob(g, rel) {
			return true
		}
	}
	return false
}

// SpecDir is the feature directory; a feature's spec sits at SpecDir/<name>/spec.md.
func (c *Config) SpecDir() string { return c.Dirs[TypeSpec] }

// PlanPath is the plan file for a named plan.
func (c *Config) PlanPath(plan string) string { return c.Paths.PlansDir + "/" + plan + "/plan.md" }

// Command renders a CLI invocation the way this repo spells it: "<cli> <args>".
func (c *Config) Command(args string) string { return c.Commands.CLI + " " + args }

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

// orList keeps an explicit list, empty included, and fills only an absent (nil) one; with
// no defaults it turns nil into an empty list.
func orList(v []string, d ...string) []string {
	if v == nil {
		return append([]string{}, d...)
	}
	return v
}
