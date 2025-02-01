package repomap

import (
	"context"
	"embed"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/y0ug/codehaven/internal/repomap/lsp"
)

//go:embed queries
var queriesFS embed.FS

type RepoMap struct {
	defines    map[string]map[string]bool // symbol -> set of files
	references map[string][]string        // symbol -> list of referencer files
}

// Helper constructor
func NewRepoMap() *RepoMap {
	return &RepoMap{
		defines:    make(map[string]map[string]bool),
		references: make(map[string][]string),
	}
}

type CaptureResult struct {
	CaptureName string
	Text        string
	// Optionally add line number, kind, etc.
}

func executeQuery(
	sourceCode []byte,
	lang *tree_sitter.Language,
	queryStr string,
) ([]CaptureResult, error) {
	parser := tree_sitter.NewParser()
	parser.SetLanguage(lang)
	defer parser.Close()

	tree := parser.Parse(sourceCode, nil)
	defer tree.Close()

	query, err := tree_sitter.NewQuery(lang, queryStr)
	if err != nil {
		return nil, err
	}
	defer query.Close()

	qc := tree_sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Matches(query, tree.RootNode(), sourceCode)

	var results []CaptureResult

	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			captureName := query.CaptureNames()[capture.Index]
			text := string(capture.Node.Utf8Text(sourceCode))
			results = append(results, CaptureResult{
				CaptureName: captureName,
				Text:        text,
			})
		}
	}
	return results, nil
}

func traverseRepo(
	root string,
	languageMap map[string]*tree_sitter.Language,
	repoMap *RepoMap,
) error {
	// Open the repository
	r, err := git.PlainOpen(root)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get the head
	ref, err := r.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	tree.Files().ForEach(func(f *object.File) error {
		// fmt.Printf("100644 blob %s    %s\n", f.Hash, f.Name)
		relPath := filepath.Join(root, f.Name)
		content, err := os.ReadFile(relPath)
		if err != nil {
			log.Printf("Failed to read file %s: %v", f.Name, err)
			return nil
		}
		processFile(f.Name, languageMap, content, repoMap)
		return nil
	})

	return nil
}

func traversePath(
	root string,
	languageMap map[string]*tree_sitter.Language,
	repoMap *RepoMap,
) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			return nil
		}
		processFile(path, languageMap, content, repoMap)
		return nil
	})
}

func processFile(
	path string,
	languageMap map[string]*tree_sitter.Language,
	content []byte,
	repoMap *RepoMap,
) {
	ext := filepath.Ext(path)
	lang, exists := languageMap[ext]
	if !exists {
		return
	}
	queryFile := ""
	switch ext {
	case ".go":
		queryFile = "queries/tree-sitter-go-tags.scm"
	default:
		return
	}

	queryBytes, err := queriesFS.ReadFile(queryFile)
	if err != nil {
		log.Printf("Failed to read query file %s: %v", queryFile, err)
		return
	}
	queryStr := string(queryBytes)

	results, err := executeQuery(content, lang, queryStr)
	if err != nil {
		log.Printf("Query execution failed for file %s: %v", path, err)
		return
	}

	for _, res := range results {
		switch res.CaptureName {
		// For Go, e.g. "name.definition.function"
		case "name.definition.function", "name.definition.method", "name.definition.type":
			symbol := res.Text
			if repoMap.defines[symbol] == nil {
				repoMap.defines[symbol] = make(map[string]bool)
			}
			repoMap.defines[symbol][path] = true

		// For Go, e.g. "name.reference.call"
		case "name.reference.call", "name.reference.type":
			symbol := res.Text
			repoMap.references[symbol] = append(repoMap.references[symbol], path)
		}
	}
}

func (rm *RepoMap) BuildGraph() map[string]map[string]float64 {
	graph := make(map[string]map[string]float64)

	for symbol, definersMap := range rm.defines {
		// definersMap is map[fileName]bool
		// references is []string
		referencers := rm.references[symbol]

		// Convert definersMap into a slice
		var definers []string
		for f := range definersMap {
			definers = append(definers, f)
		}

		// Count how many times each file references this symbol
		freqCount := make(map[string]int)
		for _, refFile := range referencers {
			freqCount[refFile]++
		}

		// Create edges refFile -> defFile with some weighting
		for refFile, count := range freqCount {
			// sqrt is optional, just to reduce large counts
			edgeWeight := math.Sqrt(float64(count))

			for _, defFile := range definers {
				// skip self-edges if you want
				if refFile == defFile {
					continue
				}
				if graph[refFile] == nil {
					graph[refFile] = make(map[string]float64)
				}
				graph[refFile][defFile] += edgeWeight
			}
		}
	}

	return graph
}

