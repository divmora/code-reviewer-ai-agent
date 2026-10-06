package rules

import (
	"os"
	"testing"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

func TestLoadRulesWithInlineJSON(t *testing.T) {
	jsonPayload := `[
		{
			"id": "custom-rule-1",
			"name": "Check Auth Token",
			"severity": "high",
			"category": "security",
			"description": "Ensure Authorization header is checked."
		}
	]`

	cfg, err := LoadRules(IngestionOptions{
		RulesJSON:    jsonPayload,
		CustomPrompt: "Custom prompt for review",
		Profile:      "assertive",
	})

	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	if cfg.CustomPrompt != "Custom prompt for review" {
		t.Errorf("unexpected custom prompt: %s", cfg.CustomPrompt)
	}

	if cfg.Reviews.Profile != model.ProfileAssertive {
		t.Errorf("expected assertive profile, got %s", cfg.Reviews.Profile)
	}

	foundCustom := false
	for _, r := range cfg.Reviews.Rules {
		if r.ID == "custom-rule-1" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Errorf("custom-rule-1 not found in loaded rules")
	}
}

func TestFilterApplicableRules(t *testing.T) {
	rules := []model.CustomRule{
		{
			ID:        "go-rule",
			Name:      "Go Rule",
			Languages: []string{"go"},
		},
		{
			ID:        "py-rule",
			Name:      "Python Rule",
			Languages: []string{"python"},
		},
		{
			ID:   "universal-rule",
			Name: "Universal Rule",
		},
	}

	// For a Go file change
	matched := FilterApplicableRules(rules, []string{"pkg/auth/login.go"})

	hasGo := false
	hasPy := false
	hasUniversal := false

	for _, r := range matched {
		if r.ID == "go-rule" {
			hasGo = true
		}
		if r.ID == "py-rule" {
			hasPy = true
		}
		if r.ID == "universal-rule" {
			hasUniversal = true
		}
	}

	if !hasGo || !hasUniversal {
		t.Errorf("expected Go and Universal rules to match, got %v", matched)
	}
	if hasPy {
		t.Errorf("Python rule should not match Go file diff")
	}
}

func TestIsPathIgnored(t *testing.T) {
	ignorePatterns := []string{"**/vendor/**", "**/*.lock", "**/go.sum"}

	if !IsPathIgnored("vendor/github.com/foo/bar.go", ignorePatterns) {
		t.Errorf("expected vendor path to be ignored")
	}
	if !IsPathIgnored("go.sum", ignorePatterns) {
		t.Errorf("expected go.sum to be ignored")
	}
	if IsPathIgnored("pkg/auth/login.go", ignorePatterns) {
		t.Errorf("expected main code file not to be ignored")
	}
}

func TestLoadRulesCustomPromptEnv(t *testing.T) {
	// 1. Test CODE_REVIEWER_CUSTOM_PROMPT
	t.Setenv("CODE_REVIEWER_CUSTOM_PROMPT", "Prompt from CODE_REVIEWER_CUSTOM_PROMPT")
	t.Setenv("CUSTOM_PROMPT", "Prompt from CUSTOM_PROMPT")

	cfg, err := LoadRules(IngestionOptions{})
	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}
	if cfg.CustomPrompt != "Prompt from CODE_REVIEWER_CUSTOM_PROMPT" {
		t.Errorf("expected CODE_REVIEWER_CUSTOM_PROMPT to take priority, got %q", cfg.CustomPrompt)
	}

	// 2. Test fallback to CUSTOM_PROMPT
	t.Setenv("CODE_REVIEWER_CUSTOM_PROMPT", "")
	cfg2, err := LoadRules(IngestionOptions{})
	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}
	if cfg2.CustomPrompt != "Prompt from CUSTOM_PROMPT" {
		t.Errorf("expected fallback to CUSTOM_PROMPT, got %q", cfg2.CustomPrompt)
	}
}

func TestLoadRulesCustomFileResolution(t *testing.T) {
	tempDir := t.TempDir()
	customFilePath := tempDir + "/custom-rules.yaml"
	content := `
custom_prompt: "Rule from custom file"
reviews:
  profile: "chill"
`
	if err := os.WriteFile(customFilePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test rules file: %v", err)
	}

	// 1. Load via explicit filename relative to workspace
	cfg, err := LoadRules(IngestionOptions{
		Workspace: tempDir,
		RulesFile: "custom-rules.yaml",
	})
	if err != nil {
		t.Fatalf("LoadRules with relative custom file failed: %v", err)
	}
	if cfg.CustomPrompt != "Rule from custom file" {
		t.Errorf("expected prompt from custom rules file, got %q", cfg.CustomPrompt)
	}
	if cfg.Reviews.Profile != model.ProfileChill {
		t.Errorf("expected chill profile from custom rules file, got %q", cfg.Reviews.Profile)
	}

	// 2. Load via CODE_REVIEWER_RULES_FILE env var
	t.Setenv("CODE_REVIEWER_RULES_FILE", "custom-rules.yaml")
	cfgEnv, err := LoadRules(IngestionOptions{
		Workspace: tempDir,
	})
	if err != nil {
		t.Fatalf("LoadRules with CODE_REVIEWER_RULES_FILE failed: %v", err)
	}
	if cfgEnv.CustomPrompt != "Rule from custom file" {
		t.Errorf("expected prompt via CODE_REVIEWER_RULES_FILE, got %q", cfgEnv.CustomPrompt)
	}
}

func TestAutoDiscoverCodeReviewerYaml(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/.code-reviewer.yaml"
	content := `
custom_prompt: "Rule from .code-reviewer.yaml"
reviews:
  profile: "assertive"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	cfg, err := LoadRules(IngestionOptions{
		Workspace: tempDir,
	})
	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}
	if cfg.CustomPrompt != "Rule from .code-reviewer.yaml" {
		t.Errorf("expected prompt from .code-reviewer.yaml, got %q", cfg.CustomPrompt)
	}
	if cfg.Reviews.Profile != model.ProfileAssertive {
		t.Errorf("expected assertive profile, got %q", cfg.Reviews.Profile)
	}
}
