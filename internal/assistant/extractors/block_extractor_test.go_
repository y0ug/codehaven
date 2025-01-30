package responseextractor

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/y0ug/ai-helper/internal/filemanager"
)

const BlockFence = "```"

func TestFindEditBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []EditResult
	}{
		{
			name: "basic edit block",
			input: `test.txt
<<<<<<< SEARCH
Hello world
=======
Hello Go
>>>>>>> REPLACE`,
			expected: []EditResult{
				{
					Edit: &Edit{
						Filename: "test.txt",
						Original: "Hello world",
						Updated:  "Hello Go",
					},
				},
			},
		},
		{
			name:  "shell command block",
			input: "```bash\necho 'Hello'\n```\n",
			expected: []EditResult{
				{
					Shell: &ShellCommand{Command: "echo 'Hello'"},
				},
			},
		},
		{
			name: "multiple blocks",
			input: `test.txt
<<<<<<< SEARCH
Old content
=======
New content
>>>>>>> REPLACE
` + BlockFence + `sh
ls -la
` + BlockFence + ``,
			expected: []EditResult{
				{
					Edit: &Edit{
						Filename: "test.txt",
						Original: "Old content",
						Updated:  "New content",
					},
				},
				{
					Shell: &ShellCommand{Command: "ls -la"},
				},
			},
		},
	}

	coder := &BlockExtractor{
		fence: [2]string{"```", "```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := coder.GetEdits(tt.input)
			if len(results) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(results))
			}

			for i, res := range results {
				exp := tt.expected[i]
				if exp.Edit != nil {
					if res.Edit == nil {
						t.Fatal("expected edit block, got nil")
					}
					if res.Edit.Filename != exp.Edit.Filename {
						t.Errorf(
							"expected filename %q, got %q",
							exp.Edit.Filename,
							res.Edit.Filename,
						)
					}
					if res.Edit.Original != exp.Edit.Original {
						t.Errorf(
							"expected original %q, got %q",
							exp.Edit.Original,
							res.Edit.Original,
						)
					}
					if res.Edit.Updated != exp.Edit.Updated {
						t.Errorf("expected updated %q, got %q", exp.Edit.Updated, res.Edit.Updated)
					}
				}
				if exp.Shell != nil {
					if res.Shell == nil {
						t.Fatal("expected shell command, got nil")
					}
					if res.Shell.Command != exp.Shell.Command {
						t.Errorf(
							"expected command %q, got %q",
							exp.Shell.Command,
							res.Shell.Command,
						)
					}
				}
			}
		})
	}
}

func TestExtractEditBlock(t *testing.T) {
	coder := &BlockExtractor{fence: [2]string{"```", "```"}}

	t.Run("valid block", func(t *testing.T) {
		lines := []string{
			"test.txt",
			"<<<<<<< SEARCH",
			"line 1",
			"line 2",
			"=======",
			"new line 1",
			"new line 2",
			">>>>>>> REPLACE",
		}

		fname, orig, updated, _, err := coder.extractEditBlock(lines, 1)
		if err != nil {
			t.Fatal(err)
		}

		if fname != "test.txt" {
			t.Errorf("expected filename 'test.txt', got %q", fname)
		}
		if orig != "line 1\nline 2" {
			t.Errorf("unexpected original content: %q", orig)
		}
		if updated != "new line 1\nnew line 2" {
			t.Errorf("unexpected updated content: %q", updated)
		}
	})

	t.Run("missing divider", func(t *testing.T) {
		lines := []string{
			"<<<<<<< SEARCH",
			"content",
		}
		_, _, _, _, err := coder.extractEditBlock(lines, 0)
		if err == nil {
			t.Fatal("expected error for missing divider")
		}
	})
}

func TestFindFilename(t *testing.T) {
	coder := &BlockExtractor{fence: [2]string{"```", "```"}}

	tests := []struct {
		name     string
		lines    []string
		expected string
	}{
		{
			name: "filename above block",
			lines: []string{
				"test.txt",
				"```",
				"<<<<<<< SEARCH",
			},
			expected: "test.txt",
		},
		{
			name: "no filename",
			lines: []string{
				"<<<<<<< SEARCH",
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fname := coder.findFilename(tt.lines, len(tt.lines)-1)
			if fname != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, fname)
			}
		})
	}
}

