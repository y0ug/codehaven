package models

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	"github.com/y0ug/codehaven/internal/tokeniz"
	"github.com/y0ug/llmhaven/chat"
	"github.com/y0ug/llmhaven/modelinfo"
)

// Constants
const (
	DefaultModelName    = "gpt-4o"
	AnthropicBetaHeader = "prompt-caching-2024-07-31,pdfs-2024-09-25"
)

// ModelSettings represents configuration for a model
type ModelSettings struct {
	modelinfo.Model
	Name             string                 `json:"name"`
	EditFormat       string                 `json:"edit_format"`
	WeakModelName    *string                `json:"weak_model_name,omitempty"`
	UseRepoMap       bool                   `json:"use_repo_map"`
	SendUndoReply    bool                   `json:"send_undo_reply"`
	Lazy             bool                   `json:"lazy"`
	Reminder         string                 `json:"reminder"`
	ExamplesAsSysMsg bool                   `json:"examples_as_sys_msg"`
	ExtraParams      map[string]interface{} `json:"extra_params,omitempty"`
	CacheControl     bool                   `json:"cache_control"`
	CachesByDefault  bool                   `json:"caches_by_default"`
	UseSystemPrompt  bool                   `json:"use_system_prompt"`
	UseTemperature   bool                   `json:"use_temperature"`
	Streaming        bool                   `json:"streaming"`
	EditorModelName  *string                `json:"editor_model_name,omitempty"`
	EditorEditFormat *string                `json:"editor_edit_format,omitempty"`
}

// Model represents an AI model with its configuration and capabilities
type Model struct {
	ModelSettings
	MaxChatHistoryTokens int
	WeakModel            *Model
	EditorModel          *Model
	MissingKeys          []string
	KeysInEnvironment    bool
}

// NewModel creates a new Model instance
func NewModel(
	modelBase modelinfo.Model,
	weakModel *Model,
	editorModel *Model,
	editorEditFormat string,
) (*Model, error) {
	// Initialize model with default settings
	model := &Model{
		ModelSettings: ModelSettings{
			Model:           modelBase,
			Name:            modelBase.Name,
			EditFormat:      "whole",
			UseSystemPrompt: true,
			UseTemperature:  true,
			Streaming:       true,
		},
		MaxChatHistoryTokens: 1024,
		// metadata:             metadata,
	}

	// Configure model settings based on name
	if err := model.configureModelSettings(); err != nil {
		return nil, err
	}

	// Set up weak model if specified
	if weakModel != nil {
		model.WeakModel = weakModel
	}

	// Set up editor model if specified
	if editorModel != nil {
		model.EditorModel = editorModel
		if editorEditFormat != "" {
			model.EditorEditFormat = &editorEditFormat
		}
	}

	return model, nil
}

// Additional model-related constants and mappings
var (
	OpenAIModels = []string{
		"gpt-4",
		"gpt-4o",
		"gpt-4o-2024-05-13",
		"gpt-4-turbo-preview",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0314",
		"gpt-4-32k-0613",
		"gpt-4-turbo",
		"gpt-4-turbo-2024-04-09",
		"gpt-4-1106-preview",
		"gpt-4-0125-preview",
		"gpt-4-vision-preview",
		"gpt-4o-mini",
		"gpt-4o-mini-2024-07-18",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-0125",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-16k-0613",
	}

	AnthropicModels = []string{
		"claude-2",
		"claude-2.1",
		"claude-3-haiku-20240307",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-sonnet-20241022",
	}

	ModelAliases = map[string]string{
		"sonnet":   "claude-3-5-sonnet-20241022",
		"haiku":    "claude-3-5-haiku-20241022",
		"opus":     "claude-3-opus-20240229",
		"4":        "gpt-4-0613",
		"4o":       "gpt-4o",
		"4-turbo":  "gpt-4-1106-preview",
		"35turbo":  "gpt-3.5-turbo",
		"35-turbo": "gpt-3.5-turbo",
		"3":        "gpt-3.5-turbo",
		"deepseek": "deepseek/deepseek-chat",
		"r1":       "deepseek/deepseek-reasoner",
		"flash":    "gemini/gemini-2.0-flash-exp",
	}
)

