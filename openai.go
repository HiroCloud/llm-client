package llm_client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/HiroCloud/llm-client/llm_models"
	openai "github.com/sashabaranov/go-openai"
) // OpenAI Go SDK

// OpenAIClient implements AIClient for OpenAI's models.
type OpenAIClient struct {
	client       *openai.Client // underlying OpenAI SDK client
	defaultModel string         // default model to use if none specified in request
}

// --- Chat Completion (OpenAI) ---
func (c *OpenAIClient) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}

	// 1) Map our Message → openai.ChatCompletionMessage
	msgs := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
			Name:    m.Name,
		}
	}

	// 2) Build the new ChatCompletionRequest
	openReq := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: float32(req.Options.Temperature),
		TopP:        float32(req.Options.TopP),
		MaxTokens:   req.Options.MaxTokens,
	}

	// 3) If you declared any FunctionDefs, convert them to Tools:
	if len(req.Functions) > 0 {
		tools := make([]openai.Tool, len(req.Functions))
		for i, fn := range req.Functions {
			// marshal your JSON schema / parameters
			paramsJSON, err := json.Marshal(fn.Parameters)
			if err != nil {
				return ChatResponse{}, err
			}
			tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        fn.Name,
					Description: fn.Description,
					Strict:      false,
					Parameters:  paramsJSON,
				},
			}
		}
		openReq.Tools = tools
		//openai.ToolChoice{}
		//		// 4) And tell OpenAI how you want the model to select a tool:
		//		//    Type can be "none", "auto", or "force"
		//		//    If you want to force a specific tool, set ToolName.
		//		openReq.ToolChoice
		//		openReq.ToolSelection = &openai.ToolSelection{
		//			Type:     openai.ToolSelectionType(req.FunctionCall), // e.g. "auto" or "none" or a specific name
		//			ToolName: "",                                         // leave blank for auto/none
		//		}
	}

	// 5) Call the API
	resp, err := c.client.CreateChatCompletion(ctx, openReq)
	if err != nil {
		return ChatResponse{}, err
	}

	// 6) Convert back to our ChatResponse
	out := ChatResponse{
		Usage: TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	for _, ch := range resp.Choices {
		choice := GenChoice{
			Content:      ch.Message.Content,
			FinishReason: string(ch.FinishReason),
		}
		// if the model invoked a tool, the response now lives under ch.Message.ToolResponse
		if tr := ch.Message.ToolCalls; tr != nil {
			if len(choice.FunctionCalls) == 0 {
				choice.FunctionCalls = make([]*FunctionCall, 0)
			}
			for _, t := range tr {
				choice.FunctionCalls = append(choice.FunctionCalls, &FunctionCall{
					ID:        t.ID,
					Name:      t.Function.Name,
					Arguments: t.Function.Arguments,
				})
			}
		}
		out.Choices = append(out.Choices, choice)
	}
	return out, nil
}

