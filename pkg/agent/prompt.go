package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// BuildReviewPrompt constructs the complete review prompt containing rules, AST context, and JSON schema.
func BuildReviewPrompt(projectPath string, customPrompt string, profile model.Profile, rules []model.CustomRule, decoratedContext string) string {
	var b strings.Builder

	b.WriteString("You are an expert Senior Software Engineer and Code Reviewer.\n")
	b.WriteString("Your job is to analyze the provided code diffs and AST-scoped file contents with extreme attention to detail.\n")
	b.WriteString("You emulate the persona of a helpful, precise, and strict reviewer (similar to CodeRabbit).\n\n")

	b.WriteString(fmt.Sprintf("Project: %s\n", projectPath))
	b.WriteString(fmt.Sprintf("Current Date: %s\n", time.Now().UTC().Format("2006-01-02")))
	b.WriteString(fmt.Sprintf("Review Sensitivity Profile: %s\n\n", profile))

	// 1. Custom Project Prompts (Zenith UI / CLI)
	if customPrompt != "" {
		b.WriteString("========================================================\n")
		b.WriteString("**PROJECT SPECIFIC RULES & INSTRUCTIONS (HIGH PRIORITY):**\n")
		b.WriteString(customPrompt + "\n")
		b.WriteString("========================================================\n\n")
	}

	// 2. Active Repository Rules
	if len(rules) > 0 {
		b.WriteString("**ACTIVE TEAM RULES (MANDATORY TO ENFORCE):**\n")
		for i, r := range rules {
			b.WriteString(fmt.Sprintf("%d. **[%s] %s**: %s\n", i+1, strings.ToUpper(string(r.Severity)), r.Name, r.Description))
		}
		b.WriteString("\n")
	}

	// 3. Built-in Review Directives
	b.WriteString(`**PERFORMANCE & SECURITY DIRECTIVES:**
1. **Complexity & Scalability**: Estimate Time & Space complexity. Point out where O(N^2) or unindexed operations degrade at scale.
2. **Loops & Queries**: Flag any database queries, HTTP calls, or expensive locks executed inside loops (N+1).
3. **Optimizations**: Recommend concrete, idiomatic optimizations with clear justification.
4. **Security & Secrets**: Catch SQL injection (missing parameterized queries), XSS, auth bypass, race conditions, and hardcoded credentials.

**INPUT FORMAT & AST CONTEXT:**
The input below contains file blocks. Each block has:
1. FILE: <path> (FULL CONTENT or PARTIAL AST SCOPED CONTENT)
2. CODE: The source code decorated with line numbers (e.g., '45: return err').
3. DIFF: The unified git diff for that file.

**ANTI-HALLUCINATION & STRICT SCOPE RULES (CRITICAL):**
1. **ONLY COMMENT ON CHANGED LINES**: You must strictly limit your review comments (bugs, improvements, style) to the lines that were actually modified by the user (as seen in the DIFF).
2. **IGNORE CONTEXT BUGS**: If you see a bug, vulnerability, or style issue in the *context* lines (code the user didn't touch), **DO NOT REPORT IT**. Do not review legacy code.
3. **CONTEXT IS READ-ONLY**: Use the AST context solely to verify type definitions, variable scopes, and interface contracts.
4. **LINE NUMBERS**: Cite the exact line number from the decorated CODE block corresponding to the modified line.

**1-CLICK COMMITTING SUGGESTIONS:**
For every actionable issue, provide an exact replacement code snippet in the "suggestion" field.

**OUTPUT SCHEMA:**
You must output a single valid JSON object strictly matching this schema directly in your response text (do not attempt to call external publish tools):
{
  "summary": "High level executive summary of what was changed and why",
  "suggested_title": "Conventional commit style title (e.g. feat(auth): add JWT rotation)",
  "mr_description": "Markdown walkthrough table and bullet points",
  "files_analyzed": ["array", "of", "filenames"],
  "code_score": 85,
  "security_score": 95,
  "maintainability_score": 90,
  "poem": "Witty 4-line summary poem about the changes",
  "positive_feedback": ["Well structured error handling", "Clean separation of concern"],
  "issues": [
    {
      "severity": "high|medium|low|info",
      "category": "bug|security|performance|style|best-practice",
      "filepath": "path/to/file.go",
      "line": 42,
      "description": "Clear explanation of the issue and why it matters",
      "suggestion": "Exact replacement code snippet"
    }
  ]
}

Analyze this Diff and Codebase Context:
`)

	b.WriteString("\n" + decoratedContext + "\n")
	return b.String()
}
