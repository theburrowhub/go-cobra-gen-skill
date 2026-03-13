package cobragenskill

import "github.com/spf13/cobra"

// Agent identifies an AI agent CLI that can be invoked in headless mode.
type Agent string

const (
	// AgentAuto tries claude → codex → gemini in order, using whichever is found.
	AgentAuto Agent = "auto"
	// AgentClaude uses Claude Code (`claude --print`).
	AgentClaude Agent = "claude"
	// AgentCodex uses OpenAI Codex CLI (`codex --no-interactive`).
	AgentCodex Agent = "codex"
	// AgentGemini uses Gemini CLI (`gemini`).
	AgentGemini Agent = "gemini"
	// AgentNone skips AI generation entirely and uses the help-text fallback.
	AgentNone Agent = "none"
)

// config holds the resolved configuration for gen-skill.
type config struct {
	skillName   string
	description string
	agent       Agent
	agentBin    string         // custom path to the agent binary
	targets     []ClientTarget // nil = auto (derived from generation agent at runtime)
	license     string
	version     string
	metadata    map[string]string
}

func defaultConfig(rootCmd *cobra.Command) *config {
	return &config{
		skillName: sanitizeName(rootCmd.Name()),
		agent:     AgentAuto,
		metadata:  make(map[string]string),
	}
}

// Option is a functional option for RegisterCommand.
type Option func(*config)

// WithSkillName overrides the skill name (must be valid: lowercase, hyphens, max 64 chars).
func WithSkillName(name string) Option {
	return func(c *config) { c.skillName = sanitizeName(name) }
}

// WithDescription sets a custom description for the skill frontmatter.
// If not set, the description is derived from the root command's Short field.
func WithDescription(desc string) Option {
	return func(c *config) { c.description = desc }
}

// WithAgent sets the AI agent to use for headless generation.
func WithAgent(a Agent) Option {
	return func(c *config) { c.agent = a }
}

// WithAgentBin sets a custom path to the agent binary.
func WithAgentBin(path string) Option {
	return func(c *config) { c.agentBin = path }
}

// WithLicense sets the license field in the skill frontmatter.
func WithLicense(lic string) Option {
	return func(c *config) { c.license = lic }
}

// WithVersion adds a "version" key to the skill metadata.
func WithVersion(v string) Option {
	return func(c *config) { c.version = v }
}

// WithMetadata adds an arbitrary key-value pair to the skill metadata section.
func WithMetadata(key, value string) Option {
	return func(c *config) { c.metadata[key] = value }
}

// WithTargets sets the agent clients to install the skill for.
// By default the target is derived from the generation agent at runtime.
// Accepts the same values as the --for flag: claude, codex, gemini, cursor, agents, all.
func WithTargets(targets ...ClientTarget) Option {
	return func(c *config) { c.targets = targets }
}
