package highlighter

import (
	"bytes"
	"context"
	"io"
	"log"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter handles syntax highlighting for markdown and code blocks
type Highlighter struct {
	w            io.Writer
	inCodeBlock  bool
	currentLang  string
	lexer        chroma.Lexer
	defaultLexer chroma.Lexer
	formatter    chroma.Formatter
	style        *chroma.Style
	lineBuffer   bytes.Buffer
}

// NewHighlighter creates a new instance of Highlighter
func NewHighlighter(writer io.Writer) *Highlighter {
	// lexer := lexers.Get("markdown")
	// if lexer == nil {
	// 	lexer = lexers.Fallback
	// }
	lexer := lexers.Fallback

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	return &Highlighter{
		w:            writer,
		lexer:        lexer,
		defaultLexer: lexer,
		formatter:    formatter,
		style:        style,
	}
}

// ProcessLine handles the highlighting of a single line of text
func (h *Highlighter) ProcessLine(line string) {
	if strings.HasPrefix(line, "```") {
		h.handleCodeBlockMarker(line)
		return
	}
	h.highlightAndPrint(line)
}

// handleCodeBlockMarker processes the start/end of code blocks
func (h *Highlighter) handleCodeBlockMarker(line string) {
	h.inCodeBlock = !h.inCodeBlock
	if h.inCodeBlock {
		line = strings.ToLower(line)
		h.highlightAndPrint(line)
		h.currentLang = strings.Trim(strings.ToLower(line[3:]), "\n")
		h.lexer = lexers.Get(h.currentLang)
		if h.lexer == nil {
			h.lexer = h.defaultLexer
		}
	} else {
		h.currentLang = ""
		h.lexer = h.defaultLexer
		h.highlightAndPrint(line)
	}
}

// GetCurrentLanguage returns the current language being highlighted
func (h *Highlighter) GetCurrentLanguage() string {
	return h.currentLang
}

// IsInCodeBlock returns whether currently processing a code block
func (h *Highlighter) IsInCodeBlock() bool {
	return h.inCodeBlock
}

// highlightAndPrint performs the actual syntax highlighting
func (h *Highlighter) highlightAndPrint(line string) {
	iterator, err := h.lexer.Tokenise(nil, line)
	if err != nil {
		log.Printf("Tokenization error: %v", err)
		return
	}

	err = h.formatter.Format(h.w, h.style, iterator)
	if err != nil {
		log.Printf("Formatting error: %v", err)
		return
	}
}

// highlightCodeBlock is just a helper function that applies
// syntax highlighting to a complete code block in one shot.
func (h *Highlighter) highlightCodeBlock(code string, language string) {
	// Save the original lexer
	origLexer := h.lexer

	// Try to get a lexer based on the language hint (if any).
	if language != "" {
		if l := lexers.Get(language); l != nil {
			h.lexer = l
		} else {
			h.lexer = h.defaultLexer
		}
	} else {
		// No language specified, just fall back to default.
		h.lexer = h.defaultLexer
	}

	iterator, err := h.lexer.Tokenise(nil, code)
	if err != nil {
		log.Printf("Tokenization error: %v", err)
		return
	}

	err = h.formatter.Format(h.w, h.style, iterator)
	if err != nil {
		log.Printf("Formatting error: %v", err)
		return
	}

	// Restore the original lexer
	h.lexer = origLexer
}

func (h *Highlighter) Write(p []byte) (int, error) {
	// Add this chunk to our line buffer.
	h.lineBuffer.Write(p)

	// Extract as many complete lines as possible.
	for {
		bufStr := h.lineBuffer.String()

		newlineIdx := strings.Index(bufStr, "\n")
		if strings.HasPrefix(bufStr, "```") && newlineIdx != -1 {
			line := bufStr[:newlineIdx+1]
			// fmt.Println("line", line)
			h.ProcessLine(line)
			h.lineBuffer.Next(newlineIdx + 1)
			bufStr = h.lineBuffer.String()
		}

		newlineIdx = strings.Index(bufStr, "\n")
		if h.inCodeBlock {
			if newlineIdx == -1 {
				break
			}
			// We have at least one full line ending in '\n'.
			line := bufStr[:newlineIdx+1]
			// Advance our buffer past that line + newline
			h.lineBuffer.Next(newlineIdx + 1)
			h.ProcessLine(line)
			continue
		}

		if strings.Contains(bufStr, "`") && newlineIdx == -1 {
			// contains ` but no newline we need to buffer
			break
		}
		if newlineIdx == -1 {
			// not new line no code block no ``` detected we print`
			data := h.lineBuffer.String()
			h.lineBuffer.Next(len(data) + 1)
			h.ProcessLine(data)
			break
		}
		// we process the line
		line := bufStr[:newlineIdx+1]
		h.ProcessLine(line)
		h.lineBuffer.Next(newlineIdx + 1)
	}
	return len(p), nil
}

// ProcessStream processes a stream of text from a channel and highlights it
func (h *Highlighter) ProcessStream(ctx context.Context, ch <-chan string) error {
	var lineBuffer bytes.Buffer // accumulates chunks until we have at least one full line
	for {
		select {
		case <-ctx.Done():
			// Context canceled or timed out.
			return ctx.Err()

		case chunk, ok := <-ch:
			if !ok {
				// Channel closed.
				// If we're in a code block and never saw the closing fence,
				// highlight whatever we have anyway (best effort).

				return nil
			}

			// Add this chunk to our line buffer.
			lineBuffer.WriteString(chunk)

			// Extract as many complete lines as possible.
			for {
				bufStr := lineBuffer.String()

				newlineIdx := strings.Index(bufStr, "\n")
				if strings.HasPrefix(bufStr, "```") && newlineIdx != -1 {
					line := bufStr[:newlineIdx+1]
					// fmt.Println("line", line)
					h.ProcessLine(line)
					lineBuffer.Next(newlineIdx + 1)
					bufStr = lineBuffer.String()
				}

				newlineIdx = strings.Index(bufStr, "\n")
				if h.inCodeBlock {
					if newlineIdx == -1 {
						break
					}
					// We have at least one full line ending in '\n'.
					line := bufStr[:newlineIdx+1]
					// Advance our buffer past that line + newline
					lineBuffer.Next(newlineIdx + 1)
					h.ProcessLine(line)
					continue
				}

				if strings.Contains(bufStr, "`") && newlineIdx == -1 {
					// contains ` but no newline we need to buffer
					break
				}
				if newlineIdx == -1 {
					// not new line no code block no ``` detected we print`
					data := lineBuffer.String()
					lineBuffer.Next(len(data) + 1)
					h.ProcessLine(data)
					break
				}
				// we process the line
				line := bufStr[:newlineIdx+1]
				h.ProcessLine(line)
				lineBuffer.Next(newlineIdx + 1)

			}
		}
	}
}

func (h *Highlighter) Flush() {
	leftover := h.lineBuffer.String()
	if len(leftover) > 0 {
		// Process whatever remains, for a best-effort highlight
		h.ProcessLine(leftover)
		h.lineBuffer.Reset()
	}
}

// ProcessStream processes a stream of text from a channel
// and push it as stream of line
// ProcessStreamToNewLine processes a stream of text and splits it into lines
// sending each complete line to the output channel
func ProcessStreamToNewLine(ctx context.Context, in <-chan string, out chan<- string) error {
	defer close(out)

	var buffer bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case content, ok := <-in:
			if !ok {
				// Process remaining content before returning
				remaining := buffer.String()
				if len(remaining) > 0 {
					if !strings.HasSuffix(remaining, "\n") {
						remaining += "\n"
					}
					out <- remaining
				}
				return nil
			}

			buffer.WriteString(content)
			for {
				currentBuffer := buffer.String()
				index := strings.Index(currentBuffer, "\n")
				if index == -1 {
					break
				}
				line := currentBuffer[:index+1]
				buffer.Next(index + 1)
				out <- line
			}
		}
	}
}
