package gitrepo

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRepoExec represents a Git repository
type GitRepoExec struct {
	logger          *slog.Logger
	root            string
	aiderIgnoreFile string

	// Configuration options
	attributeAuthor                 bool
	attributeCommitter              bool
	attributeCommitMessageAuthor    bool
	attributeCommitMessageCommitter bool
	commitPrompt                    string
	subtreeOnly                     bool
}

// // Option is a function type to configure GitRepo
type OptionRepoExec func(*GitRepoExec)

//
// // WithCommitPrompt sets the commit prompt
// func WithCommitPrompt(prompt string) Option {
// 	return func(g *GitRepoExec) {
// 		g.commitPrompt = prompt
// 	}
// }
//
// // WithAiderIgnoreFile sets the aider ignore file
// func WithAiderIgnoreFile(file string) Option {
// 	return func(g *GitRepoExec) {
// 		g.aiderIgnoreFile = file
// 	}
// }
//
// // WithAttributeOptions sets the attribution options
// func WithAttributeOptions(author, committer, msgAuthor, msgCommitter bool) Option {
// 	return func(g *GitRepoExec) {
// 		g.attributeAuthor = author
// 		g.attributeCommitter = committer
// 		g.attributeCommitMessageAuthor = msgAuthor
// 		g.attributeCommitMessageCommitter = msgCommitter
// 	}
// }
//
// // WithSubtreeOnly sets the subtree only option
// func WithSubtreeOnly(subtreeOnly bool) Option {
// 	return func(g *GitRepoExec) {
// 		g.subtreeOnly = subtreeOnly
// 	}
// }

// NewGitRepo creates a new GitRepo instance
func NewGitRepoExec(
	logger *slog.Logger,
	fnames []string,
	gitDname string,
	options ...OptionRepoExec,
) (*GitRepoExec, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	repo := &GitRepoExec{
		logger:                          logger,
		attributeAuthor:                 true,
		attributeCommitter:              true,
		attributeCommitMessageAuthor:    false,
		attributeCommitMessageCommitter: false,
	}

	for _, opt := range options {
		opt(repo)
	}

	var checkPaths []string
	if gitDname != "" {
		checkPaths = []string{gitDname}
	} else if len(fnames) > 0 {
		checkPaths = fnames
	} else {
		checkPaths = []string{"."}
	}

	root, err := repo.findGitRoot(checkPaths)
	if err != nil {
		return nil, fmt.Errorf("finding git root: %w", err)
	}
	repo.root = root

	return repo, nil
}

// GetRoot returns the repository root path
func (g *GitRepoExec) GetRoot() string {
	return g.root
}

// IsInitialized checks if the repository is initialized
func (g *GitRepoExec) IsInitialized() bool {
	return g.root != ""
}

func (g *GitRepoExec) IsDirty() bool {
	// Check both staged and unstaged changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		g.logger.Error("failed to check repository status", "error", err)
		return false
	}

	// Return true if there are any changes
	return len(strings.TrimSpace(string(output))) > 0
}

// IsFileDirty returns true if the file has any staged or unstaged changes
func (g *GitRepoExec) IsFileDirty(path string) bool {
	cmd := exec.Command("git", "status", "--porcelain", path)
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return len(strings.TrimSpace(string(output))) > 0
}

// GetUncommittedChanges returns the list of files with uncommitted changes
func (g *GitRepoExec) GetUncommittedChanges() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting uncommitted changes: %w", err)
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}

// GetCurrentBranch returns the current branch name
func (g *GitRepoExec) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Commit creates a new commit with the specified changes
func (g *GitRepoExec) Commit(
	fnames []string,
	context, message string,
	aiderEdits bool,
) (string, string, error) {
	if len(fnames) == 0 {
		// Check if repo is dirty when no specific files are provided
		if !g.IsDirty() {
			return "", "", fmt.Errorf("no changes to commit")
		}
	} else {
		// When specific files are provided, check if they have changes
		diffs, err := g.GetDiffs(fnames)
		if err != nil {
			return "", "", fmt.Errorf("checking for changes: %w", err)
		}
		if diffs == "" {
			return "", "", fmt.Errorf("no changes to commit")
		}
	}

	diffs, err := g.GetDiffs(fnames)
	if err != nil {
		return "", "", fmt.Errorf("getting diffs: %w", err)
	}
	if diffs == "" {
		return "", "", nil
	}

	commitMsg := message
	if commitMsg == "" {
		var err error
		commitMsg, err = g.getCommitMessage(diffs, context)
		if err != nil {
			return "", "", fmt.Errorf("generating commit message: %w", err)
		}
	}

	if aiderEdits && g.attributeCommitMessageAuthor {
		commitMsg = "aider: " + commitMsg
	} else if g.attributeCommitMessageCommitter {
		commitMsg = "aider: " + commitMsg
	}

	if commitMsg == "" {
		commitMsg = "(no commit message provided)"
	}

	if len(fnames) > 0 {
		for _, fname := range fnames {
			absPath := filepath.Join(g.root, fname)
			cmd := exec.Command("git", "add", absPath)
			cmd.Dir = g.root
			if err := cmd.Run(); err != nil {
				g.logger.Error("failed to add file", "file", fname, "error", err)
				return "", "", fmt.Errorf("adding file %s: %w", fname, err)
			}
		}
	}

	args := []string{"commit", "-m", commitMsg, "--no-verify"}
	if len(fnames) == 0 {
		args = append(args, "-a")
	}

	env := os.Environ()
	if g.attributeCommitter {
		origName, err := g.getGitConfig("user.name")
		if err != nil {
			return "", "", fmt.Errorf("getting user.name: %w", err)
		}
		env = append(env, fmt.Sprintf("GIT_COMMITTER_NAME=%s (aider)", origName))
	}

	if aiderEdits && g.attributeAuthor {
		origName, err := g.getGitConfig("user.name")
		if err != nil {
			return "", "", fmt.Errorf("getting user.name: %w", err)
		}
		env = append(env, fmt.Sprintf("GIT_AUTHOR_NAME=%s (aider)", origName))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		g.logger.Error("failed to commit", "error", err)
		return "", "", fmt.Errorf("committing changes: %w", err)
	}

	hash, err := g.GetHeadCommitSHA(true)
	if err != nil {
		return "", "", fmt.Errorf("getting commit hash: %w", err)
	}

	g.logger.Info("commit created", "hash", hash, "message", commitMsg)
	return hash, commitMsg, nil
}

