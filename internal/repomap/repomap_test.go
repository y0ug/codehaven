package repomap

import (
	"fmt"
	"os"
	"testing"
)

func TestTraverseRepo(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "repomap-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// // Clone a test repository
	// repo, err := cloneRepo("https://github.com/y0ug/codehaven.git", tempDir)
	// if err != nil {
	// 	t.Fatalf("Failed to clone test repo: %v", err)
	// }

	// Initialize RepoMap and language map
	// rm := NewRepoMap()
	// languageMap := map[string]*tree_sitter.Language{
	// 	".go": tree_sitter.NewLanguage(treesitter_go.Language()),
	// 	".js": tree_sitter.NewLanguage(treesitter_javascript.Language()),
	// }
	//
	// // Test both traversal methods
	// err = traverseRepo("/home/rick/codehaven/", languageMap, rm)
	// if err != nil {
	// 	t.Fatalf("Error traversing repo with tree-sitter: %v", err)
	// }

	// Test LSP traversal with gopls on a single file
	rm2 := NewRepoMap()
	err = rm2.TraverseWithLSP("/home/rick/codehaven/", "gopls", "serve")
	if err != nil {
		t.Fatalf("Error traversing repo with LSP: %v", err)
	}

	rms := []*RepoMap{rm2}
	for _, rm := range rms {
		// Dump the repo map contents
		rm.Dump()

		ranked := rm.RankedFiles()
		fmt.Println("Files by rank:")
		for _, f := range ranked {
			fmt.Println(f)
		}
	}
	// // Print results for debugging
	// fmt.Printf("Classes: %+v\n", repoMap.Classes)
	// fmt.Printf("Functions: %+v\n", repoMap.Functions)
	//
	// // Add assertions based on expected results from the test repo
	// if len(repoMap.Functions) == 0 && len(repoMap.Classes) == 0 {
	// 	t.Error("Expected to find some functions or classes, but found none")
	// }
	//
	// // Check if the repository was properly initialized
	// wt, err := repo.Worktree()
	// if err != nil {
	// 	t.Errorf("Failed to get worktree: %v", err)
	// }
	//
	// status, err := wt.Status()
	// if err != nil {
	// 	t.Errorf("Failed to get status: %v", err)
	// }
	//
	// if len(status) == 0 {
	// 	t.Error("Expected to find tracked files, but found none")
	// }
}
