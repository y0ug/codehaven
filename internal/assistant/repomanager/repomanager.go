package repomanager

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/extractors"
	"github.com/y0ug/codehaven/internal/assistant/llm"
	"github.com/y0ug/codehaven/internal/filemanager"
	"github.com/y0ug/codehaven/internal/gitrepo"
	"github.com/y0ug/llmhaven/chat"
)

type Completion func(ctx context.Context,
	messages []*chat.ChatMessage, tools []chat.Tool,
	opts ...func(*llm.Params)) (*chat.ChatResponse, error)

// Fence represents a pair of opening and closing delimiters for code blocks

type RepoManager struct {
	fence            extractors.Fence
	root             string
	logger           *slog.Logger
	fm               filemanager.FileManager
	git              gitrepo.GitRepoInterface
	absRootPathCache map[string]string
	completion       Completion
}
type FileGitStatus struct {
	InGit    bool
	IsDirty  bool
	IsStaged bool
}

var _ RepoManagerInterface = &RepoManager{}

type RepoManagerInterface interface {
	// GetFM() filemanager.FileManager
	GetGit() gitrepo.GitRepoInterface
	GetFence() [2]string
	GetRoot() string
	ChooseFence()
	GetFilesContent() string
	GetReadOnlyFilesContent() string
	GetRepoMap() string

	ApplyEdit(edits actions.FileEditAction) error

	CheckGitStatus(paths []string) (map[string]FileGitStatus, error)
	ValidateGitState(bool) error

	AddFiles(readOnly bool, filename ...string) error
	RemoveFiles(file ...string) error
	WriteFile(path string, newContent string) error
	GetFile(path string) (string, bool, error)
	ListFiles(filters filemanager.FileFilters) map[string]*filemanager.FileInfo

	AutoCommit(ctx context.Context) (string, string, error)
	Commit(message string) error

	CreateSnapshot() map[string]string
	GetDiffSummary(snapshot map[string]string) string
}

func NewRepoManager(
	root string,
	logger *slog.Logger,
	fm filemanager.FileManager,
	git gitrepo.GitRepoInterface,
	completion Completion,
) *RepoManager {
	return &RepoManager{
		root:             root,
		logger:           logger,
		fm:               fm,
		git:              git,
		absRootPathCache: make(map[string]string),
		completion:       completion,
	}
}

func (c *RepoManager) GetRoot() string {
	return c.root
}

func (c *RepoManager) GetGit() gitrepo.GitRepoInterface {
	return c.git
}

// func (c *RepoManager) GetFM() filemanager.FileManager {
// 	return c.fm
// }

func (c *RepoManager) GetFence() [2]string {
	return c.fence
}

// chooseFence selects appropriate fence markers that won't conflict with file contents
func (c *RepoManager) ChooseFence() {
	// Get all content from files to check for fence conflicts
	allContent := c.getAllContent()

	// Try each fence option until we find one that doesn't appear in the content
	for _, fence := range extractors.DefaultFences {
		if !hasFenceConflict(allContent, fence) {
			c.fence = fence
			return
		}
	}

	// If all fences conflict (unlikely), use the default and warn
	c.fence = extractors.DefaultFences[0]
	c.logger.Warn(
		"Unable to find a non-conflicting fence strategy! Falling back", "fence", c.fence)
}

// getAllContent combines content from all files being handled
func (c *RepoManager) getAllContent() string {
	var builder strings.Builder

	// Get all files
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterAll))

	for _, info := range files {
		builder.WriteString(info.Content)
		builder.WriteString("\n")
	}

	return builder.String()
}

// hasFenceConflict checks if fence markers appear in the content
func hasFenceConflict(content string, fence extractors.Fence) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, fence[0]) || strings.HasPrefix(line, fence[1]) {
			return true
		}
	}
	return false
}

// getRelativePath converts an absolute path to a path relative to the repository root or working directory
func (c *RepoManager) getRelativePath(absPath string) string {
	// Check cache first
	if relPath, ok := c.absRootPathCache[absPath]; ok {
		return relPath
	}

	// Get relative path
	relPath, err := filepath.Rel(c.root, absPath)
	if err != nil {
		// If we can't get relative path, return absolute path
		return absPath
	}

	// Cache and return the result
	c.absRootPathCache[absPath] = relPath
	return relPath
}

// absRootPath converts a relative path to absolute path using root directory
func (c *RepoManager) absRootPath(path string) string {
	// Check cache first
	if cached, ok := c.absRootPathCache[path]; ok {
		return cached
	}

	// Join with root and get absolute path
	absPath := filepath.Join(c.root, path)
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		// If we can't get absolute path, return joined path
		absPath = filepath.Join(c.root, path)
	}

	// Cache and return result
	c.absRootPathCache[path] = absPath
	return absPath
}

