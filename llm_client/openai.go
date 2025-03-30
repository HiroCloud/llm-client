package llm_client

import (
	"context"
	"github.com/HiroCloud/llm-client/llm_models"
	openai "github.com/sashabaranov/go-openai"
)

var _ LLM = OpenAI{}

type OpenAI struct {
	client *openai.Client
}

func NewOpenAI(key string, config *openai.ClientConfig) (LLM, error) {
	if config == nil {
		c := openai.DefaultConfig(key)
		config = &c
		config.AssistantVersion = "v2"
	}

	l := OpenAI{
		client: openai.NewClientWithConfig(*config),
	}

	return l, nil
}
func (o OpenAI) SupportedModels(modelType llm_models.ModelType) []string {
	//TODO implement me
	panic("implement me")
}

func (o OpenAI) AddModel(modelType llm_models.ModelType, modelName string) {
	//TODO implement me
	panic("implement me")
}

func (o OpenAI) AddTool(tool llm_models.Tool) {
	//TODO implement me
	panic("implement me")
}

func (o OpenAI) Stream(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (o OpenAI) Generate(ctx context.Context, modelType llm_models.ModelType, message *llm_models.Message) {
	//TODO implement me
	panic("implement me")
}
