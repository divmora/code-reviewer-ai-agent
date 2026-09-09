package ast

import (
	"regexp"
	"strings"
)

var (
	pythonDefRegex = regexp.MustCompile(`^(?:async\s+)?def\s+([a-zA-Z0-9_]+)\s*\(|^class\s+([a-zA-Z0-9_]+)`)
	jsFuncRegex    = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?(?:function\s*([a-zA-Z0-9_]*)|class\s+([a-zA-Z0-9_]+)|const\s+([a-zA-Z0-9_]+)\s*=\s*(?:async\s*)?\([^)]*\)\s*=>)`)
	phpFuncRegex   = regexp.MustCompile(`^(?:(?:public|protected|private|static|final|abstract)\s+)*(?:function\s+([a-zA-Z0-9_]+)|class\s+([a-zA-Z0-9_]+)|trait\s+([a-zA-Z0-9_]+)|interface\s+([a-zA-Z0-9_]+))`)
	javaFuncRegex  = regexp.MustCompile(`^(?:(?:public|protected|private|static|final|abstract|synchronized)\s+)*[\w\<\>\[\]]+\s+([a-zA-Z0-9_]+)\s*\([^)]*\)\s*\{|^(?:(?:public|protected|private|abstract)\s+)*(?:class|interface|enum)\s+([a-zA-Z0-9_]+)`)
	rustFuncRegex  = regexp.MustCompile(`^(?:pub(?:\([^)]+\))?\s+)?(?:async\s+)?(?:fn\s+([a-zA-Z0-9_]+)|struct\s+([a-zA-Z0-9_]+)|impl\b|trait\s+([a-zA-Z0-9_]+))`)
)

// GenericParser extracts enclosing function/class scopes for various languages.
type GenericParser struct {
	Language string
}

// GetRelevantScopes parses line structures and returns enclosing block ranges for changed lines.
func (p *GenericParser) GetRelevantScopes(code string, changedLines []int) []ScopeRange {
	lines := strings.Split(code, "\n")
	if len(lines) == 0 {
		return nil
	}

	changedSet := make(map[int]bool)
	for _, l := range changedLines {
		changedSet[l] = true
	}

	if p.Language == "python" {
		return p.getPythonScopes(lines, changedSet)
	}

	return p.getBraceScopes(lines, changedSet)
}

func (p *GenericParser) getPythonScopes(lines []string, changedSet map[int]bool) []ScopeRange {
	var scopes []ScopeRange

	type block struct {
		startLine int
		indent    int
	}

	var blocks []block

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		if pythonDefRegex.MatchString(trimmed) {
			blocks = append(blocks, block{startLine: lineNum, indent: indent})
		}
	}

	for _, b := range blocks {
		// Find end of block by indentation
		endLine := len(lines)
		for j := b.startLine; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			ind := len(lines[j]) - len(strings.TrimLeft(lines[j], " \t"))
			if ind <= b.indent {
				endLine = j
				break
			}
		}

		hasChange := false
		for l := b.startLine; l <= endLine; l++ {
			if changedSet[l] {
				hasChange = true
				break
			}
		}

		if hasChange {
			scopes = append(scopes, ScopeRange{StartLine: b.startLine, EndLine: endLine})
		}
	}

	return scopes
}

func (p *GenericParser) getBraceScopes(lines []string, changedSet map[int]bool) []ScopeRange {
	var scopes []ScopeRange

	var matchRegex *regexp.Regexp
	switch p.Language {
	case "javascript", "typescript":
		matchRegex = jsFuncRegex
	case "php":
		matchRegex = phpFuncRegex
	case "java":
		matchRegex = javaFuncRegex
	case "rust":
		matchRegex = rustFuncRegex
	default:
		matchRegex = phpFuncRegex
	}

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		if matchRegex.MatchString(trimmed) {
			// Find closing brace
			braceCount := 0
			foundOpen := false
			endLine := len(lines)

			for j := i; j < len(lines); j++ {
				l := lines[j]
				for _, ch := range l {
					if ch == '{' {
						braceCount++
						foundOpen = true
					} else if ch == '}' {
						braceCount--
					}
				}

				if foundOpen && braceCount <= 0 {
					endLine = j + 1
					break
				}
			}

			hasChange := false
			for l := lineNum; l <= endLine; l++ {
				if changedSet[l] {
					hasChange = true
					break
				}
			}

			if hasChange {
				scopes = append(scopes, ScopeRange{StartLine: lineNum, EndLine: endLine})
			}
		}
	}

	return scopes
}
