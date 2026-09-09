package reviewer

import (
	"regexp"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

var secretPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"AWS Access Key ID", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"AWS Secret Key", regexp.MustCompile(`(?i)\baws_secret_access_key\s*=\s*['"][a-zA-Z0-9/+=]{40}['"]`)},
	{"Generic API Key", regexp.MustCompile(`(?i)\b(api_key|apikey|secret_key|auth_token)\s*=\s*['"][a-zA-Z0-9_\-]{20,}['"]`)},
	{"OpenAI API Key", regexp.MustCompile(`\bsk-[a-zA-Z0-9]{32,}\b`)},
	{"GitHub Token", regexp.MustCompile(`\bgh[pousr]_[a-zA-Z0-9]{36,}\b`)},
	{"GitLab Token", regexp.MustCompile(`\bglpat-[a-zA-Z0-9_\-]{20,}\b`)},
	{"Private Key Header", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
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

					for _, sec := range secretPatterns {
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
