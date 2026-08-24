package git

// IsLineInDiff checks if a line number exists in the list of changed lines.
func IsLineInDiff(line int, changedLines []int) bool {
	for _, l := range changedLines {
		if l == line {
			return true
		}
	}
	return false
}

// FindNearestDiffLine finds the closest modified line to a target line.
func FindNearestDiffLine(targetLine int, changedLines []int) int {
	if len(changedLines) == 0 {
		return targetLine
	}
	best := changedLines[0]
	minDist := abs(targetLine - best)

	for _, l := range changedLines[1:] {
		dist := abs(targetLine - l)
		if dist < minDist {
			minDist = dist
			best = l
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
