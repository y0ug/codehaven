package extractors

import (
	"math"
	"strings"

	"github.com/y0ug/llmhaven/chat"
)

func GetTools(
	extractors ...Extractor,
) (tools []chat.Tool) {
	for _, e := range extractors {
		tools = append(tools, e.GetChatTools()...)
	}

	return
}

func isShellBlockStart(line string) bool {
	shellPrefixes := []string{"```bash", "```sh", "```shell", "```cmd", "```batch", "```zsh"}
	for _, prefix := range shellPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func extractShellCommand(lines []string, i int) (string, int) {
	var cmdLines []string
	i++ // Skip opening fence
	for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
		cmdLines = append(cmdLines, lines[i])
		i++
	}
	i++ // Skip closing fence
	return strings.Join(cmdLines, "\n"), i
}

func ApplyEdit(content, original, updated string) string {
	original = stripQuotedWrapping(original)
	updated = stripQuotedWrapping(updated)

	// Handle new file creation
	if original == "" && content == "" {
		return updated
	}

	// TODO: should have a switch to append vs overwrite
	if original == "" {
		return updated
	}

	// First try exact match
	if idx := strings.Index(content, original); idx != -1 {
		return content[:idx] + updated + content[idx+len(original):]
	}

	// Try flexible whitespace matching
	return replaceWithFlexibleWhitespace(content, original, updated)
}

func stripQuotedWrapping(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Remove fences
	if strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// Purpose: Find and replace a block of text while being lenient about:
// * Leading/trailing whitespace differences
// * Blank lines around the block
// * Exact indentation matching
// How it works:
// * Compares lines by their content rather than exact whitespace
// * Scores potential matches based on whitespace similarity
// * Replaces the best matching block while preserving original indentation
func replaceWithFlexibleWhitespace(content, original, updated string) string {
	contentLines := splitLines(content)
	origLines := splitLines(original)
	updatedLines := splitLines(updated)

	// Find best matching block with flexible whitespace
	bestScore := math.MaxFloat64
	bestIndex := -1

	for i := 0; i < len(contentLines)-len(origLines); i++ {
		end := i + len(origLines)
		if end > len(contentLines) {
			continue
		}
		score := compareBlocks(contentLines[i:end], origLines)
		if score < bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	if bestIndex == -1 {
		return content
	}

	// Preserve original indentation for each line
	var adjustedUpdated []string
	for i := 0; i < len(updatedLines); i++ {
		if i < len(origLines) && (bestIndex+i) < len(contentLines) {
			// Get original line's leading whitespace
			leadingWS := getLeadingWhitespace(contentLines[bestIndex+i])
			adjustedLine := leadingWS + strings.TrimSpace(updatedLines[i])
			adjustedUpdated = append(adjustedUpdated, adjustedLine)
		} else {
			adjustedUpdated = append(adjustedUpdated, updatedLines[i])
		}
	}
	var result []string
	result = append(result, contentLines[:bestIndex]...)
	result = append(result, adjustedUpdated...)
	result = append(result, contentLines[bestIndex+len(origLines):]...)
	return strings.Join(result, "\n")
}

func getLeadingWhitespace(line string) string {
	for i, char := range line {
		if char != ' ' && char != '\t' {
			return line[:i]
		}
	}
	return line
}

func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func compareBlocks(a, b []string) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	var total float64
	for i := range a {
		total += lineDistance(a[i], b[i])
	}
	return total
}

func lineDistance(a, b string) float64 {
	aClean := strings.TrimSpace(a)
	bClean := strings.TrimSpace(b)
	if aClean == bClean {
		return 0.1 * float64(len(a)-len(aClean))
	}
	return 1.0
}

func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix of size (len(s1)+1) x (len(s2)+1)
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(values ...int) int {
	minValue := values[0]
	for _, v := range values[1:] {
		if v < minValue {
			minValue = v
		}
	}
	return minValue
}

func normalizedDistance(s1, s2 string) float64 {
	distance := levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))
	if maxLen == 0 {
		return 0.0
	}
	return float64(distance) / float64(maxLen)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
