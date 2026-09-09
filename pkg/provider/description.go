package provider

import (
	"fmt"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

const aiReviewHeader = "\n\n---\n### 🤖 AI Code Review"

// ComposeMRDescription appends or replaces the AI review block in an MR description.
func ComposeMRDescription(existingDescription string, report *model.ReviewReport, headSHA string) string {
	// Strip previous AI review block if present
	cleanDesc := existingDescription
	if idx := strings.Index(cleanDesc, aiReviewHeader); idx != -1 {
		cleanDesc = strings.TrimSpace(cleanDesc[:idx])
	}

	verdictEmoji := "✅"
	switch report.CalculateVerdict() {
	case "REQUEST_CHANGES":
		verdictEmoji = "🔴"
	case "COMMENT":
		verdictEmoji = "🟡"
	}

	var newSection strings.Builder
	newSection.WriteString(aiReviewHeader + "\n\n")

	// 1. Scorecard Table
	newSection.WriteString("#### 📊 Quality Scorecard\n")
	newSection.WriteString("| Quality Score | Security Score | Maintainability Score | Verdict |\n")
	newSection.WriteString("| :---: | :---: | :---: | :---: |\n")
	newSection.WriteString(fmt.Sprintf("| **%d/100** | **%d/100** | **%d/100** | **%s %s** |\n\n",
		report.CodeScore,
		report.SecurityScore,
		report.MaintainabilityScore,
		report.Verdict,
		verdictEmoji,
	))

	// 2. Summary
	if report.Summary != "" {
		newSection.WriteString("#### 📝 Summary of Changes\n")
		newSection.WriteString(report.Summary + "\n\n")
	}

	// 3. MR Description / Walkthrough if present
	if report.MRDescription != "" {
		newSection.WriteString(report.MRDescription + "\n\n")
	}

	// 4. Positive Feedback / Praise
	if len(report.PositiveFeedback) > 0 {
		newSection.WriteString("#### ✨ Highlights\n")
		for _, pf := range report.PositiveFeedback {
			newSection.WriteString(fmt.Sprintf("- %s\n", pf))
		}
		newSection.WriteString("\n")
	}

	// 5. Poem
	if report.Poem != "" {
		poemLines := strings.Split(report.Poem, "\n")
		var quotedPoem []string
		for _, pl := range poemLines {
			quotedPoem = append(quotedPoem, fmt.Sprintf("> *%s*", strings.TrimSpace(pl)))
		}
		newSection.WriteString(strings.Join(quotedPoem, "  \n") + "\n\n")
	}

	// 6. SHA Watermark
	if headSHA != "" {
		newSection.WriteString(fmt.Sprintf("<!-- review_sha: %s -->\n", headSHA))
	}

	if cleanDesc == "" {
		return strings.TrimSpace(newSection.String())
	}
	return cleanDesc + "\n" + newSection.String()
}