// ChatCompletionStream for OpenAI returns a stream of incremental chat chunks.
func (c *OpenAIClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (ChatStream, error) {
	// (Conversion of req to openReq is same as above)
	// ... prepare openReq as in ChatCompletion ...
	openReq := openai.ChatCompletionRequest{ /* ... fill as above ... */ }
	stream, err := c.client.CreateChatCompletionStream(ctx, openReq)
	if err != nil {
		return nil, err
	}
	return &openAIChatStream{inner: stream}, nil
}

// openAIChatStream wraps openai.ChatCompletionStream to implement ChatStream.
type openAIChatStream struct {
	inner *openai.ChatCompletionStream
}

func (s *openAIChatStream) Recv() (GenChoice, error) {
	resp, err := s.inner.Recv()
	if err != nil {
		return GenChoice{}, err // err will be io.EOF when stream is done
	}
	// OpenAI stream responses provide a "delta" for incremental content:contentReference[oaicite:4]{index=4}.
	delta := resp.Choices[0].Delta
	gen := GenChoice{}
	if delta.Content != "" {
		gen.Content = delta.Content
	}
	if delta.ToolCalls != nil {
		// Partial function call data (Name or Arguments may be partial)
		if len(gen.FunctionCalls) == 0 {
			gen.FunctionCalls = make([]*FunctionCall, 0)
		}
		for _, t := range delta.ToolCalls {
			gen.FunctionCalls = append(gen.FunctionCalls, &FunctionCall{
				ID:        t.ID,
				Name:      t.Function.Name,
				Arguments: t.Function.Arguments,
			})
		}
	}
	// Note: FinishReason is only sent in a final chunk (resp.Choices[0].FinishReason)
	gen.FinishReason = string(resp.Choices[0].FinishReason)
	return gen, nil
}

func (s *openAIChatStream) Close() error {
	return s.inner.Close()
}

// --- Text Completion (OpenAI) ---
func (c *OpenAIClient) TextCompletion(ctx context.Context, req TextRequest) (TextResponse, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	openReq := openai.CompletionRequest{
		Model:       model,
		Prompt:      req.Prompt,
		Temperature: float32(req.Options.Temperature),
		TopP:        float32(req.Options.TopP),
		MaxTokens:   req.Options.MaxTokens,
	}
	resp, err := c.client.CreateCompletion(ctx, openReq)
	if err != nil {
		return TextResponse{}, err
	}
	var out TextResponse
	out.Usage = TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
	for _, choice := range resp.Choices {
		out.Choices = append(out.Choices, GenChoice{
			Content:      choice.Text,
			FinishReason: choice.FinishReason,
		})
	}
	return out, nil
}

func (c *OpenAIClient) TextCompletionStream(ctx context.Context, req TextRequest) (TextStream, error) {
	openReq := openai.CompletionRequest{
		Model:       req.Model,
		Prompt:      req.Prompt,
		Temperature: float32(req.Options.Temperature),
		TopP:        float32(req.Options.TopP),
		MaxTokens:   req.Options.MaxTokens,
		Stream:      true, // enable streaming
	}
	stream, err := c.client.CreateCompletionStream(ctx, openReq)
	if err != nil {
		return nil, err
	}
	return &openAITextStream{inner: stream}, nil
}

func (c *OpenAIClient) GenerateResponse(
	ctx context.Context,
	messages []Message,
	tools []llm_models.Tool,
) (Response, error) {
	// 1) Convert Tool → FunctionDef for the API
	funcDefs := make([]FunctionDef, len(tools))
	for i, t := range tools {
		funcDefs[i] = FunctionDef{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
		}
	}

	// 2) Build a ChatRequest that asks for tool-calling if needed
	req := ChatRequest{
		Model:        "", // empty → use client.defaultModel
		Messages:     messages,
		Functions:    funcDefs,
		FunctionCall: "auto",       // let the model decide
		Options:      GenOptions{}, // zero-values let the API defaults apply
	}

	// 3) Invoke the API
	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		return Response{}, err
	}
	if len(resp.Choices) == 0 {
		return Response{}, fmt.Errorf("no choices returned from OpenAI")
	}

	// 4) Take the first choice and map it to our Response
	choice := resp.Choices[0]
	out := Response{
		Content:       choice.Content,
		FunctionCalls: choice.FunctionCalls, // assuming GenChoice.FunctionCalls []*FunctionCall
		Usage:         resp.Usage,
	}
	return out, nil
}

type openAITextStream struct {
	inner *openai.CompletionStream
}

func (s *openAITextStream) Recv() (GenChoice, error) {
	resp, err := s.inner.Recv()
	if err != nil {
		return GenChoice{}, err
	}
	// Each stream chunk for text completion has a partial text in Choice.Text
	// (OpenAI uses a similar delta mechanism internally for completion streams)
	gen := GenChoice{Content: resp.Choices[0].Text}
	// No functionCall in plain text completions
	gen.FinishReason = resp.Choices[0].FinishReason
	return gen, nil
}
func (s *openAITextStream) Close() error {
	return s.inner.Close()
}

// --- Image Generation (OpenAI) ---
func (c *OpenAIClient) GenerateImage(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	model := req.Model
	if model == "" {
		model = "image-alpha-001" // e.g., DALL-E model (if not specified)
	}
	openReq := openai.ImageRequest{
		Prompt:         req.Prompt,
		N:              req.N,
		Size:           "",         // could set e.g. "1024x1024" if needed
		ResponseFormat: "b64_json", // request base64-encoded images for uniform handling
	}
	resp, err := c.client.CreateImage(ctx, openReq)
	if err != nil {
		return ImageResponse{}, err
	}
	var out ImageResponse
	// Map token usage if available (OpenAI may provide usage for image prompts)

	out.Usage = TokenUsage{
		//todo
		//PromptTokens:     resp.Usage.InputTokens,
		//CompletionTokens: resp.Usage.OutputTokens,
		//TotalTokens:      resp.Usage.TotalTokens,
	}
	for _, img := range resp.Data {
		data := []byte{}
		url := ""
		if img.B64JSON != "" {
			// Decode base64 string to bytes
			decoded, _ := base64.StdEncoding.DecodeString(img.B64JSON)
			data = decoded
		} else if img.URL != "" {
			url = img.URL
		}
		out.Images = append(out.Images, ImageData{Data: data, URL: url})
	}
	return out, nil
}
