package reviewer

import (
	"path/filepath"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
)

var defaultNoiseFiles = []string{
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"composer.lock",
	"go.sum",
	"Gemfile.lock",
	"Cargo.lock",
	"poetry.lock",
}

var binaryExtensions = []string{
	".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".webp",
	".pdf", ".zip", ".tar", ".gz", ".7z", ".rar",
	".mp4", ".mov", ".avi", ".webm",
	".exe", ".dll", ".so", ".dylib", ".bin",
}

// FilterDiffFiles removes lockfiles, minified files, and generated code from the diff list.
func FilterDiffFiles(files []*git.FileDiff, customIgnores []string) ([]*git.FileDiff, []string) {
	var filtered []*git.FileDiff
	var binaryWarnings []string

	for _, file := range files {
		path := file.NewPath
		if path == "" {
			path = file.OldPath
		}
		base := filepath.Base(path)
		ext := strings.ToLower(filepath.Ext(path))

		// Check binary files
		isBinary := false
		for _, binExt := range binaryExtensions {
			if ext == binExt {
				isBinary = true
				break
			}
		}
		if isBinary {
			binaryWarnings = append(binaryWarnings, path)
			continue
		}

		// Check lockfiles
		isLockfile := false
		for _, lock := range defaultNoiseFiles {
			if base == lock {
				isLockfile = true
				break
			}
		}
		if isLockfile {
			continue
		}

		// Check minified or protobuf generated files
		if strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.css") ||
			strings.HasSuffix(base, ".pb.go") || strings.HasSuffix(base, "_gen.go") {
			continue
		}

		// Check custom ignores
		if len(customIgnores) > 0 {
			ignored := false
			for _, pat := range customIgnores {
				if matched, _ := filepath.Match(pat, base); matched {
					ignored = true
					break
				}
			}
			if ignored {
				continue
			}
		}

		// Discard files with zero net changes (unless it is an explicit file deletion or raw diff is present)
		if len(file.ChangedLines) == 0 && !file.IsDeleted && !file.IsNew && len(file.Hunks) == 0 && file.RawDiff == "" {
			continue
		}

		filtered = append(filtered, file)
	}

	return filtered, binaryWarnings
}
