package gitrepo

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	gitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitRepo represents a Git repository using go-git
type GitRepo struct {
	logger          *slog.Logger
	repo            *git.Repository
	worktree        *git.Worktree
	root            string
	aiderIgnoreFile string

	attributeAuthor                 bool
	attributeCommitter              bool
	attributeCommitMessageAuthor    bool
	attributeCommitMessageCommitter bool
	commitPrompt                    string
	subtreeOnly                     bool
}

// Option configures GitRepo
type Option func(*GitRepo)

func WithCommitPrompt(prompt string) Option {
	return func(g *GitRepo) {
		g.commitPrompt = prompt
	}
}

func WithAiderIgnoreFile(file string) Option {
	return func(g *GitRepo) {
		g.aiderIgnoreFile = file
	}
}

func WithAttributeOptions(author, committer, msgAuthor, msgCommitter bool) Option {
	return func(g *GitRepo) {
		g.attributeAuthor = author
		g.attributeCommitter = committer
		g.attributeCommitMessageAuthor = msgAuthor
		g.attributeCommitMessageCommitter = msgCommitter
	}
}

func WithSubtreeOnly(subtreeOnly bool) Option {
	return func(g *GitRepo) {
		g.subtreeOnly = subtreeOnly
	}
}

// NewGitRepo initializes a new GitRepo instance using go-git
func NewGitRepo(
	logger *slog.Logger,
	fnames []string,
	gitDname string,
	options ...Option,
) (*GitRepo, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	repo := &GitRepo{
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

	gitRepo, root, err := findGitRepo(checkPaths, logger)
	if err != nil {
		return nil, fmt.Errorf("finding git repo: %w", err)
	}
	repo.repo = gitRepo
	repo.root = root

	repo.worktree, err = gitRepo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}

	return repo, nil
}

func findGitRepo(paths []string, logger *slog.Logger) (*git.Repository, string, error) {
	var (
		foundRepo *git.Repository
		repoRoot  string
	)

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			logger.Warn("failed to get absolute path", "path", path, "error", err)
			continue
		}

		repo, err := git.PlainOpenWithOptions(absPath, &git.PlainOpenOptions{DetectDotGit: true})
		if err != nil {
			continue
		}

		worktree, err := repo.Worktree()
		if err != nil {
			logger.Warn("failed to get worktree", "path", absPath, "error", err)
			continue
		}

		currentRoot := worktree.Filesystem.Root()

		if foundRepo == nil {
			foundRepo = repo
			repoRoot = currentRoot
		} else {
			if currentRoot != repoRoot {
				return nil, "", fmt.Errorf("files are in different git repos")
			}
		}
	}

	if foundRepo == nil {
		return nil, "", fmt.Errorf("no git repository found")
	}

	return foundRepo, repoRoot, nil
}

func (g *GitRepo) GetRoot() string {
	return g.root
}

func (g *GitRepo) IsInitialized() bool {
	return g.repo != nil
}

func (g *GitRepo) IsDirty() bool {
	status, err := g.worktree.Status()
	if err != nil {
		g.logger.Error("failed to get repository status", "error", err)
		return false
	}
	return !status.IsClean()
}

func (g *GitRepo) IsFileDirty(path string) bool {
	status, err := g.worktree.Status()
	if err != nil {
		return false
	}
	fileStatus, ok := status[path]
	if !ok {
		return false
	}
	return fileStatus.Staging != git.Unmodified || fileStatus.Worktree != git.Unmodified
}

func (g *GitRepo) GetUncommittedChanges() ([]string, error) {
	status, err := g.worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("getting status: %w", err)
	}

	var files []string
	for file, s := range status {
		if s.Staging != git.Unmodified || s.Worktree != git.Unmodified {
			files = append(files, file)
		}
	}
	return files, nil
}

func (g *GitRepo) GetCurrentBranch() (string, error) {
	head, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}
	if !head.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not a branch")
	}
	return head.Name().Short(), nil
}

