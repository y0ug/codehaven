package tokeniz

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/pkoukk/tiktoken-go"
	"github.com/sugarme/tokenizer/pretrained"
	"github.com/y0ug/codehaven/internal/middleware"
	"github.com/y0ug/llmhaven"
	"github.com/y0ug/llmhaven/chat"
	"github.com/y0ug/llmhaven/http/options"
	"github.com/y0ug/llmhaven/providers/anthropic"
)

type GetCoordinatesInput struct {
	Location string `json:"location" jsonschema_description:"The location to look up."`
}

var GetCoordinatesInputSchema = GenerateSchema[GetCoordinatesInput]()

type GetCoordinateResponse struct {
	Long float64 `json:"long"`
	Lat  float64 `json:"lat"`
}

func GetCoordinates(location string) GetCoordinateResponse {
	return GetCoordinateResponse{
		Long: -122.4194,
		Lat:  37.7749,
	}
}

// Get Temperature Unit

type GetTemperatureUnitInput struct {
	Country string `json:"country" jsonschema_description:"The country"`
}

var GetTemperatureUnitInputSchema = GenerateSchema[GetTemperatureUnitInput]()

func GetTemperatureUnit(country string) string {
	return "farenheit"
}

// Get Weather

type GetWeatherInput struct {
	Lat  float64 `json:"lat"  jsonschema_description:"The latitude of the location to check weather."`
	Long float64 `json:"long" jsonschema_description:"The longitude of the location to check weather."`
	Unit string  `json:"unit" jsonschema_description:"Unit for the output"`
}

var GetWeatherInputSchema = GenerateSchema[GetWeatherInput]()

type GetWeatherResponse struct {
	Unit        string  `json:"unit"`
	Temperature float64 `json:"temperature"`
}

func GetWeather(lat, long float64, unit string) GetWeatherResponse {
	return GetWeatherResponse{
		Unit:        "farenheit",
		Temperature: 122,
	}
}

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	return reflector.Reflect(v)
}

func ToPtr[T any](s T) *T {
	return &s
}

func genTools() []chat.Tool {
	tools := []chat.Tool{
		{
			Name: "get_coordinates",
			Description: ToPtr(
				"Accepts a place as an address, then returns the latitude and longitude coordinates.",
			),
			InputSchema: GetCoordinatesInputSchema,
		},
		{
			Name:        "get_temperature_unit",
			InputSchema: GetTemperatureUnitInputSchema,
		},
		{
			Name:        "get_weather",
			Description: ToPtr("Get the weather at a specific location"),
			InputSchema: GetWeatherInputSchema,
		},
	}
	return tools
}

func TestNumTokensFromMessagesOpenAI(t *testing.T) {
	testCases := []struct {
		name        string
		model       string
		messages    []*chat.ChatMessage
		tools       []chat.Tool
		description string
	}{
		{
			name:  "SimpleUserMessageGPT4",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage("Hello! How's the weather today?"),
			},
			tools:       nil,
			description: "GPT-4 - Simple user message without tools",
		},
		{
			name:  "WithToolsGPT4",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage(
					"What's the coordinates of 13 calade st come? What the weather in Paris?",
				),
			},
			tools:       genTools(),
			description: "GPT-4 - With tools",
		},
		{
			name:  "MultiTurnConversationGPT4",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewSystemMessage("You're a weather expert."),
				chat.NewUserMessage("What's the weather like in Paris?"),
				chat.NewMessage("assistant", chat.NewTextContent("It's currently sunny in Paris.")),
				chat.NewUserMessage("What about tomorrow?"),
			},
			tools:       genTools(),
			description: "GPT-4 - Multi-turn conversation with tools",
		},
		{
			name:  "GPT3.5Turbo",
			model: "gpt-3.5-turbo",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage("Explain quantum computing in simple terms"),
			},
			tools:       nil,
			description: "GPT-3.5 - Simple explanation",
		},
	}

	runTokenTest(t, "openai", testCases, TokenCounterOpenAI)
}

