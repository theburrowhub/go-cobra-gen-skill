package cobragenskill

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newGenSkillCmd(rootCmd *cobra.Command, cfg *config) *cobra.Command {
	var (
		flagScope  string
		flagNoAI   bool
		flagDryRun bool
		flagAgent  string
		flagName   string
	)

	cmd := &cobra.Command{
		Use:   "gen-skill",
		Short: "Generate an Agent Skill for use with Claude, Cursor, Codex, and more",
		Long: `Generate an Agent Skill (SKILL.md) compatible with https://agentskills.io.

The selected agent is used both to generate the skill and to determine
the install directory. .agents/skills/ is always included as well
(cross-client convention scanned by all compliant agents).

  --agent claude  → generates with Claude, installs in .claude/skills/ + .agents/skills/
  --agent codex   → generates with Codex,  installs in .codex/skills/  + .agents/skills/
  --agent gemini  → generates with Gemini, installs in .gemini/skills/ + .agents/skills/

If the agent binary is not found or fails, the command falls back to
generating the skill from the built-in help text.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := parseScope(flagScope)
			if err != nil {
				return err
			}
			agent := Agent(flagAgent)
			if _, ok := knownAgents[agent]; !ok {
				return fmt.Errorf("invalid --agent %q — valid values: claude, codex, gemini", flagAgent)
			}
			return runGenSkill(cmd, rootCmd, cfg, genSkillFlags{
				scope:  scope,
				noAI:   flagNoAI,
				dryRun: flagDryRun,
				agent:  agent,
				name:   sanitizeName(flagName),
			})
		},
	}

	f := cmd.Flags()
	f.StringVar(&flagScope, "scope", "project",
		"Where to install: project (./.<agent>/skills/) or global (~/.<agent>/skills/)")
	f.BoolVar(&flagNoAI, "no-ai", false,
		"Skip AI generation; use help-text fallback only")
	f.BoolVar(&flagDryRun, "dry-run", false,
		"Print the generated SKILL.md to stdout without writing any files")
	f.StringVar(&flagAgent, "agent", string(AgentClaude),
		"Agent to generate and install for: claude | codex | gemini")
	f.StringVar(&flagName, "name", cfg.skillName, "Skill name")

	return cmd
}

func parseScope(s string) (InstallScope, error) {
	switch strings.ToLower(s) {
	case "project", "":
		return ScopeProject, nil
	case "global":
		return ScopeGlobal, nil
	default:
		return "", fmt.Errorf("invalid --scope %q — valid values: project, global", s)
	}
}

type genSkillFlags struct {
	scope  InstallScope
	noAI   bool
	dryRun bool
	agent  Agent
	name   string
}

func runGenSkill(cmd *cobra.Command, rootCmd *cobra.Command, cfg *config, flags genSkillFlags) error {
	out := cmd.OutOrStdout()

	localCfg := *cfg
	localCfg.skillName = flags.name // always set; defaults to sanitized app name
	localCfg.agent = flags.agent

	fmt.Fprintf(out, "Collecting help from %s command tree...\n", rootCmd.Name())

	var result GenerateResult
	if flags.noAI {
		fmt.Fprintln(out, "AI generation disabled — using help-text fallback.")
		root := CollectHelp(rootCmd)
		result = GenerateResult{
			Content: generateFallback(&localCfg, rootCmd, root),
			Method:  "fallback",
		}
	} else {
		fmt.Fprintf(out, "Generating skill with %s...\n", localCfg.agent)
		result = Generate(&localCfg, rootCmd)
		if strings.HasPrefix(result.Method, "fallback") {
			fmt.Fprintln(out, "AI generation unavailable — using help-text fallback.")
		} else {
			fmt.Fprintf(out, "AI generation succeeded.\n")
		}
	}

	if flags.dryRun {
		fmt.Fprintf(out, "\n%s\n", strings.Repeat("-", 60))
		fmt.Fprintln(out, result.Content)
		fmt.Fprintln(out, strings.Repeat("-", 60))
		fmt.Fprintf(out, "\n(dry-run: no files written, method: %s)\n", result.Method)
		return nil
	}

	clients := clientTargetsFromMethod(result.Method)
	installTargets := BuildTargets(localCfg.skillName, flags.scope, clients)

	if err := Install(installTargets, result.Content); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Fprintf(out, "\nSkill %q installed (%s, method: %s)\n\n", localCfg.skillName, flags.scope, result.Method)
	for _, t := range installTargets {
		abs, _ := filepath.Abs(filepath.Join(t.SkillDir, "SKILL.md"))
		fmt.Fprintf(out, "  → %s\n", abs)
	}
	return nil
}

// clientTargetsFromMethod maps the generation result to install targets.
// "ai:claude" → [claude, agents], "ai:codex" → [codex, agents], "fallback" → [agents].
func clientTargetsFromMethod(method string) []ClientTarget {
	if after, ok := strings.CutPrefix(method, "ai:"); ok {
		return DefaultTargets(Agent(after))
	}
	return DefaultTargets("")
}
