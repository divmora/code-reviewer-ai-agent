package ast

import (
	"path/filepath"
	"strings"
)

// ScopeRange represents a 1-indexed line range [StartLine, EndLine].
type ScopeRange struct {
	StartLine int
	EndLine   int
}

// LanguageParser defines the interface for language-specific AST scope extractors.
type LanguageParser interface {
	GetRelevantScopes(code string, changedLines []int) []ScopeRange
}

// DetectLanguage returns the canonical language name based on file extension.
func DetectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	ext = strings.TrimPrefix(ext, ".")

	switch ext {
	case "go":
		return "go"
	case "py":
		return "python"
	case "js", "jsx", "mjs", "cjs":
		return "javascript"
	case "ts", "tsx", "mts", "cts":
		return "typescript"
	case "java":
		return "java"
	case "php":
		return "php"
	case "rs":
		return "rust"
	case "c", "h":
		return "c"
	case "cpp", "hpp", "cc", "cxx":
		return "cpp"
	case "rb":
		return "ruby"
	default:
		return ""
	}
}
