package filemanager

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"
)

type VersionedFileManager struct {
	files map[string]*FileInfo
	// Staged changes waiting to be committed
	staged map[string]string
	// Version history per file
	versions map[string][]FileVersion
	mu       sync.RWMutex
}

func NewVersionedFileManager() *VersionedFileManager {
	return &VersionedFileManager{
		files:    make(map[string]*FileInfo),
		staged:   make(map[string]string),
		versions: make(map[string][]FileVersion),
	}
}

func (fm *VersionedFileManager) Add(path string, readOnly bool) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", path, err)
	}

	hasher := sha256.New()
	hasher.Write(content)
	newHash := hex.EncodeToString(hasher.Sum(nil))

	status := StatusAdded
	if fi, exists := fm.files[path]; exists {
		if fi.Hash == newHash {
			// File already exists and has not changed
			return nil
		}
		status = StatusModified
	}

	fm.files[path] = &FileInfo{
		Content:    string(content),
		Hash:       newHash,
		ReadOnly:   readOnly,
		LastUpdate: time.Now(),
		Status:     status,
	}

	version := FileVersion{
		Content:   string(content),
		Hash:      newHash,
		Timestamp: time.Now(),
		CommitMsg: "Updated from disk",
	}

	// Initialize version history
	if _, exists := fm.versions[path]; !exists {
		version.CommitMsg = "Initial version"
		fm.versions[path] = []FileVersion{version}
	} else {
		fm.versions[path] = append(fm.versions[path], version)
	}

	return nil
}

func (fm *VersionedFileManager) Stage(path string, content string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.files[path]; !exists {
		return fmt.Errorf("file %s not found", path)
	}

	fm.staged[path] = content
	return nil
}

func (fm *VersionedFileManager) Commit(msg string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if len(fm.staged) == 0 {
		return fmt.Errorf("no changes staged for commit")
	}

	timestamp := time.Now()

	// Process all staged changes
	for path, content := range fm.staged {
		// Calculate new hash
		hasher := sha256.New()
		hasher.Write([]byte(content))
		newHash := hex.EncodeToString(hasher.Sum(nil))

		// Create new version
		version := FileVersion{
			Content:   content,
			Hash:      newHash,
			Timestamp: timestamp,
			CommitMsg: msg,
		}

		// Add to version history
		fm.versions[path] = append(fm.versions[path], version)

		// Update file info
		fm.files[path].Content = content
		fm.files[path].Hash = newHash
		fm.files[path].LastUpdate = timestamp
		fm.files[path].Status = StatusModified

		// Write to disk
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("error writing file %s: %w", path, err)
		}
	}

	// Clear staged changes
	fm.staged = make(map[string]string)

	return nil
}

func (fm *VersionedFileManager) GetVersions(path string) ([]FileVersion, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	versions, exists := fm.versions[path]
	if !exists {
		return nil, fmt.Errorf("no version history for file %s", path)
	}

	return versions, nil
}

func (fm *VersionedFileManager) Revert(path string, hash string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	versions, exists := fm.versions[path]
	if !exists {
		return fmt.Errorf("no version history for file %s", path)
	}

	var targetVersion FileVersion
	found := false
	for _, v := range versions {
		if v.Hash == hash {
			targetVersion = v
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("version %s not found for file %s", hash, path)
	}

	// Stage the reverted content
	return fm.Stage(path, targetVersion.Content)
}

// Implement remaining FileManager interface methods...
func (fm *VersionedFileManager) Remove(path string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	delete(fm.files, path)
	delete(fm.versions, path)
	delete(fm.staged, path)
	return nil
}

func (fm *VersionedFileManager) Get(path string) (string, bool, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fileInfo, exists := fm.files[path]
	if !exists {
		return "", false, fmt.Errorf("file %s not found", path)
	}

	return fileInfo.Content, !fileInfo.ReadOnly, nil
}

// ... implement remaining interface methods similar to LocalFileManager
