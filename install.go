package cobragenskill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallScope indicates whether a skill is installed globally or per-project.
type InstallScope string

const (
	ScopeGlobal  InstallScope = "global"
	ScopeProject InstallScope = "project"
)

// ClientTarget identifies an agent client for skill installation.
type ClientTarget string

const (
	TargetClaude ClientTarget = "claude"
	TargetCodex  ClientTarget = "codex"
	TargetGemini ClientTarget = "gemini"
	// TargetAgents is the cross-client convention (.agents/skills/) scanned by all
	// compliant agents. It is always included in every install.
	TargetAgents ClientTarget = "agents"
)

// nativeDirName maps each ClientTarget to the dot-directory it uses.
var nativeDirName = map[ClientTarget]string{
	TargetClaude: ".claude",
	TargetCodex:  ".codex",
	TargetGemini: ".gemini",
	TargetAgents: ".agents",
}

// agentClientTarget maps a generation Agent to its corresponding ClientTarget.
var agentClientTarget = map[Agent]ClientTarget{
	AgentClaude: TargetClaude,
	AgentCodex:  TargetCodex,
	AgentGemini: TargetGemini,
}

// DefaultTargets returns the install targets for the agent that performed generation:
// the agent's own native directory + always .agents/skills/.
// For AgentNone (fallback) only .agents/skills/ is returned.
func DefaultTargets(genAgent Agent) []ClientTarget {
	if native, ok := agentClientTarget[genAgent]; ok {
		return []ClientTarget{native, TargetAgents}
	}
	return []ClientTarget{TargetAgents}
}

// InstallTarget describes a single skill directory to write to.
type InstallTarget struct {
	Scope    InstallScope
	Client   ClientTarget
	SkillDir string
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
