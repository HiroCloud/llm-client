package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/HiroCloud/llm-client/llm_models"
	"github.com/sashabaranov/go-openai/jsonschema"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
)

func SaveTool(dir string, name string, t *llm_models.Tool) (string, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if name == "" {
		name = t.Function.Name
	}
	f := path.Join(dir, name) + ".json"
	err = os.WriteFile(f, data, 0644)
	if err != nil {
		return "", err
	}
	return f, nil
}

func CreateDef(f interface{}) (*llm_models.Tool, error) {
	funcType := reflect.TypeOf(f)

	if funcType.Kind() == reflect.Func {
		return createFuncDef(f)
	}
	if funcType.Kind() == reflect.Struct {
		return CreateStruct(f)
	}
	return nil, nil

}

func CreateStruct(obj interface{}) (*llm_models.Tool, error) {
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem() // Dereference the pointer to get the struct type
	}

	if objType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("CreateStruct expects a struct or pointer to a struct")
	}
	structName := objType.Name()
	structDesc := structName + " defines a " + strings.ToLower(structName) + " object"
	def := jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: make(map[string]jsonschema.Definition),
	}

	var fieldOrder []string

	// Iterate over struct fields
	var requiredFields []string
	requiredReg := regexp.MustCompile("-|omitempty|omitzero")
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		// Extract field name and type
		fieldName := field.Name
		fieldType := field.Type

		// Extract field description from struct tag or comments (if available)
		fieldDesc := field.Tag.Get("desc")
		if fieldDesc == "" {
			fieldDesc = fieldName // Fallback to field name if no JSON tag is provided
		}
		jTag := field.Tag.Get("json")

		if !requiredReg.MatchString(jTag) {
			requiredFields = append(requiredFields, fieldName)
		}
		// Check if the field is a struct and recursively handle it
		if fieldType.Kind() == reflect.Struct {
			// Recursively create definition for nested struct
			nestedDef, err := CreateStruct(reflect.New(fieldType).Interface())
			if err != nil {
				return nil, fmt.Errorf("error creating nested struct definition for field %s: %w", fieldName, err)
			}

			// Add nested struct definition to the properties
			def.Properties[fieldName] = nestedDef.Function.Parameters.(jsonschema.Definition)
		} else if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			// Check if the array/slice is of structs
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Struct {
				// Recursively create definition for nested struct in the array/slice
				nestedDef, err := CreateStruct(reflect.New(elemType).Interface())
				if err != nil {
					return nil, fmt.Errorf("error creating nested struct definition for array/slice element in field %s: %w", fieldName, err)
				}

				d := nestedDef.Function.Parameters.(jsonschema.Definition)
				// Add array/slice of structs to the properties
				def.Properties[fieldName] = jsonschema.Definition{
					Type:        jsonschema.Array,
					Items:       &d,
					Description: fieldDesc,
				}
			} else {
				// Handle array/slice of non-struct types
				def.Properties[fieldName] = jsonschema.Definition{
					Type:        mapType(elemType),
					Description: fieldDesc,
				}
			}
		} else if fieldType.Kind() == reflect.String {
			// Check if the field is a custom string type (like Role, PromptStreamCommand)
			if isCustomType(fieldType) {
				// Retrieve the constants for the custom type (e.g., Role, PromptStreamCommand)
				enumValues := getEnumValuesForCustomType(fieldType)

				if len(enumValues) > 0 {
					def.Properties[fieldName] = jsonschema.Definition{
						Type:        jsonschema.String,
						Enum:        enumValues,
						Description: fieldDesc,
					}
				}
			}
		} else {
			// Add the field to the definition if it's not a nested struct or array/slice
			def.Properties[fieldName] = jsonschema.Definition{
				Type:        mapType(fieldType),
				Description: fieldDesc,
			}
		}

		fieldOrder = append(fieldOrder, fieldName)
	}
	def.Required = requiredFields
	return &llm_models.Tool{
		Function: llm_models.FuncDef{
			Name:        structName,
			Description: structDesc,
			ParamOrder:  fieldOrder,
			Parameters:  def,
		},
		CallFunc: nil, // No function associated with a struct
	}, nil
}

func createFuncDef(f interface{}) (*llm_models.Tool, error) {
	funcType := reflect.TypeOf(f)
	if funcType.Kind() != reflect.Func {
		return nil, fmt.Errorf("CreateDef expects a function")
	}
	funcName := getFunctionName(f)
	funcDesc, data := getFunctionMetadata(f)
	def := jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: make(map[string]jsonschema.Definition),
	}

	// Iterate over function parameters
	for i := 0; i < funcType.NumIn(); i++ {
		paramType := funcType.In(i)
		paramDescription := ""
		var paramEnum []string

		// Skip context.Context if present
		if paramType == reflect.TypeOf((*context.Context)(nil)).Elem() {
			continue
		}

		paramName := fmt.Sprintf("param%d", i)
		if i < len(data) {
			paramName = data[i]
		}

		def.Properties[paramName] = jsonschema.Definition{
			Type:        mapType(paramType),
			Description: paramDescription,
			Enum:        paramEnum,
		}
		def.Required = append(def.Required, paramName)
	}

	t := &llm_models.Tool{
		Function: llm_models.FuncDef{
			Name:        funcName,
			Description: funcDesc,
			ParamOrder:  data,
			Parameters:  def,
		},
		CallFunc: f,
	}
	return t, nil

}
