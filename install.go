package cobragenskill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallScope indicates whether a skill is installed globally or per-project.
type InstallScope string

const (
	ScopeGlobal  InstallScope = "global"
	ScopeProject InstallScope = "project"
)

// ClientTarget identifies an agent client for skill installation.
// It is independent from the Agent used to generate the skill.
type ClientTarget string

const (
	// TargetClaude installs into .<scope>/claude/skills/ (Claude Code).
	TargetClaude ClientTarget = "claude"
	// TargetCodex installs into .<scope>/codex/skills/ (OpenAI Codex).
	TargetCodex ClientTarget = "codex"
	// TargetGemini installs into .<scope>/gemini/skills/ (Gemini CLI).
	TargetGemini ClientTarget = "gemini"
	// TargetCursor installs into .<scope>/cursor/skills/ (Cursor).
	TargetCursor ClientTarget = "cursor"
	// TargetAgents installs into .agents/skills/ — the cross-client convention.
	// All compliant agents scan this path, so it ensures maximum compatibility.
	TargetAgents ClientTarget = "agents"
	// TargetAll expands to every known client target.
	TargetAll ClientTarget = "all"
)

// allKnownTargets is the expansion set for TargetAll (excluding TargetAll itself).
var allKnownTargets = []ClientTarget{
	TargetClaude, TargetCodex, TargetGemini, TargetCursor, TargetAgents,
}

// nativeDirName maps each ClientTarget to the dot-directory it uses.
var nativeDirName = map[ClientTarget]string{
	TargetClaude: ".claude",
	TargetCodex:  ".codex",
	TargetGemini: ".gemini",
	TargetCursor: ".cursor",
	TargetAgents: ".agents",
}

// agentClientTarget maps an Agent constant to the ClientTarget for its native directory.
var agentClientTarget = map[Agent]ClientTarget{
	AgentClaude: TargetClaude,
	AgentCodex:  TargetCodex,
	AgentGemini: TargetGemini,
}

// ParseTargets parses a comma-separated list of client target names (e.g. "claude,codex").
// TargetAll is accepted and expands at resolve time.
func ParseTargets(s string) ([]ClientTarget, error) {
	var result []ClientTarget
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		switch ClientTarget(part) {
		case TargetClaude, TargetCodex, TargetGemini, TargetCursor, TargetAgents, TargetAll:
			result = append(result, ClientTarget(part))
		default:
			return nil, fmt.Errorf("unknown target %q — valid values: claude, codex, gemini, cursor, agents, all", part)
		}
	}
	return result, nil
}

// ResolveTargets expands TargetAll, deduplicates, and ensures TargetAgents is always present
// (it is the universal cross-client path that all compliant agents scan).
func ResolveTargets(targets []ClientTarget) []ClientTarget {
	seen := make(map[ClientTarget]bool)
	var result []ClientTarget

	add := func(t ClientTarget) {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}

	for _, t := range targets {
		if t == TargetAll {
			for _, kt := range allKnownTargets {
				add(kt)
			}
		} else {
			add(t)
		}
	}

	// Always include the cross-client path.
	add(TargetAgents)
	return result
}

// DefaultTargets returns sensible targets when the user has not specified --for:
//   - The native directory of the agent that was actually used for generation (if known)
//   - Always .agents/skills/ for cross-client compatibility
func DefaultTargets(generationAgent Agent) []ClientTarget {
	var result []ClientTarget
	if native, ok := agentClientTarget[generationAgent]; ok {
		result = append(result, native)
	}
	return ResolveTargets(result) // ResolveTargets always appends TargetAgents
}

// InstallTarget describes a single skill directory to write to.
type InstallTarget struct {
	Scope    InstallScope
	Client   ClientTarget
	SkillDir string // absolute or relative path to the skill's root directory
}

// BuildTargets constructs the list of InstallTargets for the given scope and clients.
func BuildTargets(skillName string, scope InstallScope, clients []ClientTarget) []InstallTarget {
	home, _ := os.UserHomeDir()
	var targets []InstallTarget
	for _, c := range clients {
		dir, ok := nativeDirName[c]
		if !ok {
			continue
		}
		var base string
		switch scope {
		case ScopeGlobal:
			base = filepath.Join(home, dir, "skills", skillName)
		case ScopeProject:
			base = filepath.Join(dir, "skills", skillName)
		}
		targets = append(targets, InstallTarget{Scope: scope, Client: c, SkillDir: base})
	}
	return targets
}

// Install writes content as SKILL.md inside each target directory,
// creating intermediate directories as needed.
func Install(targets []InstallTarget, content string) error {
	for _, t := range targets {
		if err := os.MkdirAll(t.SkillDir, 0o755); err != nil {
			return fmt.Errorf("creating skill directory %s: %w", t.SkillDir, err)
		}
		dest := filepath.Join(t.SkillDir, "SKILL.md")
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	}
	return nil
}

// isOnPath returns true if the binary name is found in PATH.
func isOnPath(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
