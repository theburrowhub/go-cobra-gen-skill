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
	// bin is the expected binary name on PATH.
	bin string
	// buildArgs returns the CLI arguments to pass the prompt to the agent.
	buildArgs func(prompt string) []string
}

// knownAgents maps each Agent constant to its invocation definition.
var knownAgents = map[Agent]agentDef{
	// Claude Code: `claude --print "<prompt>"` runs in non-interactive print mode.
	AgentClaude: {
		bin: "claude",
		buildArgs: func(prompt string) []string {
			return []string{"--print", prompt}
		},
	},
	// OpenAI Codex CLI: `codex --no-interactive "<prompt>"`
	AgentCodex: {
		bin: "codex",
		buildArgs: func(prompt string) []string {
			return []string{"--no-interactive", prompt}
		},
	},
	// Gemini CLI: `gemini "<prompt>"`
	AgentGemini: {
		bin: "gemini",
		buildArgs: func(prompt string) []string {
			return []string{prompt}
		},
	},
}

// autoOrder defines the priority when AgentAuto is selected.
var autoOrder = []Agent{AgentClaude, AgentCodex, AgentGemini}

// InvokeAgent calls the requested AI agent with prompt and returns its output.
// If agent is AgentAuto it tries each known agent in order until one succeeds.
// Returns an error if no agent produces output.
func InvokeAgent(agent Agent, customBin string, prompt string) (string, error) {
	if agent == AgentAuto {
		var lastErr error
		for _, a := range autoOrder {
			out, err := invokeOne(a, "", prompt)
			if err == nil {
				return out, nil
			}
			lastErr = err
		}
		return "", fmt.Errorf("no AI agent available (tried claude, codex, gemini); last error: %w", lastErr)
	}
	return invokeOne(agent, customBin, prompt)
}

func invokeOne(agent Agent, customBin string, prompt string) (string, error) {
	def, ok := knownAgents[agent]
	if !ok {
		return "", fmt.Errorf("unknown agent %q", agent)
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

	args := def.buildArgs(prompt)
	cmd := exec.CommandContext(ctx, bin, args...)

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
