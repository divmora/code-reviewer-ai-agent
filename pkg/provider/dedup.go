package provider

import (
	"fmt"
	"regexp"
	"strings"
)

var shaWatermarkRegex = regexp.MustCompile(`<!--\s*review_sha:\s*([a-f0-9]+)\s*-->`)

// ExtractReviewSHAFromDescription parses the SHA watermark from an MR description.
func ExtractReviewSHAFromDescription(description string) string {
	matches := shaWatermarkRegex.FindStringSubmatch(description)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// ShouldSkipReview determines whether to skip review if commit was already analyzed.
func ShouldSkipReview(currentSHA, description string, force bool) (bool, string) {
	if force {
		return false, ""
	}

	if currentSHA == "" {
		return false, ""
	}

	lastReviewedSHA := ExtractReviewSHAFromDescription(description)
	if lastReviewedSHA != "" {
		// Compare full or 8-char prefixes
		if lastReviewedSHA == currentSHA || (len(lastReviewedSHA) >= 8 && len(currentSHA) >= 8 && lastReviewedSHA[:8] == currentSHA[:8]) {
			return true, fmt.Sprintf("Commit %s was already reviewed (use --force to re-review)", currentSHA[:min(8, len(currentSHA))])
		}
	}

	return false, ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
