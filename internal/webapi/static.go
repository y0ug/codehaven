package webapi

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFiles embed.FS

// getFileSystem returns the embedded files as a filesystem
func getFileSystem() (fs.FS, error) {
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	return fsys, nil
}
