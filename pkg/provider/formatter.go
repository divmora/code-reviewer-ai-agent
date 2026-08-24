package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// FormatTerminal renders a rich colorized report for terminal output.
func FormatTerminal(report *model.ReviewReport) string {
	var b strings.Builder

	b.WriteString("\n" + colorBold + colorCyan + "========================================================" + colorReset + "\n")
	b.WriteString(colorBold + " 🤖 CODE REVIEW REPORT" + colorReset + "\n")
	b.WriteString(colorBold + colorCyan + "========================================================" + colorReset + "\n\n")

	// Scores
	b.WriteString(colorBold + "📊 Quality Scorecard:" + colorReset + "\n")
	b.WriteString(fmt.Sprintf("   Quality Score:        %s%d/100%s\n", scoreColor(report.CodeScore), report.CodeScore, colorReset))
	b.WriteString(fmt.Sprintf("   Security Score:       %s%d/100%s\n", scoreColor(report.SecurityScore), report.SecurityScore, colorReset))
	b.WriteString(fmt.Sprintf("   Maintainability:      %s%d/100%s\n", scoreColor(report.MaintainabilityScore), report.MaintainabilityScore, colorReset))
	b.WriteString(fmt.Sprintf("   Verdict:              %s%s%s\n\n", verdictColor(report.CalculateVerdict()), report.Verdict, colorReset))

	// Suggested Title
	if report.SuggestedTitle != "" {
		b.WriteString(colorBold + "📌 Suggested Title: " + colorReset + report.SuggestedTitle + "\n\n")
	}

	// Summary
	if report.Summary != "" {
		b.WriteString(colorBold + "📝 Executive Summary:" + colorReset + "\n")
		b.WriteString("   " + report.Summary + "\n\n")
	}

	// Poem
	if report.Poem != "" {
		b.WriteString(colorBold + "✨ Poem:" + colorReset + "\n")
		for _, line := range strings.Split(report.Poem, "\n") {
			b.WriteString("   " + colorCyan + line + colorReset + "\n")
		}
		b.WriteString("\n")
	}

	// Issues
	b.WriteString(colorBold + fmt.Sprintf("🔍 Findings (%d issues):", len(report.Issues)) + colorReset + "\n")
	b.WriteString("--------------------------------------------------------\n")

	if len(report.Issues) == 0 {
		b.WriteString(colorGreen + "   ✅ No issues found! Clean changes." + colorReset + "\n")
	} else {
		for i, issue := range report.Issues {
			b.WriteString(fmt.Sprintf("\n%d. %s[%s] %s%s on %s%s:%d%s\n",
				i+1,
				severityColor(issue.Severity),
				strings.ToUpper(string(issue.Severity)),
				issue.Category,
				colorReset,
				colorBold,
				issue.Filepath,
				issue.Line,
				colorReset,
			))
			b.WriteString(fmt.Sprintf("   %s\n", issue.Description))

			if issue.Suggestion != "" {
				b.WriteString(colorGreen + "   💡 Suggestion:" + colorReset + "\n")
				for _, sl := range strings.Split(issue.Suggestion, "\n") {
					b.WriteString(fmt.Sprintf("      %s\n", sl))
				}
			}
		}
	}

	b.WriteString("\n" + colorBold + colorCyan + "========================================================" + colorReset + "\n")
	return b.String()
}

// FormatMarkdown renders a clean GitHub/GitLab flavored markdown report.
func FormatMarkdown(report *model.ReviewReport) string {
	var b strings.Builder

	b.WriteString("# 🤖 AI Code Review Report\n\n")

	// Scorecard table
	b.WriteString("### 📊 Quality Scorecard\n\n")
	b.WriteString("| Metric | Score | Status |\n")
	b.WriteString("| :--- | :---: | :--- |\n")
	b.WriteString(fmt.Sprintf("| Quality | **%d/100** | %s |\n", report.CodeScore, scoreEmoji(report.CodeScore)))
	b.WriteString(fmt.Sprintf("| Security | **%d/100** | %s |\n", report.SecurityScore, scoreEmoji(report.SecurityScore)))
	b.WriteString(fmt.Sprintf("| Maintainability | **%d/100** | %s |\n", report.MaintainabilityScore, scoreEmoji(report.MaintainabilityScore)))
	b.WriteString(fmt.Sprintf("| **Verdict** | **%s** | %s |\n\n", report.CalculateVerdict(), verdictEmoji(report.Verdict)))

	if report.Summary != "" {
		b.WriteString("### 📝 Executive Summary\n\n")
		b.WriteString(report.Summary + "\n\n")
	}

	if report.MRDescription != "" {
		b.WriteString("### 🔍 Walkthrough\n\n")
		b.WriteString(report.MRDescription + "\n\n")
	}

	if report.Poem != "" {
		b.WriteString("### 📜 Summary Poem\n\n")
		for _, line := range strings.Split(report.Poem, "\n") {
			b.WriteString(fmt.Sprintf("> *%s*  \n", strings.TrimSpace(line)))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("### 🔍 Detailed Findings (%d)\n\n", len(report.Issues)))
	for i, issue := range report.Issues {
		b.WriteString(fmt.Sprintf("#### %d. `[%s]` %s: `%s:%d`\n\n", i+1, strings.ToUpper(string(issue.Severity)), issue.Category, issue.Filepath, issue.Line))
		b.WriteString(issue.Description + "\n\n")
		if issue.Suggestion != "" {
			b.WriteString("```suggestion\n" + issue.Suggestion + "\n```\n\n")
		}
	}

	return b.String()
}

// FormatJSON outputs the report as indented JSON.
func FormatJSON(report *model.ReviewReport) (string, error) {
	bytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func scoreColor(score int) string {
	if score >= 80 {
		return colorGreen
	} else if score >= 60 {
		return colorYellow
	}
	return colorRed
}

func verdictColor(v string) string {
	switch v {
	case "APPROVE":
		return colorGreen
	case "REQUEST_CHANGES":
		return colorRed
	default:
		return colorYellow
	}
}

func severityColor(s model.Severity) string {
	switch s {
	case model.SeverityHigh:
		return colorRed
	case model.SeverityMedium:
		return colorYellow
	case model.SeverityLow:
		return colorBlue
	default:
		return colorCyan
	}
}

func scoreEmoji(score int) string {
	if score >= 80 {
		return "🟢 Excellent"
	} else if score >= 60 {
		return "🟡 Moderate"
	}
	return "🔴 Needs Improvement"
}

func verdictEmoji(v string) string {
	switch v {
	case "APPROVE":
		return "✅"
	case "REQUEST_CHANGES":
		return "❌"
	default:
		return "💬"
	}
}
