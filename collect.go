package cobragenskill

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// CommandNode holds the collected help information for a single command
// and its subtree.
type CommandNode struct {
	// Path is the space-separated chain of subcommands (empty for root).
	Path string
	// Short is the one-line description.
	Short string
	// Long is the extended description (may be empty).
	Long string
	// HelpText is the full output of running --help on this command.
	HelpText string
	// Children holds help for each non-hidden subcommand.
	Children []*CommandNode
}

// CollectHelp walks the cobra command tree rooted at root and returns a
// CommandNode tree with help text for every visible (non-hidden) command.
// The "gen-skill" command itself and "help" are excluded to avoid recursion.
func CollectHelp(root *cobra.Command) *CommandNode {
	return collectNode(root, "")
}

func collectNode(cmd *cobra.Command, path string) *CommandNode {
	node := &CommandNode{
		Path:  path,
		Short: cmd.Short,
		Long:  cmd.Long,
	}

	// Capture help output without polluting original writers.
	var buf bytes.Buffer
	origOut := cmd.OutOrStdout()
	origErr := cmd.ErrOrStderr()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	// Use the command's own help function to produce canonical output.
	if h := cmd.HelpFunc(); h != nil {
		h(cmd, []string{})
	}
	cmd.SetOut(origOut)
	cmd.SetErr(origErr)
	node.HelpText = strings.TrimSpace(buf.String())

	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "gen-skill" {
			continue
		}
		var subPath string
		if path == "" {
			subPath = sub.Name()
		} else {
			subPath = path + " " + sub.Name()
		}
		node.Children = append(node.Children, collectNode(sub, subPath))
	}
	return node
}

// FormatForPrompt returns a string representation of the command tree
// suitable for inclusion in an AI prompt.
func FormatForPrompt(appName string, root *CommandNode) string {
	var sb strings.Builder
	formatNode(&sb, appName, root, 0)
	return sb.String()
}

func formatNode(sb *strings.Builder, appName string, node *CommandNode, depth int) {
	sep := strings.Repeat("=", 60)
	if node.Path == "" {
		fmt.Fprintf(sb, "%s\n%s (root)\n%s\n\n", sep, appName, sep)
	} else {
		fmt.Fprintf(sb, "%s\n%s %s\n%s\n\n", sep, appName, node.Path, sep)
	}
	fmt.Fprintf(sb, "%s\n\n", node.HelpText)
	for _, child := range node.Children {
		formatNode(sb, appName, child, depth+1)
	}
}