func TestNumTokensFromMessagesAnthropic(t *testing.T) {
	testCases := []struct {
		name        string
		model       string
		messages    []*chat.ChatMessage
		tools       []chat.Tool
		description string
	}{
		{
			name:  "SimpleUserMessageSonnet",
			model: "claude-3-5-sonnet-20241022",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage("Hello! How's the weather today?"),
			},
			tools:       nil,
			description: "Claude - Simple user message without tools",
		},
		{
			name:  "WithToolsSonnet",
			model: "claude-3-5-sonnet-20241022",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage(
					"What's the coordinates of 13 calade st come? What the weather in Paris?",
				),
			},
			tools:       genTools(),
			description: "Claude - With tools",
		},
		{
			name:  "MultiTurnConversationSonnet",
			model: "claude-3-5-sonnet-20241022",
			messages: []*chat.ChatMessage{
				chat.NewSystemMessage("You're a weather expert."),
				chat.NewUserMessage("What's the weather like in Paris?"),
				chat.NewMessage("assistant", chat.NewTextContent("It's currently sunny in Paris.")),
				chat.NewUserMessage("What about tomorrow?"),
			},
			tools:       genTools(),
			description: "Claude - Multi-turn conversation with tools",
		},
	}

	runTokenTest(t, "anthropic", testCases, TokenCounter)
}

func TestAnthropicApiCountToken(t *testing.T) {
	testCases := []struct {
		name        string
		model       string
		messages    []*chat.ChatMessage
		tools       []chat.Tool
		description string
	}{
		{
			name:  "SimpleUserMessageSonnet",
			model: "claude-3-5-sonnet-20241022",
			messages: []*chat.ChatMessage{
				// chat.NewSystemMessage("You're a weather expert."),
				chat.NewUserMessage(
					"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Ut finibus, quam sit amet eleifend vehicula, enim mi interdum arcu, at vehicula nulla risus eu neque. Morbi in volutpat metus. Interdum et malesuada fames ac ante ipsum primis in faucibus. In et commodo elit. Cras a fermentum ex. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Maecenas quis aliquam nisl, nec finibus urna. Nam metus nisi, consectetur non finibus ullamcorper, auctor non tortor.",
				),
				// chat.NewMessage("assistant", chat.NewTextContent("It's currently sunny in Paris.")),
				// chat.NewUserMessage("What about tomorrow?"),
			},
			tools:       nil,
			description: "Claude - Simple user message without tools",
		},
		{
			name:  "WithToolsSonnet",
			model: "claude-3-5-sonnet-20241022",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage(
					"What's the coordinates of 13 calade st come? What the weather in Paris?",
				),
			},
			tools:       genTools(),
			description: "Claude - With tools",
		},
		{
			name:  "MultiTurnConversationSonnet",
			model: "claude-3-5-sonnet-20241022",
			messages: []*chat.ChatMessage{
				chat.NewSystemMessage("You're a weather expert."),
				chat.NewUserMessage("What's the weather like in Paris?"),
				chat.NewMessage("assistant", chat.NewTextContent("It's currently sunny in Paris.")),
				chat.NewUserMessage("What about tomorrow?"),
			},
			tools:       genTools(),
			description: "Claude - Multi-turn conversation with tools",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := "anthropic"
			var counter Counter
			configFile := "anthropic_tokenizer.json"
			tk, err := pretrained.FromFile(configFile)
			if err != nil {
				t.Skipf(
					"Skipping %s: failed to load Anthropic tokenizer: %v",
					tc.description,
					err,
				)
				return
			}
			counter = TokenizerCounter(tk)
			localInputTokens := TokenCounter(counter, tc.messages, tc.tools, false)

			apiInputTokens, err := ApiCountToken(
				context.Background(),
				provider,
				tc.model,
				tc.messages,
				tc.tools,
			)

			t.Logf("Local vs API Input Tokens: %d vs %d", localInputTokens, apiInputTokens)

			if err != nil {
				t.Fatalf("API call failed: %v", err)
			}
		})
	}
}

