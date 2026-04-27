package llm_client

import (
	"context"
	"github.com/HiroCloud/llm-client/llm_models"
)

// Role constants for consistency
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleFunction  = "function"
)

// Message represents a single message in a chat (user, assistant, etc.).
type Message struct {
	Role         string // "user", "assistant", "system", or "function"
	Content      string // The text content of the message
	Name         string // Optional name (e.g. function name if Role=="function")
	FunctionCall *FunctionCall
}

// FunctionDef describes a function for the model to potentially call.
type FunctionDef struct {
	Name        string      // Function name
	Description string      // Human-readable description
	Parameters  interface{} // JSON schema or Go struct for parameters (per OpenAI API)
}

// Options for generation (temperature, max tokens, etc.)
type GenOptions struct {
	Temperature float64 // Sampling temperature (0.0-1.0)
	TopP        float64 // Nucleus sampling probability
	MaxTokens   int     // Maximum tokens to generate in the response
}

// Unified request/response types for chat, text, and image generation:
type ChatRequest struct {
	Model        string        // Model name/ID (e.g. "gpt-4" or "gemini-2.0")
	Messages     []Message     // Conversation history (last message typically from user)
	Functions    []FunctionDef // Optional function definitions for function calling
	FunctionCall string        // "auto" (default), "none", or specific function name to force
	Options      GenOptions    // Generation parameters (temperature, max tokens, etc.)
}
type ChatResponse struct {
	Choices []GenChoice // One or more generated chat completions (assistant messages)
	Usage   TokenUsage  // Token usage data for the request
}

// GenChoice represents a single generated message or completion choice.
type GenChoice struct {
	Content       string          // The generated text content (empty if function call)
	FinishReason  string          // e.g. "stop", "length", "function_call"
	FunctionCalls []*FunctionCall // Function call info (if FinishReason == "function_call")
}

// FunctionCall holds details of a model-invoked function call.
type FunctionCall struct {
	ID        string
	Name      string // Name of the function the model wants to call
	Arguments string // JSON-encoded arguments for the function
}

type TextRequest struct {
	Model   string     // Model name (e.g. "text-davinci-003" or a Gemini text model)
	Prompt  string     // Prompt text to complete
	Options GenOptions // Generation parameters
}
type TextResponse struct {
	Choices []GenChoice // One or more completion results (each as text content)
	Usage   TokenUsage  // Token usage data
}

type ImageRequest struct {
	Model  string // Model name/ID for image generation
	Prompt string // Prompt describing the image to generate
	N      int    // Number of images to generate
	// (Optional parameters like Size or AspectRatio could be included here)
}
type ImageResponse struct {
	Images []ImageData // Generated images (data or URLs)
	Usage  TokenUsage  // Token usage data (if provided by API)
}

// ImageData holds a generated image, either as raw bytes or a URL.
type ImageData struct {
	Data []byte // Image binary data (if available)
	URL  string // URL to the image (if returned by the API instead of bytes)
}

// Standardized token usage information.
type TokenUsage struct {
	PromptTokens     int // tokens in the prompt/input
	CompletionTokens int // tokens in the completion/output
	TotalTokens      int // total tokens consumed
}

type Response struct {
	Content       string          // assistant’s textual reply (empty if purely function calls)
	FunctionCalls []*FunctionCall // the model’s requested tool calls (in order)
	Usage         TokenUsage      // token usage, if available
}

// ChatStream is a streaming chat response.
type ChatStream interface {
	// Recv returns the next partial GenChoice (or io.EOF when done)
	Recv() (GenChoice, error)
	// Close cleans up the underlying stream
	Close() error
}

// TextStream is a streaming text completion.
type TextStream interface {
	// Recv returns the next partial GenChoice (or io.EOF when done)
	Recv() (GenChoice, error)
	// Close cleans up the underlying stream
	Close() error
}

// AIClient abstracts the model operations across OpenAI and Gemini.
type AIClient interface {
	// Chat completion with optional streaming
	ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error)
	ChatCompletionStream(ctx context.Context, req ChatRequest) (ChatStream, error)

	// Text completion with optional streaming
	TextCompletion(ctx context.Context, req TextRequest) (TextResponse, error)
	TextCompletionStream(ctx context.Context, req TextRequest) (TextStream, error)

	// Image generation (no streaming for images)
	GenerateImage(ctx context.Context, req ImageRequest) (ImageResponse, error)

	GenerateResponse(ctx context.Context, messages []Message, tools []llm_models.Tool) (Response, error)
}
