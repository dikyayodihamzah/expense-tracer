package vision

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type claudeProvider struct {
	client *anthropic.Client
}

func newClaude(apiKey string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required for Claude vision provider")
	}
	c := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(&http.Client{}),
	)
	return &claudeProvider{client: &c}, nil
}

func (cv *claudeProvider) ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)

	msg, err := cv.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewImageBlockBase64("image/jpeg", encoded),
				anthropic.NewTextBlock(prompt),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude vision API call: %w", err)
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude vision returned empty response")
	}

	text := msg.Content[0].Text
	if text == "" {
		return nil, fmt.Errorf("claude vision returned no text in first content block")
	}

	return ParseJSON([]byte(text))
}
