package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HiroCloud/llm-client/llm_models"
	"reflect"
	"strings"
)

// CallJSONStr parses a JSON string and dispatches a call to the appropriate tool function using
// either named or positional parameters.
func CallJSONStr(ctx context.Context, t *llm_models.Tool, jsonStr string) ([]interface{}, error) {
	if strings.HasPrefix(jsonStr, "{") {
		// Handle JSON object (named parameters)
		data := map[string]interface{}{}
		err := json.Unmarshal([]byte(jsonStr), &data)
		if err != nil {
			return nil, fmt.Errorf("json unmarshal fail: %w", err)
		}
		return CallToolMap(ctx, t, data)
	} else if strings.HasPrefix(jsonStr, "[") {
		// Handle JSON array (positional arguments)
		var args []interface{}
		err := json.Unmarshal([]byte(jsonStr), &args)
		if err != nil {
			return nil, fmt.Errorf("json unmarshal fail: %w", err)
		}

		return CallTool(ctx, t, args...)
	}

	return nil, errors.New("invalid json string")
}

// CallToolMap processes a map of parameters to invoke a specific tool function.
// It ensures that parameters match the required order and all mandatory parameters are provided.
// Returns the results of the tool execution or an error if validation or execution fails.
func CallToolMap(ctx context.Context, t *llm_models.Tool, data map[string]interface{}) ([]interface{}, error) {
	// Handle JSON object (named parameters)
	// Ensure params are in the correct order
	var args []interface{}
	for _, param := range t.Function.ParamOrder {
		if val, ok := data[param]; ok {
			args = append(args, val)
		} else {
			return nil, fmt.Errorf("missing required parameter: %s", param)
		}
	}

	return CallTool(ctx, t, args...)
}

// CallTool invokes the provided tool function with the given arguments using reflection, handling optional context contexts.
// Returns a slice of interface{} with the function results or an error if invocation fails.
// Validates argument types and counts against the tool's function signature.
func CallTool(ctx context.Context, t *llm_models.Tool, args ...interface{}) ([]interface{}, error) {
	funcValue := reflect.ValueOf(t.CallFunc)
	funcType := funcValue.Type()

	// Check if the first argument is context.Context
	hasContext := funcType.NumIn() > 0 && funcType.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem()

	expectedArgCount := funcType.NumIn()
	if hasContext {
		expectedArgCount-- // Account for ctx being optional
	}

	if len(args) != expectedArgCount {
		return nil, fmt.Errorf("expected %d arguments, got %d", expectedArgCount, len(args))
	}

	// Prepare arguments for reflection call
	var callArgs []reflect.Value
	if hasContext {
		callArgs = append(callArgs, reflect.ValueOf(ctx)) // Add context only if required
	}

	for i, arg := range args {
		expectedType := funcType.In(i)
		if hasContext {
			expectedType = funcType.In(i + 1) // Adjust index if context is present
		}

		argValue := reflect.ValueOf(arg)
		if !argValue.Type().ConvertibleTo(expectedType) {
			return nil, fmt.Errorf("argument %d: expected %s, got %s", i, expectedType, argValue.Type())
		}
		callArgs = append(callArgs, argValue.Convert(expectedType))
	}

	// Call the function
	resp := funcValue.Call(callArgs)

	// Convert response to a slice of interfaces
	var result []interface{}
	for _, r := range resp {
		if r.Kind() == reflect.Interface && r.IsNil() { // Handle nil interfaces
			result = append(result, nil)
		} else {
			result = append(result, r.Interface())
		}
	}

	// Handle error as a special case
	if len(resp) > 0 && funcType.Out(funcType.NumOut()-1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if errVal := resp[len(resp)-1]; !errVal.IsNil() {
			return result, errVal.Interface().(error)
		}
	}

	return result, nil
}
