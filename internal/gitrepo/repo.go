package gitrepo

//go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=gitrepo . GitRepoInterface,IOInterface

// GitRepoInterface defines the contract for git repository operations
type GitRepoInterface interface {
	Commit(fnames []string, context string, message string, aiderEdits bool) (string, string, error)
	GetDiffs(fnames []string) (string, error)
	// DiffCommits(pretty bool, fromCommit, toCommit string) (string, error)
	// GetTrackedFiles() ([]string, error)
	IsIgnoredFile(fname string) (bool, error)
	PathInRepo(path string) bool
	// GetDirtyFiles() ([]string, error)
	IsDirty() bool
	IsFileDirty(path string) bool
	// GetHeadCommit() (string, error)
	GetHeadCommitSHA(short bool) (string, error)
	GetHeadCommitMessage(defaultMsg string) (string, error)
	ExecGit(args ...string) (string, error)
	ListFiles() ([]string, error)
}

// _ GitRepoInterface = (*GitRepoExec)(nil)
var _ GitRepoInterface = (*GitRepo)(nil)

// IOInterface defines methods for IO operations
type IOInterface interface {
	ToolError(msg string)
	ToolOutput(msg string, bold bool)
	ToolWarning(msg string)
}

// Config holds the configuration for creating a new GitRepo
type Config struct {
	IO                              IOInterface
	Fnames                          []string
	GitDname                        string
	AiderIgnoreFile                 string
	AttributeAuthor                 bool
	AttributeCommitter              bool
	AttributeCommitMessageAuthor    bool
	AttributeCommitMessageCommitter bool
	CommitPrompt                    string
	SubtreeOnly                     bool
}
