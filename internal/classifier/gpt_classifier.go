package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type GPTClassifier struct {
	client      *openai.Client
	model       string
	maxTokens   int
	temperature float64
	maxTags     int
	logger      *zap.Logger
}

func NewGPTClassifier(apiKey string, model string, maxTokens int, temperature float64, maxTags int, logger *zap.Logger) *GPTClassifier {
	return &GPTClassifier{
		client:      openai.NewClient(apiKey),
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		maxTags:     maxTags,
		logger:      logger,
	}
}

func (c *GPTClassifier) ClassifyContent(content string) []string {
	ctx := context.Background()

	// Prepare the prompt for GPT
	prompt := fmt.Sprintf(`Analyze the following content and extract relevant tags/categories. 
The tags should be specific, relevant, and helpful for organizing and retrieving the content later.
Return the tags as a JSON array of strings, with at most %d tags.
Only return the JSON array, no other text.

Content: %s`, c.maxTags, content)

	// Create chat completion request
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens:   c.maxTokens,
			Temperature: float32(c.temperature),
		},
	)

	if err != nil {
		c.logger.Error("Failed to get GPT response", zap.Error(err))
		// Fallback to simple classification if GPT fails
		return c.fallbackClassification(content)
	}

	// Parse the response
	var tags []string
	response := strings.TrimSpace(resp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(response), &tags); err != nil {
		c.logger.Error("Failed to parse GPT response",
			zap.Error(err),
			zap.String("response", response))
		return c.fallbackClassification(content)
	}

	// Ensure we don't exceed maxTags
	if len(tags) > c.maxTags {
		tags = tags[:c.maxTags]
	}

	return tags
}

// Fallback to simple classification if GPT fails
func (c *GPTClassifier) fallbackClassification(content string) []string {
	simpleClassifier := NewSimpleClassifier(0.7, c.maxTags)
	return simpleClassifier.ClassifyContent(content)
}
