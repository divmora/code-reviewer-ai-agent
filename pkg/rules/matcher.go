package rules

import (
	"path/filepath"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/ast"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// FilterApplicableRules returns the subset of custom rules relevant to the given modified files.
func FilterApplicableRules(rules []model.CustomRule, modifiedFiles []string) []model.CustomRule {
	if len(rules) == 0 || len(modifiedFiles) == 0 {
		return rules
	}

	// Detect all active languages in the diff
	activeLangs := make(map[string]bool)
	for _, file := range modifiedFiles {
		lang := ast.DetectLanguage(file)
		if lang != "" {
			activeLangs[lang] = true
		}
	}

	var matched []model.CustomRule

	for _, rule := range rules {
		// If rule specifies no languages and no file patterns, it applies universally
		if len(rule.Languages) == 0 && len(rule.FilePatterns) == 0 {
			matched = append(matched, rule)
			continue
		}

		// Check language match
		langMatch := false
		for _, l := range rule.Languages {
			if activeLangs[strings.ToLower(l)] {
				langMatch = true
				break
			}
		}

		// Check pattern match
		patternMatch := false
		for _, pat := range rule.FilePatterns {
			for _, file := range modifiedFiles {
				if matchGlob(pat, file) {
					patternMatch = true
					break
				}
			}
			if patternMatch {
				break
			}
		}

		if langMatch || patternMatch {
			matched = append(matched, rule)
		}
	}

	return matched
}

// IsPathIgnored checks if a file matches any of the ignore patterns.
func IsPathIgnored(path string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		if matchGlob(pattern, path) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, target string) bool {
	pattern = strings.TrimPrefix(pattern, "./")
	target = strings.TrimPrefix(target, "./")

	// Match exact suffix or substring if wildcard
	if strings.HasPrefix(pattern, "**/") {
		suffix := strings.TrimPrefix(pattern, "**/")
		matched, _ := filepath.Match(suffix, filepath.Base(target))
		if matched || strings.Contains(target, strings.TrimSuffix(suffix, "/**")) {
			return true
		}
	}

	matched, _ := filepath.Match(pattern, target)
	if matched {
		return true
	}

	baseMatched, _ := filepath.Match(pattern, filepath.Base(target))
	return baseMatched
}
