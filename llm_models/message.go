package llm_models

type Message struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content" json:"content,omitempty"`
	Role    Role   `json:"role" json:"role,omitempty"`

	Refusal bool `json:"refusal,omitempty" json:"refusal,omitempty"`

	Tools      []Tool `json:"tool_calls,omitempty" json:"tool_calls,omitempty"`
	ToolCallID string `json:"toolCallID" json:"toolCallID,omitempty"`
}

type Choice struct {
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
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
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Strict      bool     `json:"strict,omitempty"`
	ParamOrder  []string `json:"param_order,omitempty"`
	Parameters  any      `json:"parameters"`
}
