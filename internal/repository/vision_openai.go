package repository

import (
	"context"
	"encoding/base64"
	"fmt"

	openai "github.com/sashabaranov/go-openai"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

// openAIVision implements VisionProvider using the OpenAI GPT-4o vision API.
type openAIVision struct {
	client *openai.Client
}

// newOpenAIVision creates a new OpenAI vision provider.
func newOpenAIVision(apiKey string) (VisionProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai API key is required for OpenAI vision provider")
	}
	return &openAIVision{client: openai.NewClient(apiKey)}, nil
}

// ExtractExpense sends the image to GPT-4o and parses the response as an Expense.
func (ov *openAIVision) ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/jpeg;base64," + encoded

	resp, err := ov.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    dataURL,
							Detail: openai.ImageURLDetailHigh,
						},
					},
					{
						Type: openai.ChatMessagePartTypeText,
						Text: visionPrompt,
					},
				},
			},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("openai vision API call: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai vision returned empty choices")
	}

	text := resp.Choices[0].Message.Content
	if text == "" {
		return nil, fmt.Errorf("openai vision returned no text in first choice")
	}

	return ParseVisionJSON([]byte(text))
}