func (g *GitRepo) Commit(
	fnames []string,
	context, message string,
	aiderEdits bool,
) (string, string, error) {
	if len(fnames) == 0 && !g.IsDirty() {
		return "", "", fmt.Errorf("no changes to commit")
	}

	// Check specific files if provided
	if len(fnames) > 0 {
		hasChanges := false
		for _, fname := range fnames {
			if g.IsFileDirty(fname) {
				hasChanges = true
				break
			}
		}
		if !hasChanges {
			return "", "", fmt.Errorf("no changes to commit in specified files")
		}
	}

	// Generate commit message
	commitMsg := message
	if commitMsg == "" {
		diffs, err := g.GetDiffs(fnames)
		if err != nil {
			return "", "", fmt.Errorf("getting diffs: %w", err)
		}
		commitMsg, err = g.getCommitMessage(diffs, context)
		if err != nil {
			return "", "", fmt.Errorf("generating commit message: %w", err)
		}
	}

	// Add attribution prefixes
	if aiderEdits && g.attributeCommitMessageAuthor {
		commitMsg = "aider: " + commitMsg
	} else if g.attributeCommitMessageCommitter {
		commitMsg = "aider: " + commitMsg
	}

	if commitMsg == "" {
		commitMsg = "(no commit message provided)"
	}

	// Stage files
	if len(fnames) > 0 {
		for _, fname := range fnames {
			if _, err := g.worktree.Add(fname); err != nil {
				g.logger.Error("failed to add file", "file", fname, "error", err)
				return "", "", fmt.Errorf("adding file %s: %w", fname, err)
			}
		}
	} else {
		// Stage all changes
		if err := g.worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
			return "", "", fmt.Errorf("staging all changes: %w", err)
		}
	}

	// Configure commit author/committer
	cfg, err := g.repo.ConfigScoped(config.LocalScope)
	if err != nil {
		return "", "", fmt.Errorf("getting git config: %w", err)
	}

	author := &object.Signature{
		Name:  cfg.User.Name,
		Email: cfg.User.Email,
		When:  time.Now(),
	}
	if aiderEdits && g.attributeAuthor {
		author.Name = fmt.Sprintf("%s (aider)", author.Name)
	}

	committer := &object.Signature{
		Name:  cfg.User.Name,
		Email: cfg.User.Email,
		When:  time.Now(),
	}
	if g.attributeCommitter {
		committer.Name = fmt.Sprintf("%s (aider)", committer.Name)
	}

	// Perform commit
	hash, err := g.worktree.Commit(commitMsg, &git.CommitOptions{
		Author:    author,
		Committer: committer,
	})
	if err != nil {
		return "", "", fmt.Errorf("committing changes: %w", err)
	}

	// Get short hash
	commitObj, err := g.repo.CommitObject(hash)
	if err != nil {
		return "", "", fmt.Errorf("getting commit object: %w", err)
	}
	shortHash := commitObj.Hash.String()[:7]

	g.logger.Info("commit created", "hash", shortHash, "message", commitMsg)
	return commitObj.Hash.String(), commitMsg, nil
}

//	func (g *GitRepo) GetDiffs(fnames []string) (string, error) {
//		var buf bytes.Buffer
//
//		head, err := g.repo.Head()
//		if err != nil {
//			return "", fmt.Errorf("getting HEAD: %w", err)
//		}
//
//		headCommit, err := g.repo.CommitObject(head.Hash())
//		if err != nil {
//			return "", fmt.Errorf("getting head commit: %w", err)
//		}
//
//		headTree, err := headCommit.Tree()
//		if err != nil {
//			return "", fmt.Errorf("getting head tree: %w", err)
//		}
//
//		status, err := g.worktree.Status()
//		if err != nil {
//			return "", fmt.Errorf("getting status: %w", err)
//		}
//
//		// Handle untracked files
//		for _, fname := range fnames {
//			if status.File(fname).Worktree == git.Untracked {
//				buf.WriteString(fmt.Sprintf("Added %s\n", fname))
//			}
//		}
//
//		// Generate diff between working tree and HEAD
//		patch, err := g.worktree.Diff(&git.DiffOptions{
//			Paths: fnames,
//		})
//		if err != nil {
//			return "", fmt.Errorf("generating diff: %w", err)
//		}
//
//		patchText, err := patch.String()
//		if err != nil {
//			return "", fmt.Errorf("rendering diff: %w", err)
//		}
//		buf.WriteString(patchText)
//
//		return buf.String(), nil
//	}
// func (g *GitRepo) GetDiffs(fnames []string) (string, error) {
// 	var buf bytes.Buffer
//
// 	head, err := g.repo.Head()
// 	if err != nil {
// 		return "", fmt.Errorf("getting HEAD: %w", err)
// 	}
//
// 	headCommit, err := g.repo.CommitObject(head.Hash())
// 	if err != nil {
// 		return "", fmt.Errorf("getting head commit: %w", err)
// 	}
//
// 	headTree, err := headCommit.Tree()
// 	if err != nil {
// 		return "", fmt.Errorf("getting head tree: %w", err)
// 	}
//
// 	status, err := g.worktree.Status()
// 	if err != nil {
// 		return "", fmt.Errorf("getting status: %w", err)
// 	}
// 	g.logger.Info("status", "status", status, "head", head.Hash())
// 	// Handle untracked/new files first
// 	for _, fname := range fnames {
// 		s := status.File(fname)
// 		g.logger.Info("status", "status", s, "fname", fname)
// 		if status.File(fname).Worktree == git.Untracked {
// 			buf.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", fname, fname))
// 			buf.WriteString(fmt.Sprintf("new file mode 100644\n"))
// 			buf.WriteString(fmt.Sprintf("--- /dev/null\n"))
// 			buf.WriteString(fmt.Sprintf("+++ b/%s\n", fname))
// 			// Add file content as additions
// 			content, err := os.ReadFile(filepath.Join(g.root, fname))
// 			if err == nil {
// 				lines := strings.Split(string(content), "\n")
// 				for _, line := range lines {
// 					buf.WriteString(fmt.Sprintf("+%s\n", line))
// 				}
// 			}
// 			continue
// 		}
// 	}
//
// 	// Generate diffs for modified files
// 	for _, fname := range fnames {
// 		// Skip untracked files already handled
// 		if status.File(fname).Worktree == git.Untracked {
// 			continue
// 		}
//
// 		// Get HEAD file content
// 		headFile, err := headTree.File(fname)
// 		if err != nil {
// 			continue // File doesn't exist in HEAD
// 		}
//
// 		headContent, err := headFile.Contents()
// 		if err != nil {
// 			return "", fmt.Errorf("reading head content for %s: %w", fname, err)
// 		}
//
// 		// Get working tree content
// 		worktreeContentBytes, err := os.ReadFile(filepath.Join(g.root, fname))
// 		if err != nil {
// 			return "", fmt.Errorf("reading worktree file %s: %w", fname, err)
// 		}
// 		worktreeContent := string(worktreeContentBytes)
//
// 		// Generate unified diff
// 		dmp := diffmatchpatch.New()
// 		diffs := dmp.DiffMain(headContent, worktreeContent, false)
// 		buf.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", fname, fname))
// 		buf.WriteString(dmp.DiffPrettyText(diffs))
// 		buf.WriteString("\n")
// 	}
//
// 	return buf.String(), nil
// }

