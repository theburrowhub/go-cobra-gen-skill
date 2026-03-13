package cobragenskill

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ── sanitizeName ──────────────────────────────────────────────────────────────

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"mytool", "mytool"},
		{"MyTool", "mytool"},
		{"my_tool", "my-tool"},
		{"my tool", "my-tool"},
		{"-mytool-", "mytool"},
		{"my--tool", "my-tool"},
		{"MY AWESOME TOOL!", "my-awesome-tool"},
		{"", "cli-tool"},
		{strings.Repeat("a", 70), strings.Repeat("a", 64)},
	}
	for _, tc := range tests {
		got := sanitizeName(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ── yamlQuote ─────────────────────────────────────────────────────────────────

func TestYAMLQuote(t *testing.T) {
	tests := []struct {
		input   string
		wantRaw bool // true = input returned unchanged (no quoting needed)
	}{
		{"simple", true},
		{"Use this skill when: user asks", false}, // contains colon
		{"has \"quotes\"", false},
		{" leading space", false},
		{"trailing space ", false},
		{"no-special-chars-here", true},
	}
	for _, tc := range tests {
		got := yamlQuote(tc.input)
		isRaw := got == tc.input
		if isRaw != tc.wantRaw {
			t.Errorf("yamlQuote(%q) = %q; wantRaw=%v", tc.input, got, tc.wantRaw)
		}
		// Quoted values must start and end with "
		if !isRaw {
			if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
				t.Errorf("yamlQuote(%q) = %q; expected double-quoted string", tc.input, got)
			}
		}
	}
}

// ── CollectHelp ───────────────────────────────────────────────────────────────

func buildTestTree() *cobra.Command {
	root := &cobra.Command{
		Use:   "testapp",
		Short: "A test application",
		Long:  "A longer description for testapp.",
	}
	deploy := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the application",
	}
	deploy.AddCommand(&cobra.Command{
		Use:   "prod",
		Short: "Deploy to production",
	})
	root.AddCommand(deploy)
	root.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show status",
	})
	hidden := &cobra.Command{
		Use:    "hidden",
		Short:  "Hidden command",
		Hidden: true,
	}
	root.AddCommand(hidden)
	return root
}

func TestCollectHelp_Structure(t *testing.T) {
	root := buildTestTree()
	node := CollectHelp(root)

	if node.Path != "" {
		t.Errorf("root path should be empty, got %q", node.Path)
	}
	if node.Short != "A test application" {
		t.Errorf("unexpected root Short: %q", node.Short)
	}
	if node.HelpText == "" {
		t.Error("root HelpText should not be empty")
	}

	// Should have deploy and status, but NOT hidden or help/gen-skill.
	names := make(map[string]bool)
	for _, c := range node.Children {
		names[c.Path] = true
	}
	if !names["deploy"] {
		t.Error("expected deploy in children")
	}
	if !names["status"] {
		t.Error("expected status in children")
	}
	if names["hidden"] {
		t.Error("hidden command should be excluded")
	}

	// deploy should have a prod child.
	var deployNode *CommandNode
	for _, c := range node.Children {
		if c.Path == "deploy" {
			deployNode = c
		}
	}
	if deployNode == nil {
		t.Fatal("deploy node not found")
	}
	if len(deployNode.Children) != 1 || deployNode.Children[0].Path != "deploy prod" {
		t.Errorf("deploy children: %v", deployNode.Children)
	}
}

func TestCollectHelp_DoesNotPollute(t *testing.T) {
	root := buildTestTree()
	var buf bytes.Buffer
	root.SetOut(&buf)

	CollectHelp(root)

	// After CollectHelp, the root's output writer must still be &buf (unchanged).
	// We verify by writing to root's output and checking buf is written to.
	root.Print("ping")
	if !strings.Contains(buf.String(), "ping") {
		t.Error("root OutOrStdout was not restored after CollectHelp")
	}
}

// ── generateFallback ──────────────────────────────────────────────────────────

func TestGenerateFallback_ValidFrontmatter(t *testing.T) {
	root := buildTestTree()
	cfg := defaultConfig(root)
	cfg.version = "2.0.0"
	cfg.license = "MIT"

	tree := CollectHelp(root)
	content := generateFallback(cfg, root, tree)

	if !strings.HasPrefix(content, "---\n") {
		t.Error("SKILL.md should start with ---")
	}
	if !strings.Contains(content, "name: testapp") {
		t.Errorf("missing name field in:\n%s", content)
	}
	if !strings.Contains(content, "description:") {
		t.Error("missing description field")
	}
	if !strings.Contains(content, "license: MIT") {
		t.Error("missing license field")
	}
	if !strings.Contains(content, "version:") {
		t.Error("missing version in metadata")
	}
	if !strings.Contains(content, "## When to use this skill") {
		t.Error("missing 'When to use this skill' section")
	}
	if !strings.Contains(content, "## Commands reference") {
		t.Error("missing 'Commands reference' section")
	}
}

