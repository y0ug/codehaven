package filemanager

//go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=filemanager .  FileManager

import (
	"time"
)

type FileStatus int

const (
	StatusUnknown FileStatus = iota
	StatusUnmodified
	StatusModified
	StatusAdded
	StatusDeleted
	StatusRenamed
	StatusOutOfSync // New status for when file content differs from disk
)

func (s FileStatus) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusUnmodified:
		return "unmodified"
	case StatusModified:
		return "modified"
	case StatusAdded:
		return "added"
	case StatusDeleted:
		return "deleted"
	case StatusRenamed:
		return "renamed"
	case StatusOutOfSync:
		return "out_of_sync"
	default:
		return "invalid"
	}
}

type FileInfo struct {
	Content    string
	Hash       string
	ReadOnly   bool
	LastUpdate time.Time
	Status     FileStatus
	LastSent   string // Stores the hash of the content when it was last sent
}

type FileFilter int

const (
	FilterAll FileFilter = iota
	FilterReadOnly
	FilterEditable
	FilterNew
)

// FileFilters allows combining multiple filters using bitwise operations
type FileFilters int

func (f FileFilters) Has(filter FileFilter) bool {
	return int(f)&(1<<filter) != 0
}

func NewFileFilters(filters ...FileFilter) FileFilters {
	var result FileFilters
	for _, f := range filters {
		result |= 1 << f
	}
	return result
}

// FileVersion represents a specific version of a file
type FileVersion struct {
	Content   string
	Hash      string
	Timestamp time.Time
	CommitMsg string
}

type FileManager interface {
	Add(path string, readOnly bool) error
	Remove(path string) error
	Get(path string) (string, bool, error)
	List(filters FileFilters) map[string]*FileInfo
	Write(path string, newContent string) error

	MarkFileAsSent(path string) error
	HasFileChanged(path string) (bool, error)
	GetNewFiles() map[string]*FileInfo
	IsFileReadOnly(path string) (bool, error)
	GetFileLastUpdate(path string) (time.Time, error)
	GetFileStatus(path string) (FileStatus, error)

	// Version control methods
	// Stage(path string, content string) error
	Commit(msg string) error
	// GetVersions(path string) ([]FileVersion, error)
	// Revert(path string, version string) error
}
