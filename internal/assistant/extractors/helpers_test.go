package extractors

import (
	"testing"
)

func TestIsShellBlockStart(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"```bash", true},
		{"```sh", true},
		{"```shell", true},
		{"```python", false},
		{"plain text", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := isShellBlockStart(tt.line)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractShellCommand(t *testing.T) {
	lines := []string{
		"```bash",
		"echo 'Hello'",
		"date",
		"```",
	}

	cmd, newI := extractShellCommand(lines, 0)
	if cmd != "echo 'Hello'\ndate" {
		t.Errorf("unexpected command: %q", cmd)
	}
	if newI != len(lines) {
		t.Errorf("expected new index %d, got %d", len(lines), newI)
	}
}

func TestReplaceWithFlexibleWhitespace(t *testing.T) {
	content := "    Line 1\n  Line 2\nLine 3"
	original := "Line 1\nLine 2"
	updated := "New Line 1\nNew Line 2"

	result := replaceWithFlexibleWhitespace(content, original, updated)
	expected := "    New Line 1\n  New Line 2\nLine 3"

	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestLineDistance(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected float64
	}{
		{"  hello", "hello", 0.2},
		{"hello", "goodbye", 1.0},
		{"  hello  ", "  hello", 0.4},
	}

	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			result := lineDistance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}
