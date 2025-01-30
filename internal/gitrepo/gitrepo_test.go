package gitrepo

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRepo(t *testing.T) *GitRepo {
	t.Helper()

	// Create temporary directory
	dir, err := os.MkdirTemp("", "gitrepo-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	// Initialize bare repository
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	// Create initial commit
	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create and add test file
	testfile := filepath.Join(dir, "README.md")
	err = os.WriteFile(testfile, []byte("# Test Repository\n"), 0644)
	require.NoError(t, err)

	_, err = wt.Add("README.md")
	require.NoError(t, err)

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create GitRepo instance
	gr, err := NewGitRepo(
		slog.Default(),
		[]string{dir},
		dir,
		WithAttributeOptions(true, true, false, false),
	)
	require.NoError(t, err)

	return gr
}

func TestCommitNewFile(t *testing.T) {
	gr := setupTestRepo(t)
	repo := gr.repo

	// Create new file
	newFile := filepath.Join(gr.root, "newfile.txt")
	err := os.WriteFile(newFile, []byte("test content\n"), 0644)
	require.NoError(t, err)

	// Commit using our wrapper
	hash, msg, err := gr.Commit(
		[]string{"newfile.txt"},
		"test context",
		"Add new file",
		true,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Equal(t, "Add new file", msg)

	// Verify commit in repository using full hash
	commit, err := repo.CommitObject(plumbing.NewHash(hash))
	require.NoError(t, err, "should find commit using full hash")
	assert.Equal(t, "Add new file", commit.Message)

	// Verify file exists in commit
	file, err := commit.File("newfile.txt")
	require.NoError(t, err)
	content, err := file.Contents()
	require.NoError(t, err)
	assert.Equal(t, "test content\n", content)
}

func TestGetDiffs(t *testing.T) {
	gr := setupTestRepo(t)

	// Modify existing file
	readmePath := filepath.Join(gr.root, "README.md")
	err := os.WriteFile(readmePath, []byte("# Modified Test Repository\n"), 0644)
	require.NoError(t, err)

	// Get expected diff from git CLI
	cmd := exec.Command("git", "diff", "--no-color", "HEAD", "--", "README.md")
	cmd.Dir = gr.root
	expectedDiffBytes, err := cmd.Output()
	require.NoError(t, err)
	expectedDiff := string(expectedDiffBytes) // normalizeDiff(string(expectedDiffBytes))

	// Get diffs from our implementation
	diffs, err := gr.GetDiffs([]string{"README.md"})
	require.NoError(t, err)
	actualDiff := diffs // normalizeDiff(diffs)

	// Compare the outputs
	assert.Equal(t, expectedDiff, actualDiff)
}

// normalizeDiff cleans up diff output for reliable comparison
func normalizeDiff(diff string) string {
	// Remove timestamps from diff headers
	re := regexp.MustCompile(`(\+{3}|-{3}) [ab]/.*\t.*\n`)
	cleaned := re.ReplaceAllString(diff, "$1 a/$2\n")

	// Trim whitespace and sort lines
	lines := strings.Split(strings.TrimSpace(cleaned), "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func OldTestGetDiffs(t *testing.T) {
	gr := setupTestRepo(t)

	// Modify existing file
	readmePath := filepath.Join(gr.root, "README.md")
	err := os.WriteFile(readmePath, []byte("# Modified Test Repository\n"), 0644)
	require.NoError(t, err)

	// Get diffs
	diffs, err := gr.GetDiffs([]string{"README.md"})
	require.NoError(t, err)

	expectedDiff := `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1 +1 @@
-# Test Repository
+# Modified Test Repository
`
	assert.Contains(t, diffs, expectedDiff)
}

func TestIsDirty(t *testing.T) {
	gr := setupTestRepo(t)

	// Initially clean
	assert.False(t, gr.IsDirty())

	// Create untracked file
	newFile := filepath.Join(gr.root, "dirty.txt")
	err := os.WriteFile(newFile, []byte("dirty\n"), 0644)
	require.NoError(t, err)

	// Verify dirty state
	assert.True(t, gr.IsDirty())
}

func TestGetUncommittedChanges(t *testing.T) {
	gr := setupTestRepo(t)

	// Create multiple files
	files := []string{"file1.txt", "file2.txt"}
	for _, f := range files {
		path := filepath.Join(gr.root, f)
		err := os.WriteFile(path, []byte(f+" content\n"), 0644)
		require.NoError(t, err)
	}

	// Stage one file
	_, err := gr.worktree.Add("file1.txt")
	require.NoError(t, err)

	// Get uncommitted changes
	changes, err := gr.GetUncommittedChanges()
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"file1.txt", "file2.txt"}, changes)
}

func TestIsIgnored(t *testing.T) {
	gr := setupTestRepo(t)

	// Create .gitignore
	ignorePath := filepath.Join(gr.root, ".gitignore")
	err := os.WriteFile(ignorePath, []byte("*.tmp\n"), 0644)
	require.NoError(t, err)

	// Create ignored file
	tmpFile := filepath.Join(gr.root, "test.tmp")
	err = os.WriteFile(tmpFile, []byte("ignored\n"), 0644)
	require.NoError(t, err)

	// Check if ignored
	ignored, err := gr.IsIgnoredFile("test.tmp")
	require.NoError(t, err)
	assert.True(t, ignored)
}

func TestCommitAttribution(t *testing.T) {
	gr := setupTestRepo(t)
	gr.attributeAuthor = true
	gr.attributeCommitMessageAuthor = true

	// Create test file
	testFile := filepath.Join(gr.root, "attribution.txt")
	err := os.WriteFile(testFile, []byte("test\n"), 0644)
	require.NoError(t, err)

	// Commit with attribution
	_, msg, err := gr.Commit(
		[]string{"attribution.txt"},
		"",
		"Test attribution",
		true,
	)
	require.NoError(t, err)

	// Verify commit message
	assert.Equal(t, "aider: Test attribution", msg)

	// Verify author attribution
	head, err := gr.repo.Head()
	require.NoError(t, err)
	commit, err := gr.repo.CommitObject(head.Hash())
	require.NoError(t, err)
	assert.Contains(t, commit.Author.Name, "(aider)")
}

func TestGetHeadCommit(t *testing.T) {
	gr := setupTestRepo(t)

	// Get initial commit
	initialHash, err := gr.GetHeadCommitSHA(false)
	require.NoError(t, err)

	// Create new commit
	newFile := filepath.Join(gr.root, "head.txt")
	err = os.WriteFile(newFile, []byte("head test\n"), 0644)
	require.NoError(t, err)
	_, _, err = gr.Commit([]string{"head.txt"}, "", "Head test commit", false)
	require.NoError(t, err)

	// Get new commit hash
	newHash, err := gr.GetHeadCommitSHA(false)
	require.NoError(t, err)
	assert.NotEqual(t, initialHash, newHash)

	// Verify commit message
	msg, err := gr.GetHeadCommitMessage("")
	require.NoError(t, err)
	assert.Equal(t, "Head test commit", msg)
}
