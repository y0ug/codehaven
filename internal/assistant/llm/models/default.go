package models

// DefaultModelSettings defines all available model settings
var DefaultModelSettings = []ModelSettings{
	// GPT-3.5 Models
	{
		Name:          "gpt-3.5-turbo",
		EditFormat:    "whole",
		WeakModelName: ptrStr("gpt-4o-mini"),
		Reminder:      "sys",
	},
	{
		Name:          "gpt-3.5-turbo-0125",
		EditFormat:    "whole",
		WeakModelName: ptrStr("gpt-4o-mini"),
		Reminder:      "sys",
	},
	{
		Name:          "gpt-3.5-turbo-1106",
		EditFormat:    "whole",
		WeakModelName: ptrStr("gpt-4o-mini"),
		Reminder:      "sys",
	},
	{
		Name:          "gpt-3.5-turbo-0613",
		EditFormat:    "whole",
		WeakModelName: ptrStr("gpt-4o-mini"),
		Reminder:      "sys",
	},
	{
		Name:          "gpt-3.5-turbo-16k-0613",
		EditFormat:    "whole",
		WeakModelName: ptrStr("gpt-4o-mini"),
		Reminder:      "sys",
	},

	// GPT-4 Models
	{
		Name:          "gpt-4-turbo-2024-04-09",
		EditFormat:    "udiff",
		WeakModelName: ptrStr("gpt-4o-mini"),
		UseRepoMap:    true,
		Lazy:          true,
		Reminder:      "sys",
	},
	{
		Name:          "gpt-4-turbo",
		EditFormat:    "udiff",
		WeakModelName: ptrStr("gpt-4o-mini"),
		UseRepoMap:    true,
		Lazy:          true,
		Reminder:      "sys",
	},
	{
		Name:             "openai/gpt-4o",
		EditFormat:       "diff",
		WeakModelName:    ptrStr("gpt-4o-mini"),
		UseRepoMap:       true,
		Lazy:             true,
		Reminder:         "sys",
		EditorEditFormat: ptrStr("editor-diff"),
		ExamplesAsSysMsg: true,
	},
	{
		Name:             "gpt-4o",
		EditFormat:       "diff",
		WeakModelName:    ptrStr("gpt-4o-mini"),
		UseRepoMap:       true,
		Lazy:             true,
		Reminder:         "sys",
		EditorEditFormat: ptrStr("editor-diff"),
		ExamplesAsSysMsg: true,
	},

	// Claude Models
	{
		Name:          "claude-3-opus-20240229",
		EditFormat:    "diff",
		WeakModelName: ptrStr("claude-3-5-haiku-20241022"),
		UseRepoMap:    true,
	},
	{
		Name:          "claude-3-sonnet-20240229",
		EditFormat:    "whole",
		WeakModelName: ptrStr("claude-3-5-haiku-20241022"),
	},
	{
		Name:             "claude-3-5-sonnet-20240620",
		EditFormat:       "diff",
		WeakModelName:    ptrStr("claude-3-5-haiku-20241022"),
		EditorModelName:  ptrStr("claude-3-5-sonnet-20240620"),
		EditorEditFormat: ptrStr("editor-diff"),
		UseRepoMap:       true,
		ExamplesAsSysMsg: true,
		CacheControl:     true,
		Reminder:         "user",
		ExtraParams: map[string]interface{}{
			"extra_headers": map[string]string{
				"anthropic-beta": AnthropicBetaHeader,
			},
			"max_tokens": 8192,
		},
	},
	{
		Name:             "claude-3-5-sonnet-20241022",
		EditFormat:       "diff",
		WeakModelName:    ptrStr("claude-3-5-haiku-20241022"),
		EditorModelName:  ptrStr("claude-3-5-sonnet-20241022"),
		EditorEditFormat: ptrStr("editor-diff"),
		UseRepoMap:       true,
		ExamplesAsSysMsg: true,
		CacheControl:     true,
		Streaming:        true,
		UseSystemPrompt:  true,
		Reminder:         "user",
		ExtraParams: map[string]interface{}{
			"extra_headers": map[string]string{
				"anthropic-beta": AnthropicBetaHeader,
			},
			"max_tokens": 8192,
		},
	},

	// Haiku Models
	{
		Name:             "claude-3-haiku-20240307",
		EditFormat:       "whole",
		WeakModelName:    ptrStr("claude-3-haiku-20240307"),
		ExamplesAsSysMsg: true,
		CacheControl:     true,
		ExtraParams: map[string]interface{}{
			"extra_headers": map[string]string{
				"anthropic-beta": AnthropicBetaHeader,
			},
		},
	},
	{
		Name:          "claude-3-5-haiku-20241022",
		EditFormat:    "diff",
		WeakModelName: ptrStr("claude-3-5-haiku-20241022"),
		UseRepoMap:    true,
		CacheControl:  true,
		ExtraParams: map[string]interface{}{
			"extra_headers": map[string]string{
				"anthropic-beta": AnthropicBetaHeader,
			},
		},
	},

	// Vertex AI Models
	{
		Name:             "vertex_ai/claude-3-5-sonnet@20240620",
		EditFormat:       "diff",
		WeakModelName:    ptrStr("vertex_ai/claude-3-5-haiku@20241022"),
		EditorModelName:  ptrStr("vertex_ai/claude-3-5-sonnet@20240620"),
		EditorEditFormat: ptrStr("editor-diff"),
		UseRepoMap:       true,
		ExamplesAsSysMsg: true,
		Reminder:         "user",
		ExtraParams: map[string]interface{}{
			"max_tokens": 8192,
		},
	},

	// Cohere Models
	{
		Name:          "command-r-plus",
		EditFormat:    "whole",
		WeakModelName: ptrStr("command-r-plus"),
		UseRepoMap:    true,
	},
	{
		Name:          "command-r-08-2024",
		EditFormat:    "whole",
		WeakModelName: ptrStr("command-r-08-2024"),
		UseRepoMap:    true,
	},

	// Gemini Models
	{
		Name:       "gemini/gemini-1.5-pro-002",
		EditFormat: "diff",
		UseRepoMap: true,
	},
	{
		Name:       "gemini/gemini-1.5-flash-002",
		EditFormat: "whole",
	},
	{
		Name:       "gemini/gemini-1.5-pro",
		EditFormat: "diff-fenced",
		UseRepoMap: true,
	},

	// DeepSeek Models
	{
		Name:             "deepseek/deepseek-chat",
		EditFormat:       "diff",
		UseRepoMap:       true,
		ExamplesAsSysMsg: true,
		Reminder:         "sys",
		CachesByDefault:  true,
		ExtraParams: map[string]interface{}{
			"max_tokens": 8192,
		},
	},
	{
		Name:             "deepseek/deepseek-coder",
		EditFormat:       "diff",
		UseRepoMap:       true,
		ExamplesAsSysMsg: true,
		Reminder:         "sys",
		CachesByDefault:  true,
		ExtraParams: map[string]interface{}{
			"max_tokens": 8192,
		},
	},
}

// Helper function to create string pointer
func ptrStr(s string) *string {
	return &s
}
