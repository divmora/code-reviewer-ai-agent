package provider

import (
	"strings"
	"testing"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

func TestParseTargetURL(t *testing.T) {
	// GitLab Self-Hosted
	glTarget, err := ParseTargetURL("https://gitlab.corp.internal/backend/auth-service/-/merge_requests/42")
	if err != nil {
		t.Fatalf("failed to parse GitLab URL: %v", err)
	}
	if glTarget.Provider != "gitlab" || glTarget.BaseURL != "https://gitlab.corp.internal" || glTarget.PRID != 42 || glTarget.ProjectPath != "backend/auth-service" {
		t.Errorf("unexpected GitLab target: %+v", glTarget)
	}

	// GitHub
	ghTarget, err := ParseTargetURL("https://github.com/divmora/localharness/pull/123")
	if err != nil {
		t.Fatalf("failed to parse GitHub URL: %v", err)
	}
	if ghTarget.Provider != "github" || ghTarget.Owner != "divmora" || ghTarget.Repo != "localharness" || ghTarget.PRID != 123 {
		t.Errorf("unexpected GitHub target: %+v", ghTarget)
	}

	// Bitbucket
	bbTarget, err := ParseTargetURL("https://bitbucket.org/myorg/billing-service/pull-requests/9")
	if err != nil {
		t.Fatalf("failed to parse Bitbucket URL: %v", err)
	}
	if bbTarget.Provider != "bitbucket" || bbTarget.Owner != "myorg" || bbTarget.Repo != "billing-service" || bbTarget.PRID != 9 {
		t.Errorf("unexpected Bitbucket target: %+v", bbTarget)
	}
}

func TestDedupWatermark(t *testing.T) {
	desc := "Initial user description\n\n---\n### 🤖 AI Code Review\n<!-- review_sha: a1b2c3d4e5f6 -->"

	extracted := ExtractReviewSHAFromDescription(desc)
	if extracted != "a1b2c3d4e5f6" {
		t.Errorf("expected extracted SHA 'a1b2c3d4e5f6', got %q", extracted)
	}

	shouldSkip, reason := ShouldSkipReview("a1b2c3d4e5f6", desc, false)
	if !shouldSkip {
		t.Errorf("expected review to be skipped when SHA matches")
	}
	if !strings.Contains(reason, "already reviewed") {
		t.Errorf("unexpected skip reason: %s", reason)
	}

	// Force override
	shouldSkipForced, _ := ShouldSkipReview("a1b2c3d4e5f6", desc, true)
	if shouldSkipForced {
		t.Errorf("expected force review NOT to be skipped")
	}

	// New commit SHA
	shouldSkipNew, _ := ShouldSkipReview("ffffffffffff", desc, false)
	if shouldSkipNew {
		t.Errorf("expected new commit SHA NOT to be skipped")
	}
}

func TestComposeMRDescription(t *testing.T) {
	report := &model.ReviewReport{
		Summary:              "Added JWT rotation",
		CodeScore:            88,
		SecurityScore:        95,
		MaintainabilityScore: 90,
		Poem:                 "Code flows clean,\nTests all pass,\nSecurity keen,\nTop of the class.",
	}

	composed := ComposeMRDescription("Original MR text", report, "abcdef123456")

	if !strings.Contains(composed, "Original MR text") {
		t.Errorf("expected original description to be preserved")
	}
	if !strings.Contains(composed, "88/100") || !strings.Contains(composed, "95/100") {
		t.Errorf("expected scorecard table with scores in description")
	}
	if !strings.Contains(composed, "<!-- review_sha: abcdef123456 -->") {
		t.Errorf("expected review_sha watermark in description")
	}
}

func TestBuildUpdatedLabels(t *testing.T) {
	currentLabels := []string{"team::backend", "feature", "Quality Score::50", "Security Score::40"}
	report := &model.ReviewReport{
		CodeScore:            90,
		SecurityScore:        95,
		MaintainabilityScore: 85,
	}

	updated := BuildUpdatedLabels(currentLabels, report)

	// Verify old score labels removed and new ones added
	hasOldQuality := false
	hasNewQuality := false
	hasBackend := false
	hasApproved := false

	for _, l := range updated {
		if l == "Quality Score::50" {
			hasOldQuality = true
		}
		if l == "Quality Score::90" {
			hasNewQuality = true
		}
		if l == "team::backend" {
			hasBackend = true
		}
		if l == "review::approved" {
			hasApproved = true
		}
	}

	if hasOldQuality {
		t.Errorf("old Quality Score::50 should have been removed")
	}
	if !hasNewQuality {
		t.Errorf("new Quality Score::90 should be present")
	}
	if !hasBackend {
		t.Errorf("original label 'team::backend' should be preserved")
	}
	if !hasApproved {
		t.Errorf("expected review::approved verdict label")
	}
}
