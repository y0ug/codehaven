package filemanager

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"
)

type LocalFileManager struct {
	files map[string]*FileInfo
	mu    sync.RWMutex
}

func NewLocalFileManager() *LocalFileManager {
	return &LocalFileManager{
		files: make(map[string]*FileInfo),
	}
}

func (fm *LocalFileManager) Add(path string, readOnly bool) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	content, err := os.ReadFile(path)
	if err != nil {
		// return fmt.Errorf("error reading file %s: %w", path, err)
		// Virtual file until commit
		content = []byte("")
	}

	hasher := sha256.New()
	hasher.Write(content)
	newHash := hex.EncodeToString(hasher.Sum(nil))

	if fileInfo, exists := fm.files[path]; exists {
		if fileInfo.Hash == newHash {
			return nil
		}
	}

	status := StatusAdded
	if _, exists := fm.files[path]; exists {
		status = StatusModified
	}

	fm.files[path] = &FileInfo{
		Content:    string(content),
		Hash:       newHash,
		ReadOnly:   readOnly,
		LastUpdate: time.Now(),
		Status:     status,
	}

	return nil
}

func (fm *LocalFileManager) Remove(path string) error {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	// fileInfo, exists := fm.files[path]
	delete(fm.files, path)
	return nil
}

func (fm *LocalFileManager) List(filters FileFilters) map[string]*FileInfo {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	files := make(map[string]*FileInfo)
	for path, info := range fm.files {
		// If no filters specified, include all files
		if filters == 0 {
			files[path] = info
			continue
		}

		// Check each active filter
		include := false

		if filters.Has(FilterAll) {
			include = true
		}
		if filters.Has(FilterReadOnly) && info.ReadOnly {
			include = true
		}
		if filters.Has(FilterEditable) && !info.ReadOnly {
			include = true
		}
		if filters.Has(FilterNew) && (info.Status == StatusAdded || info.Status == StatusModified) {
			include = true
		}

		if include {
			files[path] = info
		}
	}
	return files
}

func (fm *LocalFileManager) Get(path string) (string, bool, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fileInfo, exists := fm.files[path]
	if !exists {
		return "", false, fmt.Errorf("file %s not found", path)
	}

	// Return content, isEditable, nil
	return fileInfo.Content, !fileInfo.ReadOnly, nil
}

func (fm *LocalFileManager) IsFileReadOnly(path string) (bool, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fileInfo, exists := fm.files[path]
	if !exists {
		return false, fmt.Errorf("file %s not found", path)
	}
	return fileInfo.ReadOnly, nil
}

// GetFileLastUpdate gets the last update time of a file
func (fm *LocalFileManager) GetFileLastUpdate(path string) (time.Time, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	fileInfo, exists := fm.files[path]
	if !exists {
		return time.Time{}, fmt.Errorf("file %s not found", path)
	}
	return fileInfo.LastUpdate, nil
}

// UpdateFileContent updates file content if it's not read-only
func (fm *LocalFileManager) Write(path string, newContent string) error {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	fileInfo, exists := fm.files[path]
	if !exists {
		return fmt.Errorf("file %s not found", path)
	}

	if fileInfo.ReadOnly {
		return fmt.Errorf("cannot modify read-only file %s", path)
	}

	// Calculate new hash
	hasher := sha256.New()
	hasher.Write([]byte(newContent))
	newHash := hex.EncodeToString(hasher.Sum(nil))

	// Update file info and status
	fileInfo.Content = newContent
	fileInfo.Hash = newHash
	fileInfo.LastUpdate = time.Now()
	fileInfo.Status = StatusModified

	return os.WriteFile(path, []byte(newContent), 0644)
}

func (fm *LocalFileManager) Commit(msg string) error {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	for path, fi := range fm.List(NewFileFilters(FilterNew)) {
		if fi.Status == StatusModified {
			err := os.WriteFile(path, []byte(fi.Content), 0644)
			if err != nil {
				fmt.Printf("failed to write file %v\n", err)
			}
			fi.Status = StatusUnmodified
		}
	}
	return nil
}

func (fm *LocalFileManager) GetFileStatus(path string) (FileStatus, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fileInfo, exists := fm.files[path]
	if !exists {
		return StatusUnknown, fmt.Errorf("file %s not found", path)
	}

	// First check if file exists on disk
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fileInfo.Status = StatusDeleted
			return StatusDeleted, nil
		}
		return StatusUnknown, fmt.Errorf("error reading file: %w", err)
	}

	// Compare disk content with our stored content
	hasher := sha256.New()
	hasher.Write(content)
	currentHash := hex.EncodeToString(hasher.Sum(nil))

	if currentHash != fileInfo.Hash {
		fileInfo.Status = StatusOutOfSync
		return StatusOutOfSync, nil
	}

	// Return the tracked status if file matches disk
	return fileInfo.Status, nil
}

func (fm *LocalFileManager) GetFiles() map[string]*FileInfo {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	files := make(map[string]*FileInfo)
	for k, v := range fm.files {
		files[k] = v
	}
	return files
}

func (fm *LocalFileManager) MarkFileAsSent(path string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fileInfo, exists := fm.files[path]
	if !exists {
		return fmt.Errorf("file %s not found", path)
	}

	fileInfo.LastSent = fileInfo.Hash
	return nil
}

func (fm *LocalFileManager) HasFileChanged(path string) (bool, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fileInfo, exists := fm.files[path]
	if !exists {
		return false, fmt.Errorf("file %s not found", path)
	}

	return fileInfo.LastSent != fileInfo.Hash, nil
}

func (fm *LocalFileManager) GetNewFiles() map[string]*FileInfo {
	files := make(map[string]*FileInfo)
	for path := range fm.GetFiles() {
		// Get both status and changed state
		status, err := fm.GetFileStatus(path)
		if err != nil {
			err = fmt.Errorf("error checking file status for %s: %w", path, err)
			fmt.Println(err)
			continue
		}

		changed, err := fm.HasFileChanged(path)
		if err != nil {
			err = fmt.Errorf("error checking file changes for %s: %w", path, err)
			fmt.Println(err)
			continue
		}

		// Add file to context if:
		// 1. It has changed since last send OR
		// 2. It is out of sync with disk OR
		// 3. It has been modified
		if changed || status == StatusOutOfSync ||
			status == StatusModified {
			// content, isEditable, err := fm.GetFileContent(path)
			// if err != nil {
			// 	return nil, fmt.Errorf("error getting content for %s: %w", path, err)
			// }

			files[path] = fm.files[path] // TODO this doesn't respect the mutex

			if err := fm.MarkFileAsSent(path); err != nil {
				err = fmt.Errorf("error marking %s as sent: %w", path, err)
				fmt.Println(err)
				continue
			}
		}
	}
	return files
}
