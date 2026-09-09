package ast

import (
	"fmt"
	"sort"
	"strings"
)

const maxFullFileLines = 150

// SliceFileContext generates an AST-scoped or full decorated source text for a file.
func SliceFileContext(code, filename string, changedLines []int) string {
	lines := strings.Split(code, "\n")
	if len(lines) == 0 {
		return ""
	}

	// For small files (< 150 lines), provide full decorated content
	if len(lines) < maxFullFileLines {
		var output strings.Builder
		for i, line := range lines {
			output.WriteString(fmt.Sprintf("%d: %s\n", i+1, line))
		}
		return strings.TrimRight(output.String(), "\n")
	}

	// For large files, extract AST scopes
	lang := DetectLanguage(filename)
	var parser LanguageParser

	if lang == "go" {
		parser = &GoParser{}
	} else if lang != "" {
		parser = &GenericParser{Language: lang}
	}

	var ranges []ScopeRange
	if parser != nil {
		ranges = parser.GetRelevantScopes(code, changedLines)
	}

	// Fallback to +/- 30 lines window around changed lines if no AST scopes found
	if len(ranges) == 0 {
		for _, lineNum := range changedLines {
			start := lineNum - 30
			if start < 1 {
				start = 1
			}
			end := lineNum + 30
			if end > len(lines) {
				end = len(lines)
			}
			ranges = append(ranges, ScopeRange{StartLine: start, EndLine: end})
		}
	}

	if len(ranges) == 0 {
		return fmt.Sprintf("# Full content skipped (large file: %d lines, no diff lines)", len(lines))
	}

	// Sort and merge overlapping or adjacent ranges
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].StartLine < ranges[j].StartLine
	})

	var merged []ScopeRange
	curr := ranges[0]

	for i := 1; i < len(ranges); i++ {
		next := ranges[i]
		if next.StartLine <= curr.EndLine+1 {
			if next.EndLine > curr.EndLine {
				curr.EndLine = next.EndLine
			}
		} else {
			merged = append(merged, curr)
			curr = next
		}
	}
	merged = append(merged, curr)

	// Construct decorated output with skipped comments
	var output strings.Builder
	lastLine := 0

	for _, r := range merged {
		start := r.StartLine
		if start < 1 {
			start = 1
		}
		end := r.EndLine
		if end > len(lines) {
			end = len(lines)
		}

		if start > lastLine+1 {
			output.WriteString(fmt.Sprintf("# ... skipped %d lines ...\n", start-lastLine-1))
		}

		for lineNum := start; lineNum <= end; lineNum++ {
			output.WriteString(fmt.Sprintf("%d: %s\n", lineNum, lines[lineNum-1]))
		}
		lastLine = end
	}

	if lastLine < len(lines) {
		output.WriteString(fmt.Sprintf("# ... skipped %d lines ...\n", len(lines)-lastLine))
	}

	return strings.TrimRight(output.String(), "\n")
}
