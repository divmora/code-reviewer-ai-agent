package model

// Severity represents the criticality level of an issue.
type Severity string

const (
	SeverityHigh   Severity = "high"   // Bugs, security vulnerabilities, severe crashes
	SeverityMedium Severity = "medium" // Performance issues, missing error checks, N+1 queries
	SeverityLow    Severity = "low"    // Code style, naming, readability, refactoring
	SeverityInfo   Severity = "info"   // Best practice suggestions, documentation
)

// Category represents the domain/type of issue found.
type Category string

const (
	CategoryBug          Category = "bug"
	CategorySecurity     Category = "security"
	CategoryPerformance  Category = "performance"
	CategoryStyle        Category = "style"
	CategoryBestPractice Category = "best-practice"
)

// ReviewIssue represents an individual finding on a specific file and line.
type ReviewIssue struct {
	Severity    Severity `json:"severity"`
	Category    Category `json:"category"`
	Filepath    string   `json:"filepath"`
	Line        int      `json:"line"`
	EndLine     int      `json:"end_line,omitempty"`
	Description string   `json:"description"`
	Suggestion  string   `json:"suggestion,omitempty"` // Replacement code block
}

// ReviewReport represents the complete structured output from the code review agent.
type ReviewReport struct {
	Summary              string        `json:"summary"`
	SuggestedTitle       string        `json:"suggested_title"`
	MRDescription        string        `json:"mr_description"`
	FilesAnalyzed        []string      `json:"files_analyzed"`
	CodeScore            int           `json:"code_score"`            // 0-100 Quality Score
	SecurityScore        int           `json:"security_score"`        // 0-100 Security Score
	MaintainabilityScore int           `json:"maintainability_score"` // 0-100 Maintainability Score
	Poem                 string        `json:"poem"`                  // 4-line witty poem
	PositiveFeedback     []string      `json:"positive_feedback"`
	Issues               []ReviewIssue `json:"issues"`
	HeadSHA              string        `json:"head_sha"`
	Verdict              string        `json:"verdict"` // APPROVE, COMMENT, REQUEST_CHANGES
}

// CalculateVerdict determines the review verdict based on scores and issue severity.
func (r *ReviewReport) CalculateVerdict() string {
	if r.Verdict != "" {
		return r.Verdict
	}
	hasCritical := false
	for _, issue := range r.Issues {
		if issue.Severity == SeverityHigh {
			hasCritical = true
			break
		}
	}

	if hasCritical || r.SecurityScore < 60 || r.CodeScore < 60 {
		r.Verdict = "REQUEST_CHANGES"
	} else if r.CodeScore >= 80 && r.SecurityScore >= 80 && r.MaintainabilityScore >= 80 {
		r.Verdict = "APPROVE"
	} else {
		r.Verdict = "COMMENT"
	}
	return r.Verdict
}