func TestGenerateFallback_DescriptionTruncation(t *testing.T) {
	root := buildTestTree()
	cfg := defaultConfig(root)
	cfg.description = strings.Repeat("x", 2000)

	tree := CollectHelp(root)
	content := generateFallback(cfg, root, tree)

	// Description in YAML is quoted; the actual string value must be ≤ 1024 chars.
	idx := strings.Index(content, "description: ")
	if idx == -1 {
		t.Fatal("no description field found")
	}
	// Find the line.
	line := content[idx:]
	end := strings.Index(line, "\n")
	descLine := line[:end]
	// Strip the key prefix and any YAML quotes to measure the value length.
	val := strings.TrimPrefix(descLine, "description: ")
	val = strings.Trim(val, `"`)
	// Unescape \" back to " for length check.
	val = strings.ReplaceAll(val, `\"`, `"`)
	if len(val) > 1024 {
		t.Errorf("description value length %d exceeds 1024", len(val))
	}
}

// ── Install ───────────────────────────────────────────────────────────────────

func TestInstall_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	targets := []InstallTarget{
		{Scope: ScopeGlobal, Client: "claude", SkillDir: filepath.Join(dir, ".claude", "skills", "testapp")},
		{Scope: ScopeGlobal, Client: "agents", SkillDir: filepath.Join(dir, ".agents", "skills", "testapp")},
	}
	content := "---\nname: testapp\ndescription: test\n---\n\n# body\n"

	if err := Install(targets, content); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	for _, tgt := range targets {
		dest := filepath.Join(tgt.SkillDir, "SKILL.md")
		data, err := os.ReadFile(dest)
		if err != nil {
			t.Errorf("could not read %s: %v", dest, err)
			continue
		}
		if string(data) != content {
			t.Errorf("%s content mismatch", dest)
		}
	}
}

// ── RegisterCommand integration ───────────────────────────────────────────────

func TestRegisterCommand_AddsGenSkill(t *testing.T) {
	root := buildTestTree()
	RegisterCommand(root)

	var found bool
	for _, c := range root.Commands() {
		if c.Name() == "gen-skill" {
			found = true
		}
	}
	if !found {
		t.Error("gen-skill command was not added to root")
	}
}

func TestGenSkillCommand_DryRun(t *testing.T) {
	root := buildTestTree()
	RegisterCommand(root, WithAgent(AgentNone))

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)

	root.SetArgs([]string{"gen-skill", "--dry-run", "--no-ai"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "name: testapp") {
		t.Errorf("dry-run output missing SKILL.md content; got:\n%s", output)
	}
	if !strings.Contains(output, "dry-run") {
		t.Error("expected dry-run notice in output")
	}
}

func TestGenSkillCommand_ProjectInstall(t *testing.T) {
	// Change to a temp directory so .claude/skills/ is created there.
	orig, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	defer os.Chdir(orig)

	root := buildTestTree()
	RegisterCommand(root, WithAgent(AgentNone))

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)

	root.SetArgs([]string{"gen-skill", "--no-ai", "--project"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify .claude/skills/testapp/SKILL.md exists.
	dest := filepath.Join(dir, ".claude", "skills", "testapp", "SKILL.md")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected %s to exist: %v", dest, err)
	}

	// Verify .agents/skills/testapp/SKILL.md exists.
	dest2 := filepath.Join(dir, ".agents", "skills", "testapp", "SKILL.md")
	if _, err := os.Stat(dest2); err != nil {
		t.Errorf("expected %s to exist: %v", dest2, err)
	}
}

// ── resolveScope ──────────────────────────────────────────────────────────────

func TestResolveScope_Flags(t *testing.T) {
	tests := []struct {
		flags    genSkillFlags
		wantErr  bool
		wantScope InstallScope
	}{
		{genSkillFlags{global: true}, false, ScopeGlobal},
		{genSkillFlags{project: true}, false, ScopeProject},
		{genSkillFlags{global: true, project: true}, true, ""},
	}
	for _, tc := range tests {
		scope, err := resolveScope(io.Discard, strings.NewReader(""), tc.flags)
		if tc.wantErr {
			if err == nil {
				t.Errorf("flags %+v: expected error, got nil", tc.flags)
			}
		} else {
			if err != nil {
				t.Errorf("flags %+v: unexpected error: %v", tc.flags, err)
			}
			if scope != tc.wantScope {
				t.Errorf("flags %+v: scope = %q, want %q", tc.flags, scope, tc.wantScope)
			}
		}
	}
}

func TestResolveScope_Interactive(t *testing.T) {
	tests := []struct {
		input string
		want  InstallScope
	}{
		{"1\n", ScopeGlobal},
		{"2\n", ScopeProject},
		{"\n", ScopeGlobal}, // default
	}
	for _, tc := range tests {
		scope, err := resolveScope(io.Discard, strings.NewReader(tc.input), genSkillFlags{})
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tc.input, err)
		}
		if scope != tc.want {
			t.Errorf("input %q: scope = %q, want %q", tc.input, scope, tc.want)
		}
	}
}
