package llm_models

import "github.com/sashabaranov/go-openai/jsonschema"

// Message represents a communication structure containing name, content, role, refusal state, tools, and a tool call ID.
type Message struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content" json:"content,omitempty"`
	Role    Role   `json:"role" json:"role,omitempty"`

	Refusal string `json:"refusal,omitempty" json:"refusal,omitempty"`

	MultiContent []ChatMessagePart `json:"multi_content,omitempty"`

	Tools      []Tool `json:"tool_calls,omitempty" json:"tool_calls,omitempty"`
	ToolCallID string `json:"toolCallID" json:"toolCallID,omitempty"`

	PreviousMessages []*Message `json:"previous_messages,omitempty"`
}

type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason"`
}

type FunctionCall struct {
	Name string `json:"name,omitempty"`
	// call function with arguments in JSON format
	Arguments string `json:"arguments,omitempty"`
}

// Tool implement tool
type Tool struct {
	// function define a function
	Function    FuncDef             `json:"function"`
	CallFunc    interface{}         `json:"-"` // some go func
	ExitFunc    bool                `json:"exit_func"`
	WriteToChat PromptStreamCommand `json:"write_to_chat"`
}

type FuncDef struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Strict      bool                  `json:"strict,omitempty"`
	ParamOrder  []string              `json:"param_order,omitempty"`
	Parameters  jsonschema.Definition `json:"parameters"`
	Arguments   string                `json:"arguments,omitempty"`
}

type ModelRequestConfig struct {
	ModelType      ModelType
	Model          string
	ResponseFormat ChatCompletionResponseFormatType
}

// ChatCompletionResponse represents a response structure for chat completion API.
type Response struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint"`
}

type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details"`
}

// CompletionTokensDetails Breakdown of tokens used in a completion.
type CompletionTokensDetails struct {
	AudioTokens     int `json:"audio_tokens"`
	ReasoningTokens int `json:"reasoning_tokens"`
}

// PromptTokensDetails Breakdown of tokens used in the prompt.
type PromptTokensDetails struct {
	AudioTokens  int `json:"audio_tokens"`
	CachedTokens int `json:"cached_tokens"`
}

type ChatMessagePart struct {
	Type     ChatMessagePartType  `json:"type,omitempty"`
	Text     string               `json:"text,omitempty"`
	ImageURL *ChatMessageImageURL `json:"image_url,omitempty"`
}

type ChatMessageImageURL struct {
	URL    string         `json:"url,omitempty"`
	Detail ImageURLDetail `json:"detail,omitempty"`
}
