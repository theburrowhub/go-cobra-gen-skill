package cobragenskill

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// GenerateResult holds the produced SKILL.md content and metadata about how it was made.
type GenerateResult struct {
	// Content is the full SKILL.md text ready to write to disk.
	Content string
	// Method describes how the skill was generated ("ai:claude", "ai:codex", "fallback", etc.).
	Method string
}

// Generate tries to produce a SKILL.md using the configured AI agent and falls
// back to the help-text-based generator if the agent is unavailable or, when
// validate is true, returns output that does not look like a valid SKILL.md.
func Generate(cfg *config, rootCmd *cobra.Command, validate bool) GenerateResult {
	root := CollectHelp(rootCmd)

	prompt := buildPrompt(cfg.skillName, rootCmd.Name(), root)
	out, err := InvokeAgent(cfg.agent, cfg.agentBin, prompt)
	if err == nil && (!validate || looksLikeSkillMD(out)) {
		return GenerateResult{Content: out, Method: "ai:" + string(cfg.agent)}
	}

	return GenerateResult{
		Content: generateFallback(cfg, rootCmd, root),
		Method:  "fallback",
	}
}

// looksLikeSkillMD does a quick sanity-check that the AI output begins with
// YAML frontmatter and contains a name field.
func looksLikeSkillMD(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "---") &&
		strings.Contains(s, "name:")
}

// buildPrompt returns the prompt string sent to the AI agent.
func buildPrompt(skillName, appName string, root *CommandNode) string {
	commandsText := FormatForPrompt(appName, root)
	return fmt.Sprintf(`You are generating an Agent Skill (SKILL.md) for the CLI tool %q.

Agent Skills (https://agentskills.io) let AI coding agents understand how to use CLI tools.

Below is the complete help output for every command in the tool:

%s

Generate a valid SKILL.md file with:

1. YAML frontmatter (between --- delimiters):
   - name: %s
   - description: (max 1024 chars; what the tool does AND when an agent should activate this skill; include specific keywords)
   - compatibility: (e.g. "Requires %s to be installed and available in PATH")

2. Markdown body with these sections:
   ## Overview
   What this tool does and its primary use cases.

   ## When to use this skill
   Specific conditions or user requests that should trigger activating this skill.

   ## Commands reference
   Key subcommands with their purpose and important flags.

   ## Common workflows
   Step-by-step examples for the most common tasks.

   ## Tips and gotchas
   Important flags, common mistakes, and helpful hints.

Rules:
- Output ONLY the SKILL.md content — no preamble, no explanation.
- Start your output directly with ---
- Keep the body under 500 lines.
- Use fenced code blocks for shell examples.`,
		appName, commandsText, skillName, appName)
}

// generateFallback produces a SKILL.md from the cobra help text when no AI is available.
func generateFallback(cfg *config, rootCmd *cobra.Command, root *CommandNode) string {
	appName := rootCmd.Name()
	var sb strings.Builder

	// --- Frontmatter ---
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", cfg.skillName)

	desc := cfg.description
	if desc == "" {
		if rootCmd.Short != "" {
			desc = fmt.Sprintf("%s. Use when the user needs to run %s commands or interact with %s.", rootCmd.Short, appName, appName)
		} else {
			desc = fmt.Sprintf("CLI tool %s. Use when the user asks to run %s commands.", appName, appName)
		}
	}
	if len(desc) > 1024 {
		desc = desc[:1021] + "..."
	}
	fmt.Fprintf(&sb, "description: %s\n", yamlQuote(desc))

	if cfg.license != "" {
		fmt.Fprintf(&sb, "license: %s\n", cfg.license)
	}

	fmt.Fprintf(&sb, "compatibility: Requires %s to be installed and available in PATH\n", appName)

	hasMetadata := cfg.version != "" || len(cfg.metadata) > 0
	if hasMetadata {
		sb.WriteString("metadata:\n")
		if cfg.version != "" {
			fmt.Fprintf(&sb, "  version: %s\n", yamlQuote(cfg.version))
		}
		for k, v := range cfg.metadata {
			fmt.Fprintf(&sb, "  %s: %s\n", k, yamlQuote(v))
		}
	}
	sb.WriteString("---\n\n")

	// --- Body ---
	fmt.Fprintf(&sb, "# %s\n\n", appName)

	if rootCmd.Long != "" {
		fmt.Fprintf(&sb, "%s\n\n", strings.TrimSpace(rootCmd.Long))
	} else if rootCmd.Short != "" {
		fmt.Fprintf(&sb, "%s\n\n", rootCmd.Short)
	}

	sb.WriteString("## When to use this skill\n\n")
	fmt.Fprintf(&sb, "Activate this skill whenever the user asks to run `%s` commands, needs help with %s, ", appName, appName)
	sb.WriteString("or when the task involves automating or scripting operations provided by this tool.\n\n")

	sb.WriteString("## Commands reference\n\n")
	writeCommandsRef(&sb, appName, root, 3)

	sb.WriteString("## Usage notes\n\n")
	fmt.Fprintf(&sb, "- Run `%s --help` or `%s <command> --help` to inspect available flags.\n", appName, appName)
	fmt.Fprintf(&sb, "- Skill generated on %s using built-in help text.\n", time.Now().Format("2006-01-02"))

	return sb.String()
}

// writeCommandsRef recursively writes a Markdown section for each command node.
func writeCommandsRef(sb *strings.Builder, appName string, node *CommandNode, headingLevel int) {
	heading := strings.Repeat("#", headingLevel)

	if node.Path == "" {
		fmt.Fprintf(sb, "%s `%s` (root)\n\n", heading, appName)
	} else {
		fmt.Fprintf(sb, "%s `%s %s`\n\n", heading, appName, node.Path)
	}

	if node.Short != "" {
		fmt.Fprintf(sb, "%s\n\n", node.Short)
	}

	fmt.Fprintf(sb, "<details>\n<summary>Help output</summary>\n\n```\n%s\n```\n\n</details>\n\n",
		strings.TrimSpace(node.HelpText))

	for _, child := range node.Children {
		writeCommandsRef(sb, appName, child, headingLevel+1)
	}
}