func (m *Model) TokenCount(msg ...*chat.ChatMessage) (int, error) {
	encoding := tokeniz.GetEncodingByModelName(m.Name)
	if encoding == "" {
		// encoding not found for model
		encoding = tokeniz.GetEncodingByModelName("gpt-4o")
		if encoding == "" {
			return 0, fmt.Errorf("encoding not found for model: %s", m.Name)
		}
	}

	// Initialize tokenizer
	tkm, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return 0, fmt.Errorf("failed to get tokenizer: %w", err)
	}

	counter := tokeniz.TiktokenCounter(tkm)
	tokens, _, _ := tokeniz.CountMessage(counter, true, msg...)
	return tokens, nil
}

// We applying the OpenAI algorigthm for everybody that will just get us some overview
func (m *Model) TokenCountRequest(
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	countResponseTokens bool, // True if we include the response of the assistsant in the messages
) (int, error) {
	encoding := tokeniz.GetEncodingByModelName(m.Name)
	if encoding == "" {
		// encoding not found for model
		encoding = tokeniz.GetEncodingByModelName("gpt-4o")
		if encoding == "" {
			return 0, fmt.Errorf("encoding not found for model: %s", m.Name)
		}
	}

	// Initialize tokenizer
	tkm, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return 0, fmt.Errorf("failed to get tokenizer: %w", err)
	}

	counter := tokeniz.TiktokenCounter(tkm)
	tokens := tokeniz.TokenCounterOpenAI(counter, messages, tools, countResponseTokens)
	return tokens, nil
}

// ValidateEnvironment checks if required environment variables are set
func (m *Model) ValidateEnvironment() error {
	// Fast path for common models
	if missingKeys := m.FastValidateEnvironment(); missingKeys != nil {
		return fmt.Errorf("missing required environment variables: %v", missingKeys)
	}

	var requiredVars []string
	switch m.Provider {
	case "cohere_chat":
		requiredVars = []string{"COHERE_API_KEY"}
	case "gemini":
		requiredVars = []string{"GEMINI_API_KEY"}
	case "groq":
		requiredVars = []string{"GROQ_API_KEY"}
	}

	missingVars := m.validateVariables(requiredVars)
	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return nil
}

// FastValidateEnvironment performs quick validation for common models
func (m *Model) FastValidateEnvironment() []string {
	var requiredVar string

	if strings.HasPrefix(m.Name, "openai/") || contains(OpenAIModels, m.Name) {
		requiredVar = "OPENAI_API_KEY"
	} else if strings.HasPrefix(m.Name, "anthropic/") || contains(AnthropicModels, m.Name) {
		requiredVar = "ANTHROPIC_API_KEY"
	} else {
		return nil
	}

	if _, exists := os.LookupEnv(requiredVar); !exists {
		return []string{requiredVar}
	}

	return nil
}

