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
