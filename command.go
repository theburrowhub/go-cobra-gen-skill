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
		flagFor     string
		flagName    string
	)

	cmd := &cobra.Command{
		Use:   "gen-skill",
		Short: "Generate an Agent Skill for use with Claude, Cursor, Codex, and more",
		Long: `Generate an Agent Skill (SKILL.md) compatible with https://agentskills.io.

The skill teaches AI agents how to use this CLI tool: available commands,
flags, common workflows, and tips.

Two independent choices:

  --agent   Which AI to USE for generating the skill content.
            (auto tries claude → codex → gemini; none uses help-text fallback)

  --for     Which agent clients to INSTALL the skill for.
            Comma-separated: claude, codex, gemini, cursor, agents, all
            Default: the agent that did the generation + .agents/skills/
            "agents" means .agents/skills/ only (cross-client convention)

Install locations per scope (global / project):
  claude  ~/.claude/skills/<n>/  or  .claude/skills/<n>/
  codex   ~/.codex/skills/<n>/   or  .codex/skills/<n>/
  gemini  ~/.gemini/skills/<n>/  or  .gemini/skills/<n>/
  cursor  ~/.cursor/skills/<n>/  or  .cursor/skills/<n>/
  agents  ~/.agents/skills/<n>/  or  .agents/skills/<n>/  ← always included`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse --for
			var parsedTargets []ClientTarget
			if flagFor != "" {
				var err error
				parsedTargets, err = ParseTargets(flagFor)
				if err != nil {
					return err
				}
			}
			return runGenSkill(cmd, rootCmd, cfg, genSkillFlags{
				global:  flagGlobal,
				project: flagProject,
				noAI:    flagNoAI,
				dryRun:  flagDryRun,
				agent:   Agent(flagAgent),
				forTargets: parsedTargets,
				name:    sanitizeName(flagName),
			})
		},
	}

	f := cmd.Flags()
	f.BoolVar(&flagGlobal, "global", false,
		"Install skill globally (under ~/.<client>/skills/)")
	f.BoolVar(&flagProject, "project", false,
		"Install skill in the current project (under ./.<client>/skills/)")
	f.BoolVar(&flagNoAI, "no-ai", false,
		"Skip AI generation; use help-text fallback only")
	f.BoolVar(&flagDryRun, "dry-run", false,
		"Print the generated SKILL.md to stdout without writing any files")
	f.StringVar(&flagAgent, "agent", string(AgentAuto),
		"AI agent to USE for generation: auto | claude | codex | gemini | none")
	f.StringVar(&flagFor, "for", "",
		"Agent clients to INSTALL for (comma-separated): claude, codex, gemini, cursor, agents, all\n"+
			"\t(default: agent used for generation + agents)")
	f.StringVar(&flagName, "name", cfg.skillName,
		"Override the skill name (default: app name, sanitized)")

	return cmd
}

type genSkillFlags struct {
	global     bool
	project    bool
	noAI       bool
	dryRun     bool
	agent      Agent
	forTargets []ClientTarget // nil means derive from generation result
	name       string
}

func runGenSkill(cmd *cobra.Command, rootCmd *cobra.Command, cfg *config, flags genSkillFlags) error {
	out := cmd.OutOrStdout()

	// Apply flag overrides to a local copy.
	localCfg := *cfg
	if flags.name != "" {
		localCfg.skillName = flags.name
	}
	if flags.noAI {
		localCfg.agent = AgentNone
	} else if flags.agent != "" && flags.agent != AgentAuto {
		localCfg.agent = flags.agent
	}
	if len(flags.forTargets) > 0 {
		localCfg.targets = flags.forTargets
	}

	fmt.Fprintf(out, "Collecting help from %s command tree...\n", rootCmd.Name())

	// Generate skill content.
	var result GenerateResult
	if localCfg.agent == AgentNone {
		fmt.Fprintln(out, "AI generation disabled — using help-text fallback.")
		root := CollectHelp(rootCmd)
		result = GenerateResult{
			Content: generateFallback(&localCfg, rootCmd, root),
			Method:  "fallback",
		}
	} else {
		label := string(localCfg.agent)
		if localCfg.agent == AgentAuto {
			label = "auto (claude → codex → gemini)"
		}
		fmt.Fprintf(out, "Generating skill with AI agent (%s)...\n", label)
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

	// Resolve which clients to install for.
	clients := resolveInstallTargets(localCfg.targets, result)

	// Determine install scope.
	scope, err := resolveScope(out, os.Stdin, flags)
	if err != nil {
		return err
	}

	targets := BuildTargets(localCfg.skillName, scope, clients)

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

// resolveInstallTargets returns the final deduplicated list of ClientTargets.
// If the user specified targets explicitly (via --for or WithTargets), use those.
// Otherwise default to the agent that performed generation + always .agents/skills/.
func resolveInstallTargets(cfgTargets []ClientTarget, result GenerateResult) []ClientTarget {
	if len(cfgTargets) > 0 {
		return ResolveTargets(cfgTargets)
	}
	// Derive from the generation method ("ai:claude" → claude, "ai:codex" → codex, etc.)
	var genAgent Agent
	if after, ok := strings.CutPrefix(result.Method, "ai:"); ok {
		genAgent = Agent(after)
	}
	return DefaultTargets(genAgent)
}

// resolveScope determines ScopeGlobal or ScopeProject from flags or interactive prompt.
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

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Where would you like to install the skill?")
	fmt.Fprintln(out, "  [1] Global  — ~/.<client>/skills/  (available in all projects)")
	fmt.Fprintln(out, "  [2] Project — ./.<client>/skills/  (this project only)")
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
