package repomanager

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestWalkDirectory(t *testing.T) {
	// Setup test directory structure
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name            string
		opts            WalkOptions
		expectedFiles   []string
		expectedIgnored []string
	}{
		{
			name: "Basic walk with gitignore",
			opts: WalkOptions{
				RootPath:    tmpDir,
				IgnoreFiles: []string{".gitignore"},
				IncludeDirs: false,
			},
			expectedFiles: []string{
				".customignore",
				"pkg/util.go",
				"main.go",
				"README.md",
				"temp.tmp",
			},
			expectedIgnored: []string{
				"build",
				"node_modules",
				"test.log",
			},
		},
		{
			name: "Walk with directories included",
			opts: WalkOptions{
				RootPath:    tmpDir,
				IgnoreFiles: []string{".gitignore"},
				IncludeDirs: true,
			},
			expectedFiles: []string{
				"pkg/",
				"pkg/util.go",
				"main.go",
				"README.md",
				".customignore",
				"temp.tmp",
			},
			expectedIgnored: []string{
				"build",
				"node_modules",
				"test.log",
			},
		},
		{
			name: "Walk with custom ignore file",
			opts: WalkOptions{
				RootPath:    tmpDir,
				IgnoreFiles: []string{".customignore"},
				IncludeDirs: false,
			},
			expectedFiles: []string{
				"pkg/util.go",
				"main.go",
				"README.md",
				"node_modules/package.json",
				"build/output.txt",
				".gitignore",
				"test.log",
			},
			expectedIgnored: []string{
				"temp.tmp",
			},
		},
		{
			name: "Walk with two ignore files",
			opts: WalkOptions{
				RootPath:    tmpDir,
				IgnoreFiles: []string{".gitignore", ".customignore"},
				IncludeDirs: false,
			},
			expectedFiles: []string{
				"pkg/util.go",
				"main.go",
				"README.md",
			},
			expectedIgnored: []string{
				"temp.tmp",
				"build",
				"node_modules",
				"test.log",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WalkDirectory(tt.opts)

			// Sort results for comparison
			sort.Strings(result.Files)
			sort.Strings(result.Ignored)
			sort.Strings(tt.expectedFiles)
			sort.Strings(tt.expectedIgnored)

			// Check files
			if len(result.Files) != len(tt.expectedFiles) {
				t.Errorf("got %d files, want %d\ngot: %v\nwant: %v",
					len(result.Files), len(tt.expectedFiles),
					result.Files, tt.expectedFiles)
			}

			for i := range result.Files {
				if i >= len(tt.expectedFiles) {
					break
				}
				if result.Files[i] != tt.expectedFiles[i] {
					t.Errorf("file[%d] = %s, want %s", i, result.Files[i], tt.expectedFiles[i])
				}
			}

			// Check ignored files
			if len(result.Ignored) != len(tt.expectedIgnored) {
				t.Errorf("got %d ignored files, want %d\ngot: %v\nwant: %v",
					len(result.Ignored), len(tt.expectedIgnored),
					result.Ignored, tt.expectedIgnored)
			}

			for i := range result.Ignored {
				if i >= len(tt.expectedIgnored) {
					break
				}
				if result.Ignored[i] != tt.expectedIgnored[i] {
					t.Errorf(
						"ignored[%d] = %s, want %s",
						i,
						result.Ignored[i],
						tt.expectedIgnored[i],
					)
				}
			}

			// Check for errors
			if len(result.Errors) > 0 {
				t.Errorf("unexpected errors: %v", result.Errors)
			}
		})
	}
}

// Helper function to setup test directory structure
func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "repomanager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test directory structure
	files := map[string]string{
		".gitignore": `node_modules/
build/
*.log
`,
		".customignore": `*.tmp
temp/
`,
		"main.go":                   "package main\n\nfunc main() {}\n",
		"README.md":                 "# Test Repository\n",
		"pkg/util.go":               "package pkg\n\nfunc Util() {}\n",
		"node_modules/package.json": "{}",
		"build/output.txt":          "build output",
		"test.log":                  "log content",
		"temp.tmp":                  "temporary file",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	return tmpDir
}