// GetDiffs returns the git diff for the specified files
func (g *GitRepoExec) GetDiffs(fnames []string) (string, error) {
	var buffer bytes.Buffer

	for _, fname := range fnames {
		if !g.PathInRepo(fname) {
			buffer.WriteString(fmt.Sprintf("Added %s\n", fname))
		}
	}

	args := []string{"diff"}
	if len(fnames) > 0 {
		args = append(args, "--")
		args = append(args, fnames...)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting diffs: %w", err)
	}
	buffer.Write(output)

	return buffer.String(), nil
}

// GetFileContent returns the content of a file at a specific commit
func (g *GitRepoExec) GetFileContent(fname string, commit string) (string, error) {
	if commit == "" {
		commit = "HEAD"
	}

	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commit, fname))
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting file content: %w", err)
	}
	return string(output), nil
}

// ListFiles returns a list of all tracked files in the repository
func (g *GitRepoExec) ListFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// IsIgnored checks if a file is ignored by git
func (g *GitRepoExec) IsIgnored(fname string) (bool, error) {
	cmd := exec.Command("git", "check-ignore", fname)
	cmd.Dir = g.root
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode() == 0, nil
		}
		return false, fmt.Errorf("checking if file is ignored: %w", err)
	}
	return true, nil
}

// helper functions

func (g *GitRepoExec) findGitRoot(paths []string) (string, error) {
	var repoPaths []string

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			g.logger.Warn("failed to get absolute path", "path", path, "error", err)
			continue
		}

		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = absPath
		out, err := cmd.Output()
		if err == nil {
			repoPath := strings.TrimSpace(string(out))
			repoPaths = append(repoPaths, repoPath)
		}
	}

	if len(repoPaths) == 0 {
		return "", fmt.Errorf("no git repository found")
	}

	for i := 1; i < len(repoPaths); i++ {
		if repoPaths[i] != repoPaths[0] {
			return "", fmt.Errorf("files are in different git repos")
		}
	}

	return repoPaths[0], nil
}

func (g *GitRepoExec) getGitConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (g *GitRepoExec) GetHeadCommitSHA(short bool) (string, error) {
	args := []string{"rev-parse"}
	if short {
		args = append(args, "--short")
	}
	args = append(args, "HEAD")

	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting HEAD commit SHA: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (g *GitRepoExec) PathInRepo(path string) bool {
	cmd := exec.Command("git", "ls-files", "--error-unmatch", path)
	cmd.Dir = g.root
	return cmd.Run() == nil
}

func (g *GitRepoExec) getCommitMessage(diffs, context string) (string, error) {
	content := ""
	if context != "" {
		content += context + "\n"
	}
	content += "# Diffs:\n" + diffs

	// Here you would integrate with your commit message generation logic
	// For now, we'll return a placeholder message
	return "Automated commit", nil
}

func (g *GitRepoExec) DiffCommits(pretty bool, fromCommit, toCommit string) (string, error) {
	args := []string{"diff"}

	if pretty {
		args = append(args, "--color")
	} else {
		args = append(args, "--no-color")
	}

	args = append(args, fromCommit, toCommit)

	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting diff between commits: %w", err)
	}

	return string(output), nil
}

func (g *GitRepoExec) GetDirtyFiles() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting dirty files: %w", err)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		// Check if this is a renamed file (starts with R)
		if strings.HasPrefix(line, "R") {
			// For renamed files, take everything after the status codes up to " ->"
			parts := strings.Split(line[3:], " -> ")
			path := strings.TrimSpace(parts[1])
			if path != "" {
				files = append(files, path)
			}
			continue
		}

		// For all other cases, take everything after the status codes
		path := strings.TrimSpace(line[2:])
		if path != "" {
			files = append(files, path)
		}
	}
	return files, nil
}

// GetHeadCommitMessage returns the message of the HEAD commit
func (g *GitRepoExec) GetHeadCommitMessage(defaultMsg string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%B")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		if defaultMsg != "" {
			return defaultMsg, nil
		}
		return "", fmt.Errorf("getting HEAD commit message: %w", err)
	}
	msg := strings.TrimSpace(string(output))
	if msg == "" && defaultMsg != "" {
		return defaultMsg, nil
	}
	return msg, nil
}

func (g *GitRepoExec) GetHeadCommit() (string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%H - %an (%ad) - %s")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting HEAD commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetTrackedFiles returns a list of all tracked files in the repository
func (g *GitRepoExec) GetTrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = g.root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting tracked files: %w", err)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// IsIgnoredFile checks if a file is ignored by Git
func (g *GitRepoExec) IsIgnoredFile(fname string) bool {
	cmd := exec.Command("git", "check-ignore", "-q", fname)
	cmd.Dir = g.root

	// git check-ignore returns:
	// - exit code 0 if file is ignored
	// - exit code 1 if file is not ignored
	// - exit code 128 for errors (e.g., invalid .gitignore syntax)
	err := cmd.Run()
	return err == nil
}
