package git

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// DiffHunk represents a single diff chunk.
type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Header   string
	Lines    []string
}

// FileDiff represents changes for a single file.
type FileDiff struct {
	OldPath      string
	NewPath      string
	IsNew        bool
	IsDeleted    bool
	IsRenamed    bool
	Hunks        []DiffHunk
	ChangedLines []int  // 1-indexed line numbers modified in the new file
	RawDiff      string // Full unified diff for this file
}

// ParseUnifiedDiff parses a raw git unified diff string into structured FileDiffs.
func ParseUnifiedDiff(diffContent string) []*FileDiff {
	var files []*FileDiff
	var currentFile *FileDiff
	var currentHunk *DiffHunk
	var fileDiffLines []string

	flushCurrentFile := func() {
		if currentFile != nil {
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				currentHunk = nil
			}
			currentFile.RawDiff = strings.Join(fileDiffLines, "\n")
			files = append(files, currentFile)
			currentFile = nil
			fileDiffLines = nil
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(diffContent))
	currentNewLine := 0

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "diff --git ") {
			flushCurrentFile()
			currentFile = &FileDiff{}
			fileDiffLines = append(fileDiffLines, line)

			// Extract paths: diff --git a/foo/bar.go b/foo/bar.go
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentFile.OldPath = strings.TrimPrefix(parts[2], "a/")
				currentFile.NewPath = strings.TrimPrefix(parts[3], "b/")
			}
			continue
		}

		if currentFile == nil {
			// In case diff starts without diff --git
			currentFile = &FileDiff{}
		}
		fileDiffLines = append(fileDiffLines, line)

		if strings.HasPrefix(line, "new file mode") {
			currentFile.IsNew = true
		} else if strings.HasPrefix(line, "deleted file mode") {
			currentFile.IsDeleted = true
		} else if strings.HasPrefix(line, "similarity index") || strings.HasPrefix(line, "rename from") {
			currentFile.IsRenamed = true
		} else if strings.HasPrefix(line, "--- ") {
			path := strings.TrimPrefix(line, "--- ")
			path = strings.TrimPrefix(path, "a/")
			if path != "/dev/null" && currentFile.OldPath == "" {
				currentFile.OldPath = path
			}
		} else if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimPrefix(line, "+++ ")
			path = strings.TrimPrefix(path, "b/")
			if path != "/dev/null" {
				currentFile.NewPath = path
			}
		} else if matches := hunkHeaderRegex.FindStringSubmatch(line); len(matches) > 0 {
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			oldStart, _ := strconv.Atoi(matches[1])
			oldCount := 1
			if matches[2] != "" {
				oldCount, _ = strconv.Atoi(matches[2])
			}

			newStart, _ := strconv.Atoi(matches[3])
			newCount := 1
			if matches[4] != "" {
				newCount, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &DiffHunk{
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				Header:   line,
			}
			currentNewLine = newStart
		} else if currentHunk != nil {
			currentHunk.Lines = append(currentHunk.Lines, line)

			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				currentFile.ChangedLines = append(currentFile.ChangedLines, currentNewLine)
				currentNewLine++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				// Deletion does not advance new file line counter
			} else if !strings.HasPrefix(line, `\`) {
				// Context line
				currentNewLine++
			}
		}
	}

	flushCurrentFile()
	return files
}

// GetChangedLinesFromDiff parses a raw diff string and returns changed line numbers.
func GetChangedLinesFromDiff(diffContent string) []int {
	files := ParseUnifiedDiff(diffContent)
	if len(files) == 0 {
		return nil
	}
	return files[0].ChangedLines
}
