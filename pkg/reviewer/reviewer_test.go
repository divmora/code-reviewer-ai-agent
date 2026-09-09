package reviewer

import (
	"testing"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

func TestFilterDiffFiles(t *testing.T) {
	files := []*git.FileDiff{
		{NewPath: "pkg/auth/login.go", RawDiff: "@@ -1 +1 @@\n+func Login()"},
		{NewPath: "package-lock.json", RawDiff: "some lock diff"},
		{NewPath: "assets/logo.png", RawDiff: "binary diff"},
		{NewPath: "proto/service.pb.go", RawDiff: "generated diff"},
		{NewPath: "unchanged.go"}, // Zero net change
	}

	filtered, binaryWarnings := FilterDiffFiles(files, nil)

	if len(filtered) != 1 || filtered[0].NewPath != "pkg/auth/login.go" {
		t.Errorf("expected only login.go to pass filter, got %v", filtered)
	}

	if len(binaryWarnings) != 1 || binaryWarnings[0] != "assets/logo.png" {
		t.Errorf("expected logo.png in binary warnings, got %v", binaryWarnings)
	}
}

func TestScanForSecrets(t *testing.T) {
	diff := &git.FileDiff{
		NewPath: "pkg/config/keys.go",
		Hunks: []git.DiffHunk{
			{
				NewStart: 1,
				Lines: []string{
					"+package config",
					`+const apiKey = "sk-1234567890abcdef1234567890abcdef"`,
				},
			},
		},
	}

	issues := ScanForSecrets([]*git.FileDiff{diff})
	if len(issues) != 1 {
		t.Fatalf("expected 1 secret issue, got %d", len(issues))
	}
	if issues[0].Severity != model.SeverityHigh {
		t.Errorf("expected high severity for secret, got %s", issues[0].Severity)
	}
}

func TestApplyCommentBudget(t *testing.T) {
	var issues []model.ReviewIssue
	for i := 0; i < 20; i++ {
		sev := model.SeverityLow
		if i == 0 {
			sev = model.SeverityHigh
		}
		issues = append(issues, model.ReviewIssue{
			Severity:    sev,
			Filepath:    "file.go",
			Line:        i + 1,
			Description: "Issue",
		})
	}

	top, overflow := ApplyCommentBudget(issues, 5)
	if len(top) != 5 {
		t.Errorf("expected top 5 issues, got %d", len(top))
	}
	if len(overflow) != 15 {
		t.Errorf("expected 15 overflow issues, got %d", len(overflow))
	}
	if top[0].Severity != model.SeverityHigh {
		t.Errorf("expected high severity issue to be first in top budget")
	}
}
