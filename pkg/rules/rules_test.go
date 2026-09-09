package rules

import (
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
		CustomPrompt: "Custom prompt from Zenith",
		Profile:      "assertive",
	})

	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	if cfg.CustomPrompt != "Custom prompt from Zenith" {
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
