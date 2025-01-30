package consolecoder

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/c-bata/go-prompt"
)

func (c *Console) completer(d prompt.Document) []prompt.Suggest {
	var suggestions []prompt.Suggest
	input := d.TextBeforeCursor()
	words := strings.Fields(input)

	if len(words) <= 1 {
		// If the word starts with /, suggest commands
		word := d.GetWordBeforeCursor()
		if strings.HasPrefix(word, "/") {
			for cmdName, cmd := range c.commands {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        cmdName,
					Description: cmd.description,
				})
			}
			return prompt.FilterHasPrefix(suggestions, word, true)
		}
	} else if words[0] == "/add" {
		// Get the word being typed
		word := d.GetWordBeforeCursor()
		// If word is empty, suggest current directory
		if word == "" {
			word = "."
		}
		return c.getFileSuggestions(word)
	} else if words[0] == "/remove" {
		// Get the word being typed
		word := d.GetWordBeforeCursor()
		// If word is empty, suggest current directory
		if word == "" {
			word = "."
		}
		return c.getFileManagerSuggestions(word)
	} else if words[0] == "/confirm" {
		// Get the word being typed
		word := d.GetWordBeforeCursor()
		// If word is empty, suggest current directory
		if word == "" {
			word = "."
		}
		return c.getConfirmSuggestions(word)
	}

	return suggestions
}

func (c *Console) getFileSuggestions(pattern string) []prompt.Suggest {
	var suggestions []prompt.Suggest
	matches, err := filepath.Glob(pattern + "*")
	if err != nil {
		return suggestions
	}

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		description := "file"
		if info.IsDir() {
			description = "directory"
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        match,
			Description: description,
		})
	}
	return suggestions
}

func (c *Console) getConfirmSuggestions(pattern string) []prompt.Suggest {
	confirmations := c.confirmationManager.GetPendingConfirmations()
	pattern = strings.ToLower(pattern)

	suggestions := make([]prompt.Suggest, 0)
	seen := make(map[string]bool)

	for _, confirmation := range confirmations {
		basename := confirmation.ID

		// Skip if we've already seen this basename
		if seen[basename] {
			continue
		}
		seen[basename] = true

		// Check if pattern matches anywhere in the basename
		if strings.Contains(basename, pattern) {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        basename,
				Description: basename,
			})
		}
	}

	// Sort suggestions alphabetically
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

func (c *Console) getFileManagerSuggestions(pattern string) []prompt.Suggest {
	filesInCtx := c.coder.GetRM().ListFiles(0)
	pattern = strings.ToLower(pattern)

	suggestions := make([]prompt.Suggest, 0)
	seen := make(map[string]bool)

	for filename := range filesInCtx {
		basename := filepath.Base(filename)
		lowerBasename := strings.ToLower(basename)

		// Skip if we've already seen this basename
		if seen[basename] {
			continue
		}
		seen[basename] = true

		// Check if pattern matches anywhere in the basename
		if strings.Contains(lowerBasename, pattern) {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        basename,
				Description: filename,
			})
		}
	}

	// Sort suggestions alphabetically
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}
