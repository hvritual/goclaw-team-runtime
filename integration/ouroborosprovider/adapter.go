package ouroborosprovider

import (
	"context"
	"errors"

	"github.com/smallnest/goclaw/ouroboros"
	"github.com/smallnest/goclaw/providers"
)

// Adapter lets the Go-native Ouroboros runtime use the same configured
// GoClaw provider as chat. With codex-app-server this preserves ChatGPT
// subscription authentication without reading or copying OAuth material.
type Adapter struct {
	provider providers.Provider
}

func New(provider providers.Provider) (*Adapter, error) {
	if provider == nil {
		return nil, errors.New("provider is required")
	}
	return &Adapter{provider: provider}, nil
}

func (a *Adapter) Generate(
	ctx context.Context,
	request ouroboros.ModelRequest,
) (ouroboros.ModelResponse, error) {
	options := []providers.ChatOption{
		providers.WithTemperature(request.Temperature),
		providers.WithMaxTokens(request.MaxTokens),
	}
	if request.Model != "" {
		options = append(options, providers.WithModel(request.Model))
	}
	response, err := a.provider.Chat(ctx, []providers.Message{
		{Role: "system", Content: request.System},
		{Role: "user", Content: request.User},
	}, nil, options...)
	if err != nil {
		return ouroboros.ModelResponse{}, err
	}
	return ouroboros.ModelResponse{
		Content: response.Content,
		Model:   request.Model,
		Usage: ouroboros.ModelUsage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
			TotalTokens:  response.Usage.TotalTokens,
			Calls:        1,
		},
	}, nil
}
