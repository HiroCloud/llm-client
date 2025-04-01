package llm_client

import (
	"context"
	"fmt"
	"github.com/HiroCloud/llm-client/llm_models"
	openai "github.com/sashabaranov/go-openai"
)

var _ LLM = &OpenAI{}

type OpenAI struct {
	client          *openai.Client
	supportedModels map[llm_models.ModelType][]string
	tools           map[string]llm_models.Tool
}

func NewOpenAI(key string, config *openai.ClientConfig) (LLM, error) {
	if config == nil {
		c := openai.DefaultConfig(key)
		config = &c
		config.AssistantVersion = "v2"
	}

	l := &OpenAI{
		client:          openai.NewClientWithConfig(*config),
		supportedModels: map[llm_models.ModelType][]string{},
		tools:           map[string]llm_models.Tool{},
	}

	return l, nil
}
func (o *OpenAI) SupportedModels(modelType llm_models.ModelType) []string {
	if _, f := o.supportedModels[modelType]; !f {
		return []string{}
	} else {
		return o.supportedModels[modelType]
	}
}

func (o *OpenAI) AddModel(modelType llm_models.ModelType, modelName string) {
	if _, f := o.supportedModels[modelType]; !f {
		o.supportedModels[modelType] = []string{}
	}
	o.supportedModels[modelType] = append(o.supportedModels[modelType], modelName)
}

func (o *OpenAI) AddTool(tool llm_models.Tool) {
	o.tools[tool.Function.Name] = tool
}

func (o *OpenAI) Stream(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (o *OpenAI) Generate(ctx context.Context, config *llm_models.ModelRequestConfig, message *llm_models.Message) (*llm_models.Response, error) {
	if _, f := o.supportedModels[config.ModelType]; !f {
		return nil, fmt.Errorf("unsupported model type: %s", config.ModelType)
	}
	if !inList(config.Model, o.supportedModels[config.ModelType]) {
		return nil, fmt.Errorf("model not supported")
	}
	req := openai.ChatCompletionRequest{
		Model: config.Model,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatType(config.ResponseFormat),
		},
		Tools: make([]openai.Tool, 0),
	}
	for _, tool := range o.tools {
		req.Tools = append(req.Tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Strict:      tool.Function.Strict,
				Parameters:  tool.Function.Parameters,
			},
		})

	}
	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	llmResponse := &llm_models.Response{
		ID:                resp.ID,
		Object:            resp.Object,
		Created:           resp.Created,
		Model:             resp.Model,
		Choices:           []llm_models.Choice{},
		Usage:             llm_models.Usage{},
		SystemFingerprint: resp.SystemFingerprint,
	}
	for _, choice := range resp.Choices {
		var tools []llm_models.Tool
		var mtc []llm_models.ChatMessagePart
		for _, tc := range choice.Message.ToolCalls {
			if t, found := o.tools[tc.Function.Name]; found {
				tool := llm_models.Tool{
					Function: llm_models.FuncDef{
						Name:      tc.Function.Name,
						Strict:    t.Function.Strict,
						Arguments: tc.Function.Arguments,
					},
					CallFunc:    t.CallFunc,
					ExitFunc:    t.ExitFunc,
					WriteToChat: t.WriteToChat,
				}
				tools = append(tools, tool)
			}
		}
		for _, mc := range choice.Message.MultiContent {
			var imgURL *llm_models.ChatMessageImageURL
			if mc.ImageURL != nil {
				imgURL = &llm_models.ChatMessageImageURL{
					URL:    mc.ImageURL.URL,
					Detail: llm_models.ImageURLDetail(mc.ImageURL.Detail),
				}
			}
			mtc = append(mtc, llm_models.ChatMessagePart{
				Type:     llm_models.ChatMessagePartType(mc.Type),
				Text:     mc.Text,
				ImageURL: imgURL,
			})
		}

		m := llm_models.Message{
			Name:         choice.Message.Name,
			Content:      choice.Message.Content,
			Role:         llm_models.Role(choice.Message.Role),
			Refusal:      choice.Message.Refusal,
			MultiContent: mtc,
			Tools:        tools,
			ToolCallID:   choice.Message.ToolCallID,
		}

		var messages []*llm_models.Message
		if len(message.PreviousMessages) > 0 {
			messages = append(messages, message.PreviousMessages...)
		}
		messages = append(messages, &m)
		m.PreviousMessages = messages
		c := llm_models.Choice{
			Index:        choice.Index,
			Message:      m,
			FinishReason: llm_models.FinishReason(choice.FinishReason),
		}
		llmResponse.Choices = append(llmResponse.Choices, c)
	}

	return llmResponse, nil
}

func inList(m string, list []string) bool {
	for _, item := range list {
		if item == m {
			return true
		}
	}
	return false
}
