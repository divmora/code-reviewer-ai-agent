package reviewer

import (
	"regexp"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

var secretPatterns = []struct {
	name     string
	pattern  *regexp.Regexp
	keywords []string
}{
	{"AWS Access Key ID", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), []string{"akia"}},
	{"AWS Secret Key", regexp.MustCompile(`(?i)\baws_secret_access_key\s*=\s*['"][a-zA-Z0-9/+=]{40}['"]`), []string{"aws_secret_access_key"}},
	{"Generic API Key", regexp.MustCompile(`(?i)\b(api_key|apikey|secret_key|auth_token)\s*=\s*['"][a-zA-Z0-9_\-]{20,}['"]`), []string{"api_key", "apikey", "secret_key", "auth_token"}},
	{"OpenAI API Key", regexp.MustCompile(`\bsk-[a-zA-Z0-9]{32,}\b`), []string{"sk-"}},
	{"GitHub Token", regexp.MustCompile(`\bgh[pousr]_[a-zA-Z0-9]{36,}\b`), []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"}},
	{"GitLab Token", regexp.MustCompile(`\bglpat-[a-zA-Z0-9_\-]{20,}\b`), []string{"glpat-"}},
	{"Private Key Header", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`), []string{"-----begin"}},
}

// ScanForSecrets inspects added lines in diffs for hardcoded credentials.
func ScanForSecrets(files []*git.FileDiff) []model.ReviewIssue {
	var issues []model.ReviewIssue

	for _, file := range files {
		path := file.NewPath
		if path == "" {
			continue
		}

		for _, hunk := range file.Hunks {
			currentLine := hunk.NewStart

			for _, line := range hunk.Lines {
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					content := strings.TrimPrefix(line, "+")
					lowerContent := strings.ToLower(content)

					for _, sec := range secretPatterns {
						matchCandidate := false
						for _, kw := range sec.keywords {
							if strings.Contains(lowerContent, kw) {
								matchCandidate = true
								break
							}
						}
						if !matchCandidate {
							continue
						}

						if sec.pattern.MatchString(content) {
							issues = append(issues, model.ReviewIssue{
								Severity:    model.SeverityHigh,
								Category:    model.CategorySecurity,
								Filepath:    path,
								Line:        currentLine,
								Description: "Potential hardcoded " + sec.name + " detected! Never commit secrets or API keys into git. Use environment variables or secret managers.",
								Suggestion:  `os.Getenv("SECRET_NAME") // Read from environment or secret manager`,
							})
							break
						}
					}
					currentLine++
				} else if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, `\`) {
					currentLine++
				}
			}
		}
	}

	return issues
}