func (c *RepoManager) GetFilesContent() string {
	var content string
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterEditable))

	for fname, info := range files {
		relPath := c.getRelativePath(fname)
		content += "\n" + relPath + "\n"
		content += c.fence[0] + "\n"
		content += info.Content
		content += c.fence[1] + "\n"
	}
	return content
}

func (c *RepoManager) GetReadOnlyFilesContent() string {
	var content string
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterReadOnly))

	for fname, info := range files {
		relPath := c.getRelativePath(fname)
		content += "\n" + relPath + "\n"
		content += c.fence[0] + "\n"
		content += info.Content
		content += c.fence[1] + "\n"
	}
	return content
}

func (gfm *RepoManager) CheckGitStatus(
	paths []string,
) (map[string]FileGitStatus, error) {
	result := make(map[string]FileGitStatus)

	for _, path := range paths {
		status := FileGitStatus{}

		// Check if file is in git
		status.InGit = gfm.git.PathInRepo(path)

		// Check if file is dirty
		status.IsDirty = gfm.git.IsFileDirty(path)

		// Check if file is staged
		// We can get this from git status porcelain output
		status.IsStaged = false // TODO: implement proper staging check

		result[path] = status
	}

	return result, nil
}

func (c *RepoManager) AddFiles(readOnly bool, files ...string) error {
	// First check Git status of all files
	statuses, err := c.CheckGitStatus(files)
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// Check if any files need attention
	var dirtyFiles []string
	var unversionedFiles []string

	for file, status := range statuses {
		if !status.InGit {
			unversionedFiles = append(unversionedFiles, file)
		}
		if status.IsDirty || status.IsStaged {
			dirtyFiles = append(dirtyFiles, file)
		}
	}

	// Handle unversioned files
	if len(unversionedFiles) > 0 {
		err = fmt.Errorf("files not in git repository: %v", unversionedFiles)
		c.logger.Warn("AddFiles", "error", err)
	}

	// Handle dirty/staged files
	if len(dirtyFiles) > 0 {
		err = fmt.Errorf("files have uncommitted changes: %v", dirtyFiles)
		c.logger.Warn("AddFiles", "error", err)
	}

	// Now we can safely add files to our manager
	for _, file := range files {
		if err := c.fm.Add(file, readOnly); err != nil {
			return fmt.Errorf("failed to add file %s: %w", file, err)
		}
	}

	return nil
}

func (c *RepoManager) RemoveFiles(files ...string) error {
	for _, file := range files {
		err := c.fm.Remove(file)
		if err != nil {
			c.logger.Warn("error removing file", "filename", file, "error", err)
		}
	}
	return nil
}

func (c *RepoManager) ListFiles(filters filemanager.FileFilters) map[string]*filemanager.FileInfo {
	return c.fm.List(filters)
}

func (c *RepoManager) GetFile(path string) (string, bool, error) {
	return c.fm.Get(path)
}

func (c *RepoManager) WriteFile(path string, newContent string) error {
	return c.fm.Write(path, newContent)
}

func (c *RepoManager) ValidateGitState(autoCommit bool) error {
	// Get all files in current conversation scope
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterEditable))

	// Check if we have uncommitted changes
	needsCommit := false
	var notInRepo []string
	var dirtyFiles []string

	for path := range files {
		// Check if file is in repo
		if !c.git.PathInRepo(path) {
			notInRepo = append(notInRepo, path)
			continue
		}

		// Check if file has uncommitted changes
		if c.git.IsFileDirty(path) {
			dirtyFiles = append(dirtyFiles, path)
			needsCommit = true
		}
	}

	if len(notInRepo) > 0 {
		return fmt.Errorf("files not in git repository: %v", notInRepo)
	}

	if needsCommit {
		// Option 1: Return error asking user to commit
		// return fmt.Errorf(
		// 	"uncommitted changes in files: %v. Please commit changes before proceeding",
		// 	dirtyFiles,
		// )
		c.logger.Info(
			"uncommitted changes in files: %v. Please auto-commit",
			"dirtyFiles",
			dirtyFiles,
		)
		c.AutoCommit(context.Background())
		// Option 2: Auto-commit changes
		/*
		   commitMsg := "Auto-commit before LLM request"
		   if _, _, err := a.rm.GetRepo().Commit(dirtyFiles, "", commitMsg, false); err != nil {
		       return fmt.Errorf("failed to auto-commit changes: %w", err)
		   }
		*/
	}

	return nil
}

