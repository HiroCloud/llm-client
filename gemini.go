package llm_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/HiroCloud/llm-client/llm_models"
	genai "google.golang.org/genai"
) // Google GenAI SDK

// GoogleClient implements AIClient for Google Gemini/Vertex AI models.
type GoogleClient struct {
	client       *genai.Client // underlying GenAI client (configured for Gemini or Vertex AI)
	defaultModel string        // default model name/ID to use if none specified
}

func NewGC() (AIClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	ctx := context.Background()

	// create the client
	// 💡 FIX: The NewClient function takes a context AND an option list.
	// The API key is passed using option.WithAPIKey()
	cc := &genai.ClientConfig{}
	cc.APIKey = apiKey
	c, err := genai.NewClient(ctx, cc)
	if err != nil {
		return nil, err
	}

	return &GoogleClient{
		client:       c,
		defaultModel: "gemini-3-flash-preview", // Recommending the latest flash model
	}, nil
}

// --- Chat Completion (Google Gemini) ---
func (c *GoogleClient) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	// Convert Messages to genai.Content parts.
	// We provide the conversation as a sequence of parts (text segments).
	// Since the Gemini API doesn't have explicit roles in the request, we concatenate
	// system and user messages in order as input parts. (Assistant responses in history
	// can also be included as context.)
	var contentParts []*genai.Part
	for _, m := range req.Messages {
		// We only include user/system content as prompt context for the model.
		// (The assistant's prior messages should also be included as context if present.)
		part := genai.NewPartFromText(m.Content)
		contentParts = append(contentParts, part)
	}
	contents := []*genai.Content{{Parts: contentParts}}
	// Prepare config with generation parameters
	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(float32(req.Options.Temperature)),
		TopP:            genai.Ptr(float32(req.Options.TopP)),
		MaxOutputTokens: int32(req.Options.MaxTokens),
	}
	// Call Google's content generation API (for chat or prompt completion)
	result, err := c.client.Models.GenerateContent(ctx, model, contents, genConfig)
	if err != nil {
		return ChatResponse{}, err
	}
	// Map GenerateContentResponse to ChatResponse
	var out ChatResponse
	if usage := result.UsageMetadata; usage != nil {
		out.Usage = TokenUsage{
			PromptTokens:     int(usage.PromptTokenCount),
			CompletionTokens: int(usage.CandidatesTokenCount), // tokens in output
			TotalTokens:      int(usage.TotalTokenCount),
		}
	}
	// The response may contain multiple candidates (if requested)
	for _, cand := range result.Candidates {
		// Each Candidate has Content (which may be composed of Parts)
		text := ""
		if cand.Content != nil {
			// Concatenate all text parts of the content
			for _, part := range cand.Content.Parts {
				text += part.Text
			}
		}
		genChoice := GenChoice{Content: text}
		// If the model returned a function call (Gemini supports tool invocation), capture it
		if cand.Content != nil && len(cand.Content.Parts) > 0 && cand.Content.Parts[0].FunctionCall != nil {
			args, err := json.Marshal(cand.Content.Parts[0].FunctionCall.Args)
			if err != nil {
				return ChatResponse{}, err
			}
			genChoice.FunctionCalls = []*FunctionCall{
				{
					Name:      cand.Content.Parts[0].FunctionCall.Name,
					Arguments: string(args), // (Arguments would be in the text or a structured field if available)
				},
			}
		}
		// Finish reason: Gemini provides a finish reason in Candidate.FinishMessage
		genChoice.FinishReason = cand.FinishMessage
		out.Choices = append(out.Choices, genChoice)
	}
	return out, nil
}

func (c *GoogleClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (ChatStream, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	var contentParts []*genai.Part
	for _, m := range req.Messages {
		contentParts = append(contentParts, genai.NewPartFromText(m.Content))
	}
	contents := []*genai.Content{{Parts: contentParts}}
	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(float32(req.Options.Temperature)),
		TopP:            genai.Ptr(float32(req.Options.TopP)),
		MaxOutputTokens: int32(req.Options.MaxTokens),
	}
	streamIter := c.client.Models.GenerateContentStream(ctx, model, contents, genConfig)
	next, stop := iter.Pull2(streamIter)
	return &googleChatStream{next: next, stop: stop, ctx: ctx}, nil
}

type googleChatStream struct {
	ctx  context.Context
	next func() (*genai.GenerateContentResponse, error, bool)
	stop func()
}

func (s *googleChatStream) Recv() (GenChoice, error) {
	result, err, f := s.next()
	if err != nil {
		return GenChoice{}, err
	}
	if !f {
		return GenChoice{}, io.EOF
	}
	// Each streamed GenerateContentResponse represents the next chunk of output
	gen := GenChoice{}
	if len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
		// Take the first candidate's text from parts
		for _, part := range result.Candidates[0].Content.Parts {
			gen.Content += part.Text
		}

		// If a function call part is present in this chunk:
		if len(result.Candidates[0].Content.Parts) > 0 && result.Candidates[0].Content.Parts[0].FunctionCall != nil {
			args, err := json.Marshal(result.Candidates[0].Content.Parts[0].FunctionCall.Args)
			if err != nil {
				return GenChoice{}, err
			}
			gen.FunctionCalls = []*FunctionCall{
				{
					Name:      result.Candidates[0].Content.Parts[0].FunctionCall.Name,
					Arguments: string(args), // (Arguments would be in the text or a structured field if available)
				},
			}
		}
		// Gemini doesn't provide an explicit finish reason per chunk; stream ends when complete
	}
	return gen, nil
}

