package cobragenskill

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newGenSkillCmd(rootCmd *cobra.Command, cfg *config) *cobra.Command {
	var (
		flagGlobal  bool
		flagProject bool
		flagNoAI    bool
		flagDryRun  bool
		flagAgent   string
		flagName    string
	)

	cmd := &cobra.Command{
		Use:   "gen-skill",
		Short: "Generate an Agent Skill for use with Claude, Cursor, Codex, and more",
		Long: `Generate an Agent Skill (SKILL.md) compatible with https://agentskills.io.

The skill teaches AI agents (Claude Code, Cursor, OpenAI Codex, and others) how to use
this CLI tool effectively, including all available commands, flags, and common workflows.

Generation strategy
  1. Collect help text from the entire command tree.
  2. Try to invoke an AI agent in headless mode to produce a rich, context-aware skill.
  3. If no agent is available (or --no-ai is set), fall back to a help-text-based skill.

Install locations (both are written for maximum compatibility)
  Global:  ~/.claude/skills/<name>/SKILL.md  and  ~/.agents/skills/<name>/SKILL.md
  Project: .claude/skills/<name>/SKILL.md   and  .agents/skills/<name>/SKILL.md`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenSkill(cmd, rootCmd, cfg, genSkillFlags{
				global:  flagGlobal,
				project: flagProject,
				noAI:    flagNoAI,
				dryRun:  flagDryRun,
				agent:   Agent(flagAgent),
				name:    sanitizeName(flagName),
			})
		},
	}

	f := cmd.Flags()
	f.BoolVar(&flagGlobal, "global", false,
		"Install skill globally (~/.claude/skills/ and ~/.agents/skills/)")
	f.BoolVar(&flagProject, "project", false,
		"Install skill in the current project (.claude/skills/ and .agents/skills/)")
	f.BoolVar(&flagNoAI, "no-ai", false,
		"Skip AI generation; use help-text fallback only")
	f.BoolVar(&flagDryRun, "dry-run", false,
		"Print the generated SKILL.md to stdout without writing any files")
	f.StringVar(&flagAgent, "agent", string(AgentAuto),
		"AI agent to use: auto | claude | codex | gemini | none")
	f.StringVar(&flagName, "name", cfg.skillName,
		"Override the skill name (default: app name, sanitized)")

	return cmd
}

type genSkillFlags struct {
	global  bool
	project bool
	noAI    bool
	dryRun  bool
	agent   Agent
	name    string
}

func runGenSkill(cmd *cobra.Command, rootCmd *cobra.Command, cfg *config, flags genSkillFlags) error {
	out := cmd.OutOrStdout()

	// Apply flag overrides to a copy so we don't mutate the shared config.
	localCfg := *cfg
	if flags.name != "" {
		localCfg.skillName = flags.name
	}
	if flags.noAI {
		localCfg.agent = AgentNone
	} else if flags.agent != "" && flags.agent != AgentAuto {
		localCfg.agent = flags.agent
	}

	fmt.Fprintf(out, "Collecting help from %s command tree...\n", rootCmd.Name())

	// Generate the skill content.
	var result GenerateResult
	if localCfg.agent == AgentNone {
		fmt.Fprintln(out, "AI generation disabled — using help-text fallback.")
		root := CollectHelp(rootCmd)
		result = GenerateResult{
			Content: generateFallback(&localCfg, rootCmd, root),
			Method:  "fallback",
		}
	} else {
		agentLabel := string(localCfg.agent)
		if localCfg.agent == AgentAuto {
			agentLabel = "auto (claude → codex → gemini)"
		}
		fmt.Fprintf(out, "Generating skill with AI agent (%s)...\n", agentLabel)
		result = Generate(&localCfg, rootCmd)
		if strings.HasPrefix(result.Method, "fallback") {
			fmt.Fprintln(out, "AI generation unavailable — using help-text fallback.")
		} else {
			fmt.Fprintf(out, "AI generation succeeded (%s).\n", result.Method)
		}
	}

	// Dry-run: just print.
	if flags.dryRun {
		fmt.Fprintf(out, "\n%s\n", strings.Repeat("-", 60))
		fmt.Fprintln(out, result.Content)
		fmt.Fprintln(out, strings.Repeat("-", 60))
		fmt.Fprintf(out, "\n(dry-run: no files written, method: %s)\n", result.Method)
		return nil
	}

	// Determine install scope.
	scope, err := resolveScope(out, os.Stdin, flags)
	if err != nil {
		return err
	}

	// Build targets and install.
	var targets []InstallTarget
	switch scope {
	case ScopeGlobal:
		targets = GlobalTargets(localCfg.skillName)
	case ScopeProject:
		targets = ProjectTargets(localCfg.skillName)
	}

	if err := Install(targets, result.Content); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Fprintf(out, "\nSkill %q installed successfully (method: %s)\n\n", localCfg.skillName, result.Method)
	for _, t := range targets {
		abs, _ := filepath.Abs(filepath.Join(t.SkillDir, "SKILL.md"))
		fmt.Fprintf(out, "  → %s\n", abs)
	}
	return nil
}

// resolveScope determines ScopeGlobal or ScopeProject from flags or user prompt.
func resolveScope(out io.Writer, in io.Reader, flags genSkillFlags) (InstallScope, error) {
	if flags.global && flags.project {
		return "", fmt.Errorf("--global and --project are mutually exclusive")
	}
	if flags.global {
		return ScopeGlobal, nil
	}
	if flags.project {
		return ScopeProject, nil
	}

	// Interactive prompt.
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Where would you like to install the skill?")
	fmt.Fprintln(out, "  [1] Global  — ~/.claude/skills/ and ~/.agents/skills/  (all projects)")
	fmt.Fprintln(out, "  [2] Project — .claude/skills/ and .agents/skills/      (this project only)")
	fmt.Fprint(out, "\nChoice [1/2] (default: 1): ")

	scanner := bufio.NewScanner(in)
	scanner.Scan()
	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "", "1":
		return ScopeGlobal, nil
	case "2":
		return ScopeProject, nil
	default:
		return "", fmt.Errorf("invalid choice %q — expected 1 or 2", choice)
	}
}
