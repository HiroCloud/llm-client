package llm_models

type PromptStreamCommand string

const (
	PromptStreamCommandText           = PromptStreamCommand("text")
	PromptStreamCommandFunction       = PromptStreamCommand("function")
	PromptStreamCommandFunctionFinish = PromptStreamCommand("function_finish")
	PromptStreamUpdateQuestions       = PromptStreamCommand("function_new_question")
	PromptStreamCommandEnd            = PromptStreamCommand("end")
)

type Role string

const (
	RoleUser      Role = "user"
	RoleSystem    Role = "system"
	RoleAssistant Role = "assistant"
)

type ModelType string

var (
	ModelTypeChat       ModelType = "chat"
	ModelTypeImage      ModelType = "image"
	ModelTypeMultiModal ModelType = "multiple_modal"
	ModelTypeAudio      ModelType = "audio"
)

type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonFunctionCall  FinishReason = "function_call"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonNull          FinishReason = "null"
)

type ChatMessagePartType string

const (
	ChatMessagePartTypeText     ChatMessagePartType = "text"
	ChatMessagePartTypeImageURL ChatMessagePartType = "image_url"
)

type ImageURLDetail string

const (
	ImageURLDetailHigh ImageURLDetail = "high"
	ImageURLDetailLow  ImageURLDetail = "low"
	ImageURLDetailAuto ImageURLDetail = "auto"
)

type ChatCompletionResponseFormatType string

const (
	ChatCompletionResponseFormatTypeJSONObject ChatCompletionResponseFormatType = "json_object"
	ChatCompletionResponseFormatTypeJSONSchema ChatCompletionResponseFormatType = "json_schema"
	ChatCompletionResponseFormatTypeText       ChatCompletionResponseFormatType = "text"
)