func (s *googleChatStream) Close() error {
	s.stop()
	// No explicit close needed for google iterator (will end when Next is done)
	return nil
}

// --- Text Completion (Google) ---
func (c *GoogleClient) TextCompletion(ctx context.Context, req TextRequest) (TextResponse, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	parts := []*genai.Part{genai.NewPartFromText(req.Prompt)}
	contents := []*genai.Content{{Parts: parts}}
	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(float32(req.Options.Temperature)),
		TopP:            genai.Ptr(float32(req.Options.TopP)),
		MaxOutputTokens: int32(req.Options.MaxTokens),
	}
	result, err := c.client.Models.GenerateContent(ctx, model, contents, genConfig)
	if err != nil {
		return TextResponse{}, err
	}
	var out TextResponse
	if usage := result.UsageMetadata; usage != nil {
		out.Usage = TokenUsage{
			PromptTokens:     int(usage.PromptTokenCount),
			CompletionTokens: int(usage.CandidatesTokenCount),
			TotalTokens:      int(usage.TotalTokenCount),
		}
	}
	for _, cand := range result.Candidates {
		text := ""
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				text += part.Text
			}
		}
		out.Choices = append(out.Choices, GenChoice{
			Content:      text,
			FinishReason: cand.FinishMessage,
		})
	}
	return out, nil
}

func (c *GoogleClient) TextCompletionStream(ctx context.Context, req TextRequest) (TextStream, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	contents := []*genai.Content{{Parts: []*genai.Part{genai.NewPartFromText(req.Prompt)}}}
	genConfig := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(float32(req.Options.Temperature)),
		TopP:            genai.Ptr(float32(req.Options.TopP)),
		MaxOutputTokens: int32(req.Options.MaxTokens),
	}
	streamIter := c.client.Models.GenerateContentStream(ctx, model, contents, genConfig)
	next, stop := iter.Pull2(streamIter)
	return &googleTextStream{next: next, stop: stop, ctx: ctx}, nil
}

type googleTextStream struct {
	next func() (*genai.GenerateContentResponse, error, bool)
	stop func()
	ctx  context.Context
}

func (s *googleTextStream) Recv() (GenChoice, error) {
	result, err, f := s.next()
	if err != nil {
		return GenChoice{}, err
	}
	if !f {
		return GenChoice{}, io.EOF
	}
	gen := GenChoice{}
	if len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
		for _, part := range result.Candidates[0].Content.Parts {
			gen.Content += part.Text
		}
		gen.FinishReason = result.Candidates[0].FinishMessage
	}
	return gen, nil
}
func (s *googleTextStream) Close() error {
	s.stop()
	return nil
}

// --- Image Generation (Google Gemini) ---
func (c *GoogleClient) GenerateImage(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	genConfig := &genai.GenerateImagesConfig{
		NumberOfImages: int32(req.N),
		// Optionally, set other fields like AspectRatio or Size if needed.
	}
	result, err := c.client.Models.GenerateImages(ctx, model, req.Prompt, genConfig)
	if err != nil {
		return ImageResponse{}, err
	}
	var out ImageResponse
	// (Gemini may not provide token usage for images; if it did, map it here)
	for _, img := range result.GeneratedImages {
		data := []byte{}
		url := ""
		if img.Image != nil {
			if len(img.Image.ImageBytes) > 0 {
				data = img.Image.ImageBytes
			} else if img.Image.GCSURI != "" {
				url = img.Image.GCSURI
			}
		}
		out.Images = append(out.Images, ImageData{Data: data, URL: url})
	}
	return out, nil
}

func (c *GoogleClient) GenerateResponse(
	ctx context.Context,
	messages []Message,
	tools []llm_models.Tool,
) (Response, error) {
	// 1) Convert Tool → FuncDef (the JSON schema part only)
	funcDefs := make([]FunctionDef, len(tools))
	for i, t := range tools {
		funcDefs[i] = FunctionDef{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
		}
	}

	// 2) Reuse your ChatCompletion impl (it converts FuncDef → genai schemas)
	chatReq := ChatRequest{
		Model:        "", // let defaultModel apply
		Messages:     messages,
		Functions:    funcDefs, // your GoogleChatConversion will use these
		FunctionCall: "auto",   // or "" if you want default
		Options:      GenOptions{},
	}
	chatResp, err := c.ChatCompletion(ctx, chatReq)
	if err != nil {
		return Response{}, err
	}
	if len(chatResp.Choices) == 0 {
		return Response{}, fmt.Errorf("no choices returned from Google")
	}

	// 3) Map first GenChoice → Response
	choice := chatResp.Choices[0]
	out := Response{
		Content:       choice.Content,
		FunctionCalls: choice.FunctionCalls,
		Usage:         chatResp.Usage,
	}
	return out, nil
}
