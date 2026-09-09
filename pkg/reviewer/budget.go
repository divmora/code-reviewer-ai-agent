package reviewer

import (
	"sort"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// ApplyCommentBudget filters and caps inline issues to the top maxComments by severity.
func ApplyCommentBudget(issues []model.ReviewIssue, maxComments int) ([]model.ReviewIssue, []model.ReviewIssue) {
	if maxComments <= 0 {
		maxComments = 15
	}

	if len(issues) <= maxComments {
		return issues, nil
	}

	// Sort issues: High > Medium > Low > Info
	severityRank := map[model.Severity]int{
		model.SeverityHigh:   4,
		model.SeverityMedium: 3,
		model.SeverityLow:    2,
		model.SeverityInfo:   1,
	}

	sorted := make([]model.ReviewIssue, len(issues))
	copy(sorted, issues)

	sort.SliceStable(sorted, func(i, j int) bool {
		return severityRank[sorted[i].Severity] > severityRank[sorted[j].Severity]
	})

	topIssues := sorted[:maxComments]
	overflowIssues := sorted[maxComments:]

	return topIssues, overflowIssues
}
