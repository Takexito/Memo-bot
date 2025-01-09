package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xaenox/memo-bot/internal/storage"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type GPTResponse struct {
	Category            string   `json:"category"`
	Keywords            []string `json:"keywords"`
	Summary             string   `json:"summary"`
	AttachmentsAnalysis string   `json:"attachments_analysis"`
	Links               []string `json:"links"`
}

type GPTClassifier struct {
	client      *openai.Client
	assistantID string
	model       string
	maxTokens   int
	temperature float64
	maxTags     int
	logger      *zap.Logger
	threads     map[int64]string // In-memory cache
	threadMutex sync.RWMutex
	storage     storage.ThreadStorage
}

func NewGPTClassifier(apiKey string, assistantID string, model string, maxTokens int, temperature float64, maxTags int, storage storage.ThreadStorage, logger *zap.Logger) *GPTClassifier {
	return &GPTClassifier{
		client:      openai.NewClient(apiKey),
		assistantID: assistantID,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		maxTags:     maxTags,
		logger:      logger,
		threads:     make(map[int64]string),
		threadMutex: sync.RWMutex{},
		storage:     storage,
	}
}

func (c *GPTClassifier) ClassifyContent(content string, userID int64) []string {
	ctx := context.Background()

	// Update the prompt to request structured response
	prompt := content

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
		return c.fallbackClassification(content, userID)
	}

	// Parse the structured response
	var gptResponse GPTResponse
	response := strings.TrimSpace(resp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(response), &gptResponse); err != nil {
		c.logger.Error("Failed to parse GPT response",
			zap.Error(err),
			zap.String("response", response))
		return c.fallbackClassification(content, userID)
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
func (c *GPTClassifier) fallbackClassification(content string, userID int64) []string {
	simpleClassifier := NewSimpleClassifier(0.7, c.maxTags)
	return simpleClassifier.ClassifyContent(content, userID)
}
func (c *GPTClassifier) getOrCreateThread(ctx context.Context, userID int64) (string, error) {
	// First check in-memory cache
	c.threadMutex.RLock()
	threadID, exists := c.threads[userID]
	c.threadMutex.RUnlock()

	if exists {
		// Verify thread still exists and is valid
		_, err := c.client.RetrieveThread(ctx, threadID)
		if err == nil {
			// Update last used timestamp
			if err := c.storage.UpdateThreadLastUsed(userID); err != nil {
				c.logger.Warn("Failed to update thread last used timestamp",
					zap.Error(err),
					zap.Int64("user_id", userID))
			}
			return threadID, nil
		}
		// Thread doesn't exist anymore, remove it from both cache and storage
		c.threadMutex.Lock()
		delete(c.threads, userID)
		c.threadMutex.Unlock()

		if err := c.storage.DeleteThread(userID); err != nil {
			c.logger.Error("Failed to delete invalid thread from storage",
				zap.Error(err),
				zap.Int64("user_id", userID))
		}
	}

	// Check storage if not in cache
	if !exists {
		storedThreadID, err := c.storage.GetThread(userID)
		if err != nil {
			c.logger.Error("Failed to get thread from storage",
				zap.Error(err),
				zap.Int64("user_id", userID))
		} else if storedThreadID != "" {
			// Verify stored thread is still valid
			_, err := c.client.RetrieveThread(ctx, storedThreadID)
			if err == nil {
				// Update cache and last used timestamp
				c.threadMutex.Lock()
				c.threads[userID] = storedThreadID
				c.threadMutex.Unlock()

				if err := c.storage.UpdateThreadLastUsed(userID); err != nil {
					c.logger.Warn("Failed to update thread last used timestamp",
						zap.Error(err),
						zap.Int64("user_id", userID))
				}
				return storedThreadID, nil
			}
			// Thread invalid, delete from storage
			if err := c.storage.DeleteThread(userID); err != nil {
				c.logger.Error("Failed to delete invalid thread from storage",
					zap.Error(err),
					zap.Int64("user_id", userID))
			}
		}
	}

	// Create new thread
	thread, err := c.client.CreateThread(ctx, openai.ThreadRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to create thread: %w", err)
	}

	// Store in both cache and persistent storage
	c.threadMutex.Lock()
	c.threads[userID] = thread.ID
	c.threadMutex.Unlock()

	if err := c.storage.SaveThread(userID, thread.ID); err != nil {
		c.logger.Error("Failed to save thread to storage",
			zap.Error(err),
			zap.Int64("user_id", userID),
			zap.String("thread_id", thread.ID))
	}

	return thread.ID, nil
}

func (c *GPTClassifier) GetStructuredAnalysis(content string, userID int64) GPTResponse {
	ctx := context.Background()

	// Create a thread
	thread, err := c.client.CreateThread(ctx, openai.ThreadRequest{})
	if err != nil {
		c.logger.Error("Failed to create thread", zap.Error(err))
		return c.fallbackResponse(content)
	}

	// Add a message to the thread
	_, err = c.client.CreateMessage(ctx, thread.ID, openai.MessageRequest{
		Role:    "user",
		Content: content,
	})
	if err != nil {
		c.logger.Error("Failed to create message", zap.Error(err))
		return c.fallbackResponse(content)
	}

	// Run the assistant
	run, err := c.client.CreateRun(ctx, thread.ID, openai.RunRequest{
		AssistantID: c.assistantID,
	})
	if err != nil {
		c.logger.Error("Failed to create run", zap.Error(err))
		return c.fallbackResponse(content)
	}

	// Poll for completion
	for {
		run, err = c.client.RetrieveRun(ctx, thread.ID, run.ID)
		if err != nil {
			c.logger.Error("Failed to retrieve run", zap.Error(err))
			return c.fallbackResponse(content)
		}

		if run.Status == "completed" {
			break
		}

		if run.Status == "failed" || run.Status == "expired" || run.Status == "cancelled" {
			c.logger.Error("Run failed", zap.String("status", string(run.Status)))
			return c.fallbackResponse(content)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Get the messages
	messages, err := c.client.ListMessage(ctx, thread.ID, nil, nil, nil, nil)
	if err != nil {
		c.logger.Error("Failed to list messages", zap.Error(err))
		return c.fallbackResponse(content)
	}

	// Get the last assistant message
	var lastAssistantMessage string
	for _, msg := range messages.Messages {
		if msg.Role == "assistant" {
			lastAssistantMessage = msg.Content[0].Text.Value
			break
		}
	}

	if lastAssistantMessage == "" {
		c.logger.Error("No assistant response found")
		return c.fallbackResponse(content)
	}

	// Parse the response
	var gptResponse GPTResponse
	if err := json.Unmarshal([]byte(lastAssistantMessage), &gptResponse); err != nil {
		c.logger.Error("Failed to parse assistant response",
			zap.Error(err),
			zap.String("response", lastAssistantMessage))
		return c.fallbackResponse(content)
	}

	// Clean up the thread (optional, depending on your needs)
	_, err = c.client.DeleteThread(ctx, thread.ID)
	if err != nil {
		c.logger.Warn("Failed to delete thread", zap.Error(err))
	}

	return gptResponse
}

func (c *GPTClassifier) fallbackResponse(content string) GPTResponse {
	return GPTResponse{
		Category: "general",
		Keywords: []string{"unclassified"},
		Summary:  content,
	}
}
