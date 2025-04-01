package tools

import (
	"github.com/HiroCloud/llm-client/llm_models"
	"github.com/Hirocloud/mcp-go/mcp"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// GetToolMCP converts an llm_models.Tool object into a mcp.Tool object with configured options based on its definition.
func GetToolMCP(t *llm_models.Tool) (*mcp.Tool, error) {
	def := t.Function.Parameters

	options, err := getMCPToolOptions(t, def)
	if err != nil {
		return nil, err
	}
	baseOptions := []mcp.ToolOption{
		mcp.WithDescription(t.Function.Description),
	}
	tool := mcp.NewTool(t.Function.Name, append(baseOptions, options...)...)
	return &tool, nil
}

// getMCPToolOptions generates a list of MCP tool options based on the provided tool and JSON schema definition.
func getMCPToolOptions(t *llm_models.Tool, def jsonschema.Definition) ([]mcp.ToolOption, error) {
	var options []mcp.ToolOption
	requiredMap := map[string]bool{}
	for _, r := range def.Required {
		requiredMap[r] = true
	}
	for k, v := range def.Properties {
		var option mcp.ToolOption
		switch v.Type {
		case jsonschema.String:
			if _, ok := requiredMap[k]; ok {
				option = mcp.WithString(k,
					mcp.Description(v.Description),
					mcp.Required(),
				)
			} else {
				option = mcp.WithString(k,
					mcp.Description(v.Description),
				)
			}
		case jsonschema.Integer:
			if _, ok := requiredMap[k]; ok {
				option = mcp.WithNumber(k,
					mcp.Description(v.Description),
					mcp.Required(),
				)
			} else {
				option = mcp.WithNumber(k,
					mcp.Description(v.Description),
				)
			}
		case jsonschema.Boolean:
			if _, ok := requiredMap[k]; ok {
				option = mcp.WithBoolean(k,
					mcp.Description(v.Description),
					mcp.Required(),
				)
			} else {
				option = mcp.WithBoolean(k,
					mcp.Description(v.Description),
				)
			}
		case jsonschema.Array:
			var props interface{}
			props = v.Items
			if _, ok := requiredMap[k]; ok {
				option = mcp.WithArray(k,
					mcp.Description(v.Description),
					mcp.Items(props),
					mcp.Required(),
				)
			} else {
				option = mcp.WithArray(k,
					mcp.Description(v.Description),
					mcp.Items(props),
				)
			}
		case jsonschema.Object:
			props := make(map[string]interface{})
			for k, prop := range v.Properties {
				props[k] = prop
			}
			if _, ok := requiredMap[k]; ok {
				option = mcp.WithObject(k,
					mcp.Description(v.Description),
					mcp.Properties(props),
					mcp.Required(),
				)
			} else {
				option = mcp.WithObject(k,
					mcp.Description(v.Description),
					mcp.Properties(props),
				)
			}

		case jsonschema.Number:
			if _, ok := requiredMap[k]; ok {
				option = mcp.WithNumber(k,
					mcp.Description(v.Description),
					mcp.Required(),
				)
			} else {
				option = mcp.WithNumber(k,
					mcp.Description(v.Description),
				)
			}
		case jsonschema.Null:
			fallthrough
		default:
			continue
		}

		options = append(options, option)
	}

	return options, nil
}