func pageRank(
	graph map[string]map[string]float64,
	damping float64,
	maxIter int,
	tol float64,
) map[string]float64 {
	// 1) gather all nodes
	nodes := make([]string, 0, len(graph))
	for n := range graph {
		nodes = append(nodes, n)
	}
	// also gather nodes that appear only as targets
	seenTargets := make(map[string]bool)
	for _, edges := range graph {
		for target := range edges {
			seenTargets[target] = true
		}
	}
	for t := range seenTargets {
		if _, ok := graph[t]; !ok {
			graph[t] = make(map[string]float64)
			nodes = append(nodes, t)
		}
	}

	n := float64(len(nodes))
	rank := make(map[string]float64)
	for _, node := range nodes {
		rank[node] = 1.0 / n
	}

	// 2) precompute out-sum
	outSum := make(map[string]float64)
	for src, edges := range graph {
		var sum float64
		for _, w := range edges {
			sum += w
		}
		outSum[src] = sum
	}

	// 3) iterate
	for i := 0; i < maxIter; i++ {
		diff := 0.0
		newRank := make(map[string]float64)

		base := (1 - damping) / n
		for _, node := range nodes {
			newRank[node] = base
		}

		for src, edges := range graph {
			if len(edges) == 0 {
				// "sink"
				for _, node := range nodes {
					newRank[node] += damping * (rank[src] / n)
				}
			} else {
				for dst, w := range edges {
					if outSum[src] > 0 {
						share := damping * (w / outSum[src]) * rank[src]
						newRank[dst] += share
					}
				}
			}
		}

		// measure difference
		for _, node := range nodes {
			diff += math.Abs(newRank[node] - rank[node])
		}
		rank = newRank

		// if close to convergence, stop
		if diff < tol {
			break
		}
	}

	return rank
}

// Dump prints the contents of the RepoMap for debugging purposes
func (rm *RepoMap) Dump() {
	fmt.Println("=== RepoMap Dump ===")
	
	fmt.Println("\nDefinitions:")
	for symbol, files := range rm.defines {
		fmt.Printf("%s defined in:\n", symbol)
		for file := range files {
			fmt.Printf("  - %s\n", file)
		}
	}

	fmt.Println("\nReferences:")
	for symbol, files := range rm.references {
		fmt.Printf("%s referenced in:\n", symbol)
		// Count references per file
		refCount := make(map[string]int)
		for _, file := range files {
			refCount[file]++
		}
		for file, count := range refCount {
			fmt.Printf("  - %s (%d times)\n", file, count)
		}
	}
	fmt.Println("==================")
}

// TraverseWithLSP uses a language server to analyze the codebase
func (rm *RepoMap) TraverseWithLSP(root string, serverCmd string, args ...string) error {
	cmd := exec.Command(serverCmd, args...)
	
	client, err := lsp.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create LSP client: %w", err)
	}

	ctx := context.Background()
	rootURI := "file://" + root

	if err := client.Initialize(ctx, rootURI); err != nil {
		return fmt.Errorf("failed to initialize LSP server: %w", err)
	}
	defer client.Shutdown(ctx)

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Filter files based on extension if needed
		ext := filepath.Ext(path)
		if ext != ".go" { // Add more extensions as needed
			return nil
		}

		uri := "file://" + path
		symbols, err := client.DocumentSymbols(ctx, uri)
		if err != nil {
			log.Printf("Failed to get symbols for %s: %v", path, err)
			return nil
		}

		for _, symbol := range symbols {
			switch symbol.Kind {
			case 12: // Function
				fallthrough
			case 6:  // Method
				fallthrough
			case 5:  // Class/Interface
				if rm.defines[symbol.Name] == nil {
					rm.defines[symbol.Name] = make(map[string]bool)
				}
				rm.defines[symbol.Name][path] = true

				// Get references for this symbol
				refs, err := client.References(ctx, uri, 
					symbol.Location.Range.Start.Line,
					symbol.Location.Range.Start.Character)
				if err != nil {
					if !strings.Contains(err.Error(), "no identifier found") {
						log.Printf("Failed to get references for %s: %v", symbol.Name, err)
					}
					continue
				}
				// fmt.Printf("refs: %d",
				// Add references to the map
				for _, ref := range refs {
					// Convert URI to file path
					refPath := ref.URI
					if len(refPath) > 7 && refPath[:7] == "file://" {
						refPath = refPath[7:]
					}
					// Skip self-references in the same file
					if refPath != path {
						rm.references[symbol.Name] = append(rm.references[symbol.Name], refPath)
					}
				}
			}
		}

		return nil
	})

	return err
}

func (rm *RepoMap) RankedFiles() []string {
	graph := rm.BuildGraph()
	ranks := pageRank(graph, 0.85, 100, 1e-6)

	// Convert to slice for sorting
	type fs struct {
		file  string
		score float64
	}
	var results []fs
	for file, score := range ranks {
		results = append(results, fs{file, score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Just return file names in descending rank order
	rankedFiles := make([]string, len(results))
	for i, fs := range results {
		rankedFiles[i] = fs.file
		// optionally print them
		fmt.Printf("%s => %.3f\n", fs.file, fs.score)
	}
	return rankedFiles
}
