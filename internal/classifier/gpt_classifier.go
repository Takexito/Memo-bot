package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type GPTResponse struct {
	Category            string   `json:"category"`
	Keywords            []string `json:"keywords"`
	Summary            string   `json:"summary"`
	AttachmentsAnalysis string   `json:"attachments_analysis"`
	Links              []string `json:"links"`
}

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

	// Update the prompt to request structured response
	prompt := fmt.Sprintf(`Analyze the following content and provide a structured analysis with:
- A single main category
- Relevant keywords/tags (max %d)
- A brief summary
- Analysis of any attachments mentioned
- Any URLs/links found in the content

Return the response as a JSON object with this structure:
{
    "category": "main_category",
    "keywords": ["keyword1", "keyword2", ...],
    "summary": "brief_summary",
    "attachments_analysis": "analysis_of_attachments",
    "links": ["url1", "url2", ...]
}

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
		return c.fallbackClassification(content)
	}

	// Parse the structured response
	var gptResponse GPTResponse
	response := strings.TrimSpace(resp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(response), &gptResponse); err != nil {
		c.logger.Error("Failed to parse GPT response",
			zap.Error(err),
			zap.String("response", response))
		return c.fallbackClassification(content)
	}

	// Combine category and keywords for tags
	tags := make([]string, 0, len(gptResponse.Keywords)+1)
	tags = append(tags, strings.ToLower(gptResponse.Category))
	tags = append(tags, gptResponse.Keywords...)

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
func (c *GPTClassifier) GetStructuredAnalysis(content string) GPTResponse {
	ctx := context.Background()

	prompt := fmt.Sprintf(`Analyze the following content and provide a structured analysis with:
- A single main category
- Relevant keywords/tags (max %d)
- A brief summary
- Analysis of any attachments mentioned
- Any URLs/links found in the content

Return the response as a JSON object with this structure:
{
    "category": "main_category",
    "keywords": ["keyword1", "keyword2", ...],
    "summary": "brief_summary",
    "attachments_analysis": "analysis_of_attachments",
    "links": ["url1", "url2", ...]
}

Content: %s`, c.maxTags, content)

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
		return GPTResponse{
			Category: "general",
			Keywords: []string{"unclassified"},
			Summary:  content,
		}
	}

	var gptResponse GPTResponse
	response := strings.TrimSpace(resp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(response), &gptResponse); err != nil {
		c.logger.Error("Failed to parse GPT response",
			zap.Error(err),
			zap.String("response", response))
		return GPTResponse{
			Category: "general",
			Keywords: []string{"unclassified"},
			Summary:  content,
		}
	}

	return gptResponse
}