func TestDoReplace(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		content := "Hello world\nSecond line"
		original := "Hello world"
		updated := "Hello Go"

		result := ApplyEdit(content, original, updated)
		expected := "Hello Go\nSecond line"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("flexible whitespace", func(t *testing.T) {
		content := "  Hello world  \nSecond line"
		original := "Hello world"
		updated := "Hello Go"

		result := ApplyEdit(content, original, updated)
		expected := "  Hello Go  \nSecond line"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestApplyEdits(t *testing.T) {
	tempDir := t.TempDir()
	coder := &BlockExtractor{
		fence:  [2]string{"```", "```"},
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	t.Run("successful edit", func(t *testing.T) {
		fm := filemanager.NewLocalFileManager()
		originalData := []byte("hello\noriginal content\nbye")
		testFile := filepath.Join(tempDir, "success.txt")
		if err := os.WriteFile(testFile, originalData, 0644); err != nil {
			t.Fatal(err)
		}
		fm.Add(testFile, false) // Editable

		edit := Edit{
			Filename: testFile,
			Original: "original content",
			Updated:  "new content",
		}

		err := coder.ApplyEdits(fm, []Edit{edit}, false)
		if err != nil {
			t.Fatal(err)
		}

		fm.Commit("test")
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		expected := "hello\nnew content\nbye"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}

		contentS, _, err := fm.Get(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if contentS != expected {
			t.Errorf("expected %q, got %q", expected, contentS)
		}
	})

	t.Run("dryrun does not modify file", func(t *testing.T) {
		fm := filemanager.NewLocalFileManager()
		originalData := []byte("hello\ndryrun content\nbye")
		testFile := filepath.Join(tempDir, "dryrun.txt")
		if err := os.WriteFile(testFile, originalData, 0644); err != nil {
			t.Fatal(err)
		}
		fm.Add(testFile, false) // Editable

		edit := Edit{
			Filename: testFile,
			Original: "dryrun content",
			Updated:  "modified content",
		}

		err := coder.ApplyEdits(fm, []Edit{edit}, true) // Dryrun
		if err != nil {
			t.Fatal(err)
		}

		fm.Commit("dryrun test")
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != string(originalData) {
			t.Errorf(
				"dryrun should not modify file; expected %q, got %q",
				originalData,
				string(content),
			)
		}

		contentS, _, err := fm.Get(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if contentS != string(originalData) {
			t.Errorf("file manager content unchanged; expected %q, got %q", originalData, contentS)
		}
	})

	t.Run("read-only file not modified", func(t *testing.T) {
		fm := filemanager.NewLocalFileManager()
		originalData := []byte("read-only content")
		testFile := filepath.Join(tempDir, "readonly.txt")
		if err := os.WriteFile(testFile, originalData, 0644); err != nil {
			t.Fatal(err)
		}
		fm.Add(testFile, true) // Read-only

		edit := Edit{
			Filename: testFile,
			Original: "read-only content",
			Updated:  "new content",
		}

		err := coder.ApplyEdits(fm, []Edit{edit}, false)
		if err != nil {
			t.Fatal(err)
		}

		fm.Commit("read-only test")
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != string(originalData) {
			t.Errorf(
				"read-only file should not be modified; expected %q, got %q",
				originalData,
				string(content),
			)
		}

		contentS, _, err := fm.Get(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if contentS != string(originalData) {
			t.Errorf("file manager content unchanged; expected %q, got %q", originalData, contentS)
		}
	})

	t.Run("file not in manager not modified", func(t *testing.T) {
		fm := filemanager.NewLocalFileManager()
		originalData := []byte("not in manager content")
		testFile := filepath.Join(tempDir, "notinmanager.txt")
		if err := os.WriteFile(testFile, originalData, 0644); err != nil {
			t.Fatal(err)
		}
		// Do not add to file manager

		edit := Edit{
			Filename: testFile,
			Original: "not in manager content",
			Updated:  "new content",
		}

		err := coder.ApplyEdits(fm, []Edit{edit}, false)
		if err != nil {
			t.Fatal(err)
		}

		// No commit needed as file isn't managed
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != string(originalData) {
			t.Errorf(
				"file not in manager should not be modified; expected %q, got %q",
				originalData,
				string(content),
			)
		}

		// Verify file is not in manager
		_, _, err = fm.Get(testFile)
		if err == nil {
			t.Error("expected error when getting file not in manager, got nil")
		}
	})
}