// Helper function to run the token tests
func runTokenTest(t *testing.T, provider string, testCases []struct {
	name        string
	model       string
	messages    []*chat.ChatMessage
	tools       []chat.Tool
	description string
}, tokenCounter func(Counter, []*chat.ChatMessage, []chat.Tool, bool) int,
) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var counter Counter

			switch provider {
			case "anthropic":
				configFile := "anthropic_tokenizer.json"
				tk, err := pretrained.FromFile(configFile)
				if err != nil {
					t.Skipf(
						"Skipping %s: failed to load Anthropic tokenizer: %v",
						tc.description,
						err,
					)
					return
				}
				counter = TokenizerCounter(tk)

			case "openai":
				encoding := GetEncodingByModelName(tc.model)
				if encoding == "" {
					t.Skipf(
						"Skipping %s: encoding not found for model %s",
						tc.description,
						tc.model,
					)
					return
				}
				tkm, err := tiktoken.GetEncoding(encoding)
				if err != nil {
					t.Fatalf("Failed to get encoding: %v", err)
				}
				counter = TiktokenCounter(tkm)

			default:
				t.Fatalf("Unknown provider: %s", provider)
			}

			localInputTokens := tokenCounter(counter, tc.messages, tc.tools, false)

			resp, err := ChatCompletion(
				context.Background(),
				provider,
				tc.model,
				tc.messages,
				tc.tools,
			)
			if err != nil {
				t.Fatalf("API call failed: %v", err)
			}

			respMsg := resp.ToMessageParams()
			localOutputTokens := tokenCounter(counter, []*chat.ChatMessage{respMsg}, nil, true)

			totalMsgs := append(tc.messages, respMsg)
			totalTokens := tokenCounter(counter, totalMsgs, tc.tools, true) // Log results

			t.Logf("\n=== Test Case: %s ===", tc.description)
			t.Logf("Model: %s", tc.model)
			t.Logf("Local vs API Input Tokens: %d vs %d", localInputTokens, resp.Usage.InputTokens)
			t.Logf(
				"Local vs API Output Tokens: %d vs %d",
				localOutputTokens,
				resp.Usage.OutputTokens,
			)
			t.Logf("Local vs API Total Tokens: %d vs %d",
				localInputTokens+localOutputTokens,
				resp.Usage.InputTokens+resp.Usage.OutputTokens+resp.Usage.InputCachedTokens)
			t.Logf("TotalTokens vs API Total Tokens: %d vs %d",
				totalTokens,
				resp.Usage.InputTokens+resp.Usage.OutputTokens+resp.Usage.InputCachedTokens)
			t.Logf("Local vs TotalTokens Total Tokens: %d vs %d",
				localInputTokens+localOutputTokens,
				totalTokens)

			// Validation checks
			if abs(totalTokens-localOutputTokens-localInputTokens) > 2 {
				t.Errorf("Total token mismatch exceeds threshold: %d vs %d",
					totalTokens,
					localOutputTokens+localInputTokens)
			}
			if abs(localInputTokens-resp.Usage.InputTokens) > 2 {
				t.Errorf("Input token mismatch exceeds threshold: %d vs %d",
					localInputTokens,
					resp.Usage.InputTokens)
			}
			if abs(localOutputTokens-resp.Usage.OutputTokens) > 2 {
				t.Errorf("Output token mismatch exceeds threshold: %d vs %d",
					localOutputTokens,
					resp.Usage.OutputTokens)
			}
		})
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func ChatCompletion(
	ctx context.Context,
	provider string,
	model string,
	msgs []*chat.ChatMessage,
	tools []chat.Tool,
) (*chat.ChatResponse, error) {
	ctxRequest, cancelFn := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()

	requestOpts := []options.RequestOption{
		options.WithMiddleware(middleware.LoggingMiddleware()),
		// options.WithMiddleware(middleware.TimeitMiddleware(nil)),
	}

	// modelInfoProvider, _ := modelinfo.New("")

	llm, err := llmhaven.New(provider, requestOpts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to create provider: %v", err)
	}

	params := chat.NewChatParams(
		chat.WithModel(model),
		chat.WithMessages(msgs...),
		chat.WithTools(tools...),
		chat.WithMaxTokens(1000),
	)
	resp, err := llm.Send(
		ctxRequest,
		*params,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to send message: %v", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("Response is nil")
	}

	return resp, nil
}

func ApiCountToken(
	ctx context.Context,
	provider string,
	model string,
	msgs []*chat.ChatMessage,
	tools []chat.Tool,
) (int64, error) {
	ctxRequest, cancelFn := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()

	requestOpts := []options.RequestOption{
		// options.WithMiddleware(middleware.LoggingMiddleware()),
		// options.WithMiddleware(middleware.TimeitMiddleware(nil)),
	}

	// modelInfoProvider, _ := modelinfo.New("")

	llm, err := llmhaven.New(provider, requestOpts...)
	if err != nil {
		return 0, fmt.Errorf("Failed to create provider: %v", err)
	}

	params := chat.NewChatParams(
		chat.WithModel(model),
		chat.WithMessages(msgs...),
		chat.WithTools(tools...))

	if llm, ok := llm.(*anthropic.Provider); ok {
		tokens, err := llm.CountTokens(ctxRequest, *params)
		if err != nil {
			return 0, fmt.Errorf("Failed to count tokens: %v", err)
		}
		return tokens, nil
	}

	return 0, fmt.Errorf("Provider not supported")
}
