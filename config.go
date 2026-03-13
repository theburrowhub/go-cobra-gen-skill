package cobragenskill

import "github.com/spf13/cobra"

// Agent identifies an AI agent CLI that can be invoked in headless mode.
type Agent string

const (
	AgentClaude Agent = "claude"
	AgentCodex  Agent = "codex"
	AgentGemini Agent = "gemini"
)

// config holds the resolved configuration for gen-skill.
type config struct {
	skillName   string
	description string
	agent       Agent
	agentBin    string // custom path to the agent binary
	license     string
	version     string
	metadata    map[string]string
}

func defaultConfig(rootCmd *cobra.Command) *config {
	return &config{
		skillName: sanitizeName(rootCmd.Name()),
		agent:     AgentClaude,
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
