package tools

import (
	"fmt"
	"github.com/HiroCloud/llm-client/llm_models"
	"os"
)

func NewTool(funcCall interface{}, def llm_models.FuncDef) (*llm_models.Tool, error) {
	return nil, nil
}

func NewToolFromFile(funcCall interface{}, fileName string) (*llm_models.Tool, error) {
	if _, err := os.Stat(fileName); err != nil {
		return nil, fmt.Errorf("file %s not exists", fileName)
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return NewToolFromBytes(funcCall, data)
}

func NewToolFromBytes(funcCall interface{}, data []byte) (*llm_models.Tool, error) {
	return nil, nil
}
