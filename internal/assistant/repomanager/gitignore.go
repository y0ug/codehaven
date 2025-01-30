package repomanager

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// WalkOptions contains configuration for directory walking
type WalkOptions struct {
	RootPath    string
	IgnoreFiles []string // List of ignore filenames to use (e.g. [".gitignore", ".aiderignore"])
	IncludeDirs bool     // Whether to include directories in results
}

// WalkResult contains the results of directory walking
type WalkResult struct {
	Files   []string // List of relative file paths
	Ignored []string // List of ignored paths
	Errors  []error  // Errors encountered during walking
}

type IgnoreMatcher struct {
	patterns []gitignore.Pattern
	matcher  gitignore.Matcher
	fs       billy.Filesystem
}

func (im *IgnoreMatcher) IsIgnored(path string, isDir bool) bool {
	// Convert path to relative segments
	relPath := strings.TrimPrefix(path, im.fs.Root())
	segments := strings.Split(relPath, string(filepath.Separator))

	// Filter out empty segments
	cleanSegments := make([]string, 0, len(segments))
	for _, s := range segments {
		if s != "" {
			cleanSegments = append(cleanSegments, s)
		}
	}

	return im.matcher.Match(cleanSegments, isDir)
}

func NewIgnoreMatcher(rootPath string, ignoreFiles []string) (*IgnoreMatcher, error) {
	fs := osfs.New(rootPath)
	path := []string{}

	var patterns []gitignore.Pattern

	for _, ignoreFile := range ignoreFiles {
		ps, err := readIgnoreFile(fs, path, ignoreFile)
		if err == nil {
			patterns = append(patterns, ps...)
		}
	}

	// Read directory-specific ignore files
	fis, err := fs.ReadDir(fs.Join(path...))
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if fi.IsDir() && fi.Name() != ".git" {
			if gitignore.NewMatcher(patterns).Match(append(path, fi.Name()), true) {
				continue
			}

			var subps []gitignore.Pattern
			for _, ignoreFile := range ignoreFiles {
				ps, err := readIgnoreFile(fs, append(path, fi.Name()), ignoreFile)
				if err == nil {
					subps = append(subps, ps...)
				}
			}

			if len(subps) > 0 {
				patterns = append(patterns, subps...)
			}
		}
	}

	return &IgnoreMatcher{
		patterns: patterns,
		matcher:  gitignore.NewMatcher(patterns),
		fs:       fs,
	}, nil
}

func WalkDirectory(opts WalkOptions) *WalkResult {
	result := &WalkResult{}

	matcher, err := NewIgnoreMatcher(opts.RootPath, opts.IgnoreFiles)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result
	}

	filepath.Walk(opts.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		relPath, err := filepath.Rel(opts.RootPath, path)
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		if relPath == "." {
			return nil
		}

		// Skip the ignore files themselves
		// for _, ignoreFile := range opts.IgnoreFiles {
		// 	if relPath == ignoreFile {
		// 		return nil
		// 	}
		// }

		// Check if path is ignored
		isIgnored := matcher.IsIgnored(relPath, info.IsDir())
		if isIgnored {
			result.Ignored = append(result.Ignored, relPath)
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Record directories if requested
		if info.IsDir() {
			if opts.IncludeDirs {
				result.Files = append(result.Files, relPath+string(filepath.Separator))
			}
		} else {
			result.Files = append(result.Files, relPath)
		}

		return nil
	})

	return result
}

// Also update readIgnoreFile to properly handle whitespace
func readIgnoreFile(
	fs billy.Filesystem,
	path []string,
	ignoreFile string,
) (ps []gitignore.Pattern, err error) {
	f, err := fs.Open(fs.Join(append(path, ignoreFile)...))
	if err == nil {
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if s != "" && !strings.HasPrefix(s, "#") {
				ps = append(ps, gitignore.ParsePattern(s, path))
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return
}
