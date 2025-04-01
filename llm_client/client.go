package llm_client

import (
	"context"
	"github.com/HiroCloud/llm-client/llm_models"
)

type LLM interface {
	SupportedModels(modelType llm_models.ModelType) []string
	AddModel(modelType llm_models.ModelType, modelName string)
	AddTool(tool llm_models.Tool)

	Stream(ctx context.Context)
	Generate(ctx context.Context, config *llm_models.ModelRequestConfig, message *llm_models.Message) (*llm_models.Response, error)
}
