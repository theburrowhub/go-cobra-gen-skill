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

// InstallTarget describes a single skill directory to write to.
type InstallTarget struct {
	Scope     InstallScope
	Client    string // e.g. "claude", "agents"
	SkillDir  string // absolute or relative path to the skill's root directory
}

// GlobalTargets returns the standard install targets for global scope.
//
//   - ~/.claude/skills/<name>/   — Claude Code native location
//   - ~/.agents/skills/<name>/   — cross-client convention
func GlobalTargets(skillName string) []InstallTarget {
	home, _ := os.UserHomeDir()
	return []InstallTarget{
		{
			Scope:    ScopeGlobal,
			Client:   "claude",
			SkillDir: filepath.Join(home, ".claude", "skills", skillName),
		},
		{
			Scope:    ScopeGlobal,
			Client:   "agents",
			SkillDir: filepath.Join(home, ".agents", "skills", skillName),
		},
	}
}

// ProjectTargets returns the standard install targets for project scope.
//
//   - .claude/skills/<name>/   — Claude Code native location
//   - .agents/skills/<name>/   — cross-client convention
func ProjectTargets(skillName string) []InstallTarget {
	return []InstallTarget{
		{
			Scope:    ScopeProject,
			Client:   "claude",
			SkillDir: filepath.Join(".claude", "skills", skillName),
		},
		{
			Scope:    ScopeProject,
			Client:   "agents",
			SkillDir: filepath.Join(".agents", "skills", skillName),
		},
	}
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
