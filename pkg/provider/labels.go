package provider

import (
	"fmt"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

var scorePrefixes = []string{
	"Quality Score::",
	"Security Score::",
	"Maintainability Score::",
	"review::",
}

// BuildUpdatedLabels takes current labels, strips old AI labels, and appends new score/verdict labels.
func BuildUpdatedLabels(currentLabels []string, report *model.ReviewReport) []string {
	var updated []string

	for _, l := range currentLabels {
		isScoreLabel := false
		for _, prefix := range scorePrefixes {
			if strings.HasPrefix(l, prefix) {
				isScoreLabel = true
				break
			}
		}
		if !isScoreLabel {
			updated = append(updated, l)
		}
	}

	// Append updated score labels
	updated = append(updated, fmt.Sprintf("Quality Score::%d", report.CodeScore))
	updated = append(updated, fmt.Sprintf("Security Score::%d", report.SecurityScore))
	updated = append(updated, fmt.Sprintf("Maintainability Score::%d", report.MaintainabilityScore))

	// Append verdict label
	switch report.CalculateVerdict() {
	case "APPROVE":
		updated = append(updated, "review::approved")
	case "REQUEST_CHANGES":
		updated = append(updated, "review::changes-requested")
	case "COMMENT":
		updated = append(updated, "review::comment")
	}

	return updated
}

// GetScoreColor returns color hex for a given 0-100 score.
func GetScoreColor(score int) string {
	if score >= 80 {
		return "#2da44e" // Green
	} else if score >= 60 {
		return "#d97706" // Yellow/Amber
	}
	return "#cf222e" // Red
}
