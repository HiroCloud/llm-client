package tools

import (
	"encoding/json"
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
	var loadedTool llm_models.Tool
	err := json.Unmarshal(data, &loadedTool)
	if err != nil {
		return nil, fmt.Errorf("invalid tool format:%w", err)
	}
	d, err := CreateDef(funcCall)
	if err != nil {
		return nil, fmt.Errorf("failed ceating tool verification def: %w", err)
	}
	if err := verifyTool(d, &loadedTool); err != nil {
		return nil, err
	}
	return &loadedTool, nil
}

func verifyTool(t1, t2 *llm_models.Tool) error {
	if len(t1.Function.ParamOrder) != len(t2.Function.ParamOrder) {
		return fmt.Errorf(
			"invalid function param order, expected %d params, actual %d",
			len(t1.Function.ParamOrder),
			len(t2.Function.ParamOrder),
		)
	}
	t1Def := t1.Function.Parameters
	t2Def := t2.Function.Parameters
	if t1Def.Type != t2Def.Type {
		return fmt.Errorf("missmatch type def")
	}
	//todo run more verification
	return nil
}