func (c *RepoManager) ApplyEdit(edit actions.FileEditAction) error {
	content, isEditable, err := c.fm.Get(edit.Filename)
	if err != nil {
		c.logger.Error("error reading file ", "filename", edit.Filename, "error", err)
		c.logger.Info("file is not in the file list adding it", "filename", edit.Filename)
		content = ""
		c.fm.Add(edit.Filename, false)
	} else if !isEditable {
		c.logger.Info("file is read-only we will not edit it", "filename", edit.Filename)
		return fmt.Errorf("file %s is read-only we will not edit it", edit.Filename)
	}

	newContent := extractors.ApplyEdit(string(content), edit.Original, edit.Updated)
	if newContent == string(content) {
		err = fmt.Errorf("new content is the same as the original content")
	}

	c.logger.Info(
		"applying edit to file",
		"filename",
		edit.Filename,
		"content",
		string(content),
		"new_content",
		newContent,
		"original",
		edit.Original,
		"updated",
		edit.Updated,
	)
	if newContent != string(content) {
		err := c.fm.Write(edit.Filename, newContent)
		if err != nil {
			return fmt.Errorf("error writing file %s: %w", edit.Filename, err)
		}
	}

	return nil
}

func (c *RepoManager) Commit(message string) error {
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterNew))
	var paths []string
	for path := range files {
		paths = append(paths, path)
	}

	c.logger.Info("committing changes", "files", paths, "message", message)
	_, _, err := c.git.Commit(paths, "", message, false)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

func (c *RepoManager) AutoCommit(ctx context.Context) (string, string, error) {
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterNew))
	var paths []string
	for path := range files {
		paths = append(paths, path)
	}

	// Sending LLM request to get a commit message
	diffs, err := c.git.GetDiffs(paths)
	if err != nil {
		return "", "", fmt.Errorf("getting diffs: %w", err)
	}
	commitPrompt := `You are an expert software engineer that generates concise, \
one-line Git commit messages based on the provided diffs.
Review the provided context and diffs which are about to be committed to a git repo.
Review the diffs carefully.
Generate a one-line commit message for those changes.
The commit message should be structured as follows: <type>: <description>
Use these for <type>: fix, feat, build, chore, ci, docs, style, refactor, perf, test

Ensure the commit message:
- Starts with the appropriate prefix.
- Is in the imperative mood (e.g., \"Add feature\" not \"Added feature\" or \"Adding feature\").
- Does not exceed 72 characters.

Reply only with the one-line commit message, without any additional text, explanations, \
or line breaks.`

	chatMsg := []*chat.ChatMessage{
		chat.NewMessage("system", chat.NewTextContent(commitPrompt)),
		chat.NewMessage("user", chat.NewTextContent(diffs)),
	}

	c.logger.Debug("commit", "diffs", diffs)
	resp, err := c.completion(ctx, chatMsg, nil)
	if err != nil {
		return "", "", fmt.Errorf("error getting response from completion: %w", err)
	}

	if len(resp.Choice) == 0 {
		return "", "", fmt.Errorf("no response from completion")
	}

	commitMsg := resp.Choice[0].Content[0].Text

	c.logger.Info("committing changes", "files", paths, "message", commitMsg)
	commitHash, msg, err := c.git.Commit(paths, "", commitMsg, true)
	if err != nil {
		return "", "", fmt.Errorf("failed to commit changes: %w", err)
	}
	c.logger.Info("commit", "hash", commitHash)

	return commitHash, msg, nil
}

func (c *RepoManager) CreateSnapshot() map[string]string {
	snapshot := make(map[string]string)
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterAll))

	for path, info := range files {
		snapshot[path] = info.Content
	}
	return snapshot
}

func (c *RepoManager) GetDiffSummary(snapshot map[string]string) string {
	current := c.CreateSnapshot()
	var diffs strings.Builder

	// Check for modified files
	for path, currentContent := range current {
		if original, exists := snapshot[path]; exists && original != currentContent {
			diffs.WriteString(GenerateDiff(original, currentContent))
		}
	}

	return diffs.String()
}

func GenerateDiff(original, updated string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(original, updated, false)
	return dmp.DiffPrettyText(diffs)
}

func (c *RepoManager) GenerateRepoMap() *RepoMap {
	var files []string
	var err error
	if c.git != nil {
		files, err = c.git.ListFiles()
		if err != nil {
			c.logger.Error("error listing git files", "error", err)
			return nil
		}
		return NewRepoMap(files)
	} else {
		result := WalkDirectory(WalkOptions{
			RootPath:    c.root,
			IgnoreFiles: []string{".gitignore", ".aiderignore"}, // Custom ignore files
			IncludeDirs: false,
		})
		files = result.Files
	}
	return NewRepoMap(files)
}

func (c *RepoManager) GetRepoMap() string {
	repoMap := c.GenerateRepoMap()
	return repoMap.String()
}
