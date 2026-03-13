package cobragenskill

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// agentDef describes how to invoke an AI agent in headless (non-interactive) mode.
type agentDef struct {
	bin       string
	buildArgs func(prompt string) []string
}

var knownAgents = map[Agent]agentDef{
	AgentClaude: {
		bin: "claude",
		buildArgs: func(prompt string) []string {
			return []string{"--print", prompt}
		},
	},
	AgentCodex: {
		bin: "codex",
		buildArgs: func(prompt string) []string {
			return []string{"--no-interactive", prompt}
		},
	},
	AgentGemini: {
		bin: "gemini",
		buildArgs: func(prompt string) []string {
			return []string{prompt}
		},
	},
}

// InvokeAgent calls the agent binary with the given prompt and returns its output.
func InvokeAgent(agent Agent, customBin string, prompt string) (string, error) {
	def, ok := knownAgents[agent]
	if !ok {
		return "", fmt.Errorf("unknown agent %q — valid values: claude, codex, gemini", agent)
	}

	bin := def.bin
	if customBin != "" {
		bin = customBin
	}

	if _, err := exec.LookPath(bin); err != nil {
		return "", fmt.Errorf("agent binary %q not found in PATH", bin)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, def.buildArgs(prompt)...)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("agent %q timed out after 2 minutes", bin)
		}
		return "", fmt.Errorf("agent %q exited with error: %w", bin, err)
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", fmt.Errorf("agent %q returned empty output", bin)
	}
	return result, nil
}