// validateVariables checks if required environment variables are set
func (m *Model) validateVariables(vars []string) []string {
	var missingVars []string
	for _, v := range vars {
		if _, exists := os.LookupEnv(v); !exists {
			missingVars = append(missingVars, v)
		}
	}
	return missingVars
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (m *Model) configureModelSettings() error {
	// First try exact model match from default settings
	exactMatch := false
	for _, ms := range DefaultModelSettings {
		if m.Name == ms.Name {
			m.copyModelSettings(ms)
			exactMatch = true
			break
		}
	}

	// If no exact match found, apply generic settings based on model name
	if !exactMatch {
		m.applyGenericModelSettings()
	}

	// Apply extra params if they exist
	if err := m.applyExtraParams(); err != nil {
		return err
	}

	return nil
}

func (m *Model) copyModelSettings(ms ModelSettings) {
	m.EditFormat = ms.EditFormat
	m.WeakModelName = ms.WeakModelName
	m.UseRepoMap = ms.UseRepoMap
	m.SendUndoReply = ms.SendUndoReply
	m.Lazy = ms.Lazy
	m.Reminder = ms.Reminder
	m.ExamplesAsSysMsg = ms.ExamplesAsSysMsg
	m.ExtraParams = ms.ExtraParams
	m.CacheControl = ms.CacheControl
	m.CachesByDefault = ms.CachesByDefault
	m.UseSystemPrompt = ms.UseSystemPrompt
	m.UseTemperature = ms.UseTemperature
	m.Streaming = ms.Streaming
	m.EditorModelName = ms.EditorModelName
	m.EditorEditFormat = ms.EditorEditFormat
}

// applyGenericModelSettings applies settings based on model name patterns
func (m *Model) applyGenericModelSettings() {
	modelName := strings.ToLower(m.Name)

	switch {
	case strings.Contains(modelName, "llama3") || strings.Contains(modelName, "llama-3"):
		if strings.Contains(modelName, "70b") {
			m.EditFormat = "diff"
			m.UseRepoMap = true
			m.SendUndoReply = true
			m.ExamplesAsSysMsg = true
		}

	case strings.Contains(modelName, "gpt-4-turbo") ||
		(strings.Contains(modelName, "gpt-4-") && strings.Contains(modelName, "-preview")):
		m.EditFormat = "udiff"
		m.UseRepoMap = true
		m.SendUndoReply = true

	case strings.Contains(modelName, "gpt-4") || strings.Contains(modelName, "claude-3-opus"):
		m.EditFormat = "diff"
		m.UseRepoMap = true
		m.SendUndoReply = true

	case strings.Contains(modelName, "gpt-3.5") || strings.Contains(modelName, "gpt-4"):
		m.Reminder = "sys"

	case strings.Contains(modelName, "3.5-sonnet") || strings.Contains(modelName, "3-5-sonnet"):
		m.EditFormat = "diff"
		m.UseRepoMap = true
		m.ExamplesAsSysMsg = true
		m.Reminder = "user"

	case strings.HasPrefix(modelName, "o1-") || strings.Contains(modelName, "/o1-"):
		m.UseSystemPrompt = false
		m.UseTemperature = false

	case strings.Contains(modelName, "qwen") &&
		strings.Contains(modelName, "coder") &&
		(strings.Contains(modelName, "2.5") || strings.Contains(modelName, "2-5")) &&
		strings.Contains(modelName, "32b"):
		m.EditFormat = "diff"
		m.EditorEditFormat = ptrStr("editor-diff")
		m.UseRepoMap = true
		if strings.HasPrefix(modelName, "ollama/") || strings.HasPrefix(modelName, "ollama_chat/") {
			m.ExtraParams = map[string]interface{}{
				"num_ctx": 8 * 1024,
			}
		}
	}

	// If edit_format is "diff", set use_repo_map to true by default
	if m.EditFormat == "diff" {
		m.UseRepoMap = true
	}
}

// applyExtraParams applies any additional parameters from extra model settings
func (m *Model) applyExtraParams() error {
	// Find extra settings
	var extraSettings *ModelSettings
	for _, ms := range DefaultModelSettings {
		if ms.Name == "aider/extra_params" {
			extraSettings = &ms
			break
		}
	}

	if extraSettings != nil && extraSettings.ExtraParams != nil {
		// Initialize extra_params if it doesn't exist
		if m.ExtraParams == nil {
			m.ExtraParams = make(map[string]interface{})
		}

		// Deep merge the extra_params maps
		if err := m.mergeExtraParams(extraSettings.ExtraParams); err != nil {
			return fmt.Errorf("failed to merge extra params: %w", err)
		}
	}

	return nil
}

// mergeExtraParams performs a deep merge of extra parameters
func (m *Model) mergeExtraParams(params map[string]interface{}) error {
	for key, value := range params {
		switch v := value.(type) {
		case map[string]interface{}:
			// If the existing value is also a map, merge recursively
			if existing, ok := m.ExtraParams[key].(map[string]interface{}); ok {
				newMap := make(map[string]interface{})
				for k, v := range existing {
					newMap[k] = v
				}
				for k, v := range v {
					newMap[k] = v
				}
				m.ExtraParams[key] = newMap
			} else {
				m.ExtraParams[key] = v
			}
		default:
			// For non-map values, simply update
			m.ExtraParams[key] = v
		}
	}
	return nil
}
