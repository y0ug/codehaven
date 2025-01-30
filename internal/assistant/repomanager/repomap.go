// internal/assistant/repomanager/repomap.go
package repomanager

import (
	"fmt"
	"sort"
	"strings"
)

type RepoMap struct {
	files []string
	tree  map[string]interface{}
}

func NewRepoMap(files []string) *RepoMap {
	rm := &RepoMap{
		files: files,
		tree:  make(map[string]interface{}),
	}
	rm.buildTree()
	return rm
}

func (rm *RepoMap) Generate() {
	rm.buildTree()
}

func (rm *RepoMap) buildTree() {
	rm.tree = make(map[string]interface{})
	for _, file := range rm.files {
		if file == "" {
			continue
		}
		parts := strings.Split(file, "/")
		current := rm.tree
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = nil // File
			} else {
				if _, ok := current[part]; !ok {
					current[part] = make(map[string]interface{})
				}
				current = current[part].(map[string]interface{})
			}
		}
	}
}

func (rm *RepoMap) String() string {
	var sb strings.Builder
	rm.buildTreeString(&sb, rm.tree, 0)
	return sb.String()
}

func (rm *RepoMap) buildTreeString(sb *strings.Builder, node map[string]interface{}, level int) {
	entries := make([]string, 0, len(node))
	for entry := range node {
		entries = append(entries, entry)
	}
	sort.Strings(entries)

	for _, entry := range entries {
		indent := strings.Repeat("  ", level)
		child := node[entry]

		if child == nil {
			fmt.Fprintf(sb, "%s%s\n", indent, entry)
		} else {
			fmt.Fprintf(sb, "%s%s/\n", indent, entry)
			rm.buildTreeString(sb, child.(map[string]interface{}), level+1)
		}
	}
}

// Additional methods that can be added later
func (rm *RepoMap) GetFiles() []string {
	return rm.files
}

func (rm *RepoMap) GetTree() map[string]interface{} {
	return rm.tree
}

func (rm *RepoMap) Filter(extensions []string) *RepoMap {
	filtered := make([]string, 0)
	for _, file := range rm.files {
		for _, ext := range extensions {
			if strings.HasSuffix(file, ext) {
				filtered = append(filtered, file)
				break
			}
		}
	}
	return NewRepoMap(filtered)
}

func (rm *RepoMap) CountFiles() int {
	count := 0
	var countRecursive func(node map[string]interface{})

	countRecursive = func(node map[string]interface{}) {
		for _, v := range node {
			if v == nil {
				count++
			} else {
				countRecursive(v.(map[string]interface{}))
			}
		}
	}

	countRecursive(rm.tree)
	return count
}
