package extractors

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/llmhaven/chat"
)

type BlockExtractor struct {
	fence  Fence
	logger *slog.Logger
	format ExtractorType
	name   string
}

const (
	searchMarker  = "<<<<<<< SEARCH"
	divider       = "======="
	replaceMarker = ">>>>>>> REPLACE"
)

var (
	headRe    = regexp.MustCompile(`^<{5,9} SEARCH\s*$`)
	dividerRe = regexp.MustCompile(`^={5,9}\s*$`)
	replaceRe = regexp.MustCompile(`^>{5,9} REPLACE\s*$`)
)

func NewBlockExtractor(
	logger *slog.Logger,
	format ExtractorType,
	fence Fence,
) *BlockExtractor {
	return &BlockExtractor{
		fence:  fence,
		logger: logger,
		format: format,
	}
}

func (c *BlockExtractor) Name() string {
	return "BlockExtractor"
}

func (c *BlockExtractor) SupportedActions() []actions.ActionType {
	return []actions.ActionType{"apply_edit", "shell_command"}
}

func (c *BlockExtractor) Type() ExtractorType {
	return c.format
}

func (c *BlockExtractor) GetChatTools() []chat.Tool {
	return nil
}

func (c *BlockExtractor) SetFence(fence Fence) {
	c.fence = fence
}

func (c *BlockExtractor) Extract(
	msg *chat.ChatMessage,
) ([]actions.Action, error) {
	results := make([]actions.Action, 0)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeText {
			actions := c.getEdits(content.String())
			results = append(results, actions...)
		}
	}
	return results, nil
}

func (c *BlockExtractor) getEdits(content string) []actions.Action {
	var results []actions.Action
	lines := strings.Split(content, "\n")
	i := 0
	var parentAction *actions.Action

	// Collect all batchEdits into a slice
	var batchEdits []actions.BatchEdit

	for i < len(lines) {
		line := lines[i]
		if headRe.MatchString(line) {
			action, newI, err := c.extractEditBlock(lines, i)
			if err == nil {
				batchEdits = append(batchEdits, actions.BatchEdit{
					Edit: action.Payload.(actions.FileEditAction),
				})
			}
			i = newI
			continue
		}

		if isShellBlockStart(line) {
			cmd, newI := extractShellCommand(lines, i)
			action := actions.NewShellCommand(cmd, true).WithParent(parentAction)
			if parentAction == nil {
				parentAction = &action
			}
			results = append(results, action)
			i = newI
			continue
		}
		i++
	}

	// If there are edits, create a single action for the batch and append at the begining
	// of  results. This is to ensure that we run the batch edit before any other actions.
	// but maybe we should not batctedit and just executing everything as it come from the
	// LLM and add the commit here at then end of all the actions
	if len(batchEdits) > 0 {
		batchAction := actions.NewBatchEditAction(batchEdits).WithParent(parentAction)
		results = append([]actions.Action{batchAction}, results...)
	}

	return results
}

func (c *BlockExtractor) extractEditBlock(
	lines []string,
	start int,
) (actions.Action, int, error) {
	filename := c.findFilename(lines, start)
	if filename == "" {
		return actions.Action{}, start, fmt.Errorf("filename not found")
	}

	var original, updated []string
	i := start + 1
	for ; i < len(lines) && !dividerRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		original = append(original, lines[i])
	}

	if i >= len(lines) {
		return actions.Action{}, i, fmt.Errorf("missing divider")
	}
	i++ // Skip divider

	for ; i < len(lines) && !replaceRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		updated = append(updated, lines[i])
	}

	if i >= len(lines) {
		return actions.Action{}, i, fmt.Errorf("missing replace marker")
	}
	i++ // Skip replace marker

	originalStr := strings.Join(original, "\n")
	updatedStr := strings.Join(updated, "\n")
	return actions.NewApplyEdit(filename, originalStr, updatedStr), i, nil
}

func (c *BlockExtractor) findFilename(lines []string, current int) string {
	codeBlockLanguages := map[string]bool{
		"python": true, "bash": true, "sh": true,
		"javascript": true, "go": true, "typescript": true,
		"c": true, "cpp": true, "js": true, "ts": true,
	}

	for i := current - 1; i >= 0 && i >= current-3; i-- {
		line := strings.TrimSpace(lines[i])

		if line == "" || strings.HasPrefix(line, c.fence[0]) {
			continue
		}

		if _, isLang := codeBlockLanguages[strings.ToLower(line)]; isLang {
			continue
		}

		return strings.Trim(line, "`'\"")
	}
	return ""
}
