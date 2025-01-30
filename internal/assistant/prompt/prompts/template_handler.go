package prompts

//go:generate go run ./cmd/genprompts/main.go

import (
	"bytes"
	"log/slog"
	"text/template"

	"github.com/y0ug/codehaven/internal/assistant/settings"
)

type TemplateHandler struct {
	prompts  Prompter
	data     TemplateData
	settings *settings.CoderSettings
	logger   *slog.Logger
}

func NewTemplateHandler(
	prompts Prompter,
	settings *settings.CoderSettings,
	logger *slog.Logger,
) *TemplateHandler {
	th := &TemplateHandler{
		prompts:  prompts,
		settings: settings,
		logger:   logger,
		data:     make(TemplateData),
	}

	th.UpdateData()
	return th
}

func (c *TemplateHandler) UpdateData() {
	for key, value := range c.settings.GetTemplateData() {
		c.Set(key, value)
	}
	c.Set("LazyPrompt", c.getLazyPrompt())
	c.Set("ShellCmdPrompt", c.Render(c.prompts.GetShellCmdPrompt()))
	c.Set("ShellCmdReminder", c.Render(c.prompts.GetShellCmdReminder()))
}

func (c *TemplateHandler) getLazyPrompt() string {
	if c.settings.IsLazyModel() {
		return c.Render(c.prompts.GetLazyPrompt())
	}
	return ""
}

func (th *TemplateHandler) Set(key string, value interface{}) {
	th.data[key] = value
}

func (th *TemplateHandler) RenderData(templateText string, data map[string]interface{}) string {
	for key, value := range th.data {
		if _, ok := data[key]; !ok {
			data[key] = value
		}
	}
	tmpl, err := template.New("prompt").Parse(templateText)
	if err != nil {
		th.logger.Error("Failed to parse template", "error", err)
		return templateText
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		th.logger.Error("Failed to render template", "error", err)
		return templateText
	}

	return buf.String()
}

func (th *TemplateHandler) Render(templateText string) string {
	return th.RenderData(templateText, make(map[string]interface{}))
}

// TemplateData type changed to map for flexibility
type TemplateData map[string]interface{}
