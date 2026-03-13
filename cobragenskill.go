// Package cobragenskill adds a "gen-skill" command to any Cobra-based CLI that generates
// an Agent Skill compatible with https://agentskills.io for use with Claude Code, Cursor,
// OpenAI Codex, and other supported agents.
//
// Usage:
//
//	import cobragenskill "github.com/theburrowhub/go-cobra-gen-skill"
//
//	func main() {
//	    root := &cobra.Command{Use: "mytool", Short: "My CLI tool"}
//	    cobragenskill.RegisterCommand(root,
//	        cobragenskill.WithVersion("1.2.0"),
//	        cobragenskill.WithLicense("MIT"),
//	    )
//	    root.Execute()
//	}
//
// The generated command ("mytool gen-skill") will:
//  1. Collect help text from the entire command tree
//  2. Try to use an AI agent (claude, codex, gemini) in headless mode to produce a rich skill
//  3. Fall back to a help-text-based skill if no agent is available
//  4. Ask the user whether to install globally or per-project
package cobragenskill

import (
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// RegisterCommand adds the "gen-skill" subcommand to rootCmd.
// Use Option functions to customise the skill metadata.
func RegisterCommand(rootCmd *cobra.Command, opts ...Option) {
	cfg := defaultConfig(rootCmd)
	for _, o := range opts {
		o(cfg)
	}
	rootCmd.AddCommand(newGenSkillCmd(rootCmd, cfg))
}

// sanitizeName converts an arbitrary string into a valid Agent Skill name:
// lowercase alphanumerics and hyphens, no leading/trailing/consecutive hyphens,
// max 64 characters.
func sanitizeName(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-{2,}`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 64 {
		s = s[:64]
		s = strings.TrimRight(s, "-")
	}
	if s == "" {
		s = "cli-tool"
	}
	return s
}

// yamlQuote returns a YAML-safe value for a scalar string field.
// Uses double-quoted style only when special characters are present.
func yamlQuote(s string) string {
	needsQuoting := strings.ContainsAny(s, `:#{}[]|>&'"`) ||
		strings.HasPrefix(s, "-") ||
		strings.HasPrefix(s, " ") ||
		strings.HasSuffix(s, " ") ||
		strings.Contains(s, "\n")
	if !needsQuoting {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}
