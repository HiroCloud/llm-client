package llm_client

import (
	"context"
	"fmt"
	"github.com/HiroCloud/llm-client/llm_models"
	t "github.com/HiroCloud/llm-client/tools"
)

// ResolveChatWithTools drives a chat with an AI model that can call functions (tools).
// - client: an AIClient capable of generating chat responses with potential function calls.
// - messages: the current conversation history (slice of Message or similar, including system/user/assistant messages).
// - tools: a list of available Tool definitions (each with FuncDef, CallFunc, WriteToChat, ExitFunc).
// - maxCalls: safety limit on the total number of function calls to execute (to avoid infinite loops).
// Returns the final answer from the model after handling any tool calls, or an error.
func ResolveChatWithTools(
	ctx context.Context,
	client AIClient,
	messages []Message,
	tools []llm_models.Tool,
	maxCalls int,
) (string, error) {
	// Prepare a lookup map for tools by name for convenience.
	toolMap := make(map[string]llm_models.Tool)
	for _, tool := range tools {
		name := tool.Function.Name // assuming FuncDef has a Name field
		toolMap[name] = tool
	}

	callCount := 0
	for callCount < maxCalls {
		// Ask the AI model for the next response (which may include function call requests).
		result, err := client.GenerateResponse(ctx, messages, tools)
		if err != nil {
			return "", fmt.Errorf("AIClient generation error: %w", err)
		}

		// Check if the model is requesting any function calls.
		if len(result.FunctionCalls) == 0 {
			// No function call means the model has produced a final answer.
			finalAnswer := result.Content // assume Content holds the assistant's reply
			return finalAnswer, nil
		}

		// The model has requested one or more function calls (possibly parallel calls).
		for _, fc := range result.FunctionCalls {
			callCount++
			// Prevent exceeding maxCalls in the middle of processing multiple calls
			if callCount > maxCalls {
				return "", fmt.Errorf("maxCalls limit (%d) exceeded – aborting to prevent infinite loop", maxCalls)
			}

			toolName := fc.Name
			tool, ok := toolMap[toolName]
			if !ok {
				// Unknown function requested by the model.
				errMsg := fmt.Sprintf("model requested unknown tool '%s'", toolName)
				// Append an error message for transparency, then stop.
				messages = append(messages, Message{
					Role:    "assistant",
					Content: errMsg,
				})
				return "", fmt.Errorf(errMsg)
			}

			// Use reflection or provided CallTool utility to invoke the tool function with arguments.
			var toolResult interface{}
			var toolErr error
			if tool.CallFunc != nil {
				// CallTool is assumed to execute the function pointer with the given arguments map.
				toolResult, toolErr = t.CallJSONStr(ctx, &tool, fc.Arguments)
			} else {
				return "", fmt.Errorf("call func does not exist")
			}

			// Format the tool result for the chat. Use WriteToChat if available, otherwise default formatting.
			var resultText string
			if toolErr != nil {
				// If the tool returned an error, include that info.
				resultText = fmt.Sprintf("Error calling %s: %v", toolName, toolErr)
			} else {
				resultText = fmt.Sprintf("%v", toolResult)
			}

			// Append the function call and its result to the conversation history.
			// a) Record the assistant's function call (for the model's context).
			messages = append(messages, Message{
				Role:         "assistant",
				Content:      "", // no direct content, but we set the function call info
				FunctionCall: fc, // store the function call details (name & arguments)
			})
			// b) Record the function's response as a message from the function.
			messages = append(messages, Message{
				Role:    "function",
				Name:    toolName,
				Content: resultText, // output from the tool (formatted)
			})

			// If this tool is an exit signal, we break out early with its result.
			if tool.ExitFunc {
				return resultText, nil
			}
		}

		// After executing all requested function calls, loop continues.
		// The updated `messages` (with tool call results) will be sent in the next iteration
		// to get the model's follow-up response.
	}

	// If we exit the loop due to maxCalls exhaustion, return an error.
	return "", fmt.Errorf("stopped after %d function calls to prevent infinite loop", maxCalls)
}