func (g *GitRepo) GetDiffs(fnames []string) (string, error) {
	args := append([]string{"diff", "--no-color", "HEAD", "--"}, fnames...)
	return g.ExecGit(args...)
}

func (g *GitRepo) ExecGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("executing git command: %w", err)
	}
	return string(out), nil
}

func (g *GitRepo) GetFileContent(fname string, commit string) (string, error) {
	var commitObj *object.Commit
	var err error

	if commit == "" {
		head, err := g.repo.Head()
		if err != nil {
			return "", fmt.Errorf("getting HEAD: %w", err)
		}
		commitObj, err = g.repo.CommitObject(head.Hash())
	} else {
		hash, err := g.repo.ResolveRevision(plumbing.Revision(commit))
		if err != nil {
			return "", fmt.Errorf("resolving commit: %w", err)
		}
		commitObj, err = g.repo.CommitObject(*hash)
	}
	if err != nil {
		return "", fmt.Errorf("getting commit: %w", err)
	}

	file, err := commitObj.File(fname)
	if err != nil {
		return "", fmt.Errorf("getting file: %w", err)
	}

	content, err := file.Contents()
	if err != nil {
		return "", fmt.Errorf("reading file content: %w", err)
	}

	return content, nil
}

func (g *GitRepo) ListFiles() ([]string, error) {
	head, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD: %w", err)
	}

	commit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("getting commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("getting tree: %w", err)
	}

	var files []string
	tree.Files().ForEach(func(f *object.File) error {
		files = append(files, f.Name)
		return nil
	})

	return files, nil
}

func (g *GitRepo) IsIgnoredFile(fname string) (bool, error) {
	// Implement using go-git's ignore patterns
	ps, err := gitignore.ReadPatterns(g.worktree.Filesystem, nil)
	if err != nil {
		return false, fmt.Errorf("reading ignore patterns: %w", err)
	}

	matcher := gitignore.NewMatcher(ps)
	relPath, err := filepath.Rel(g.root, filepath.Join(g.root, fname))
	if err != nil {
		return false, fmt.Errorf("getting relative path: %w", err)
	}

	segments := strings.Split(relPath, string(filepath.Separator))
	return matcher.Match(segments, false), nil
}

// Remaining methods (PathInRepo, GetHeadCommitSHA, getCommitMessage, etc.)
// follow similar patterns using go-git's API

func (g *GitRepo) PathInRepo(path string) bool {
	head, err := g.repo.Head()
	if err != nil {
		return false
	}

	commit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		return false
	}

	_, err = commit.File(path)
	return err == nil
}

func (g *GitRepo) GetHeadCommitSHA(short bool) (string, error) {
	head, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}

	hash := head.Hash().String()
	if short {
		hash = hash[:7]
	}
	return hash, nil
}

func (g *GitRepo) getCommitMessage(diffs, context string) (string, error) {
	// Placeholder implementation
	if context != "" {
		return fmt.Sprintf("%s\n\nChanges:\n%s", context, diffs), nil
	}
	return "Automated commit", nil
}

// Additional helper methods for other functionality...
func (g *GitRepo) GetHeadCommitMessage(defaultMsg string) (string, error) {
	head, err := g.repo.Head()
	if err != nil {
		if defaultMsg != "" {
			return defaultMsg, nil
		}
		return "", fmt.Errorf("getting HEAD: %w", err)
	}

	commit, err := g.repo.CommitObject(head.Hash())
	if err != nil {
		if defaultMsg != "" {
			return defaultMsg, nil
		}
		return "", fmt.Errorf("getting commit: %w", err)
	}

	msg := strings.TrimSpace(commit.Message)
	if msg == "" && defaultMsg != "" {
		return defaultMsg, nil
	}
	return msg, nil
}
