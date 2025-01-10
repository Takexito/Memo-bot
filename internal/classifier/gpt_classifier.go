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
	// Get the structured analysis
	analysis := c.GetStructuredAnalysis(content, userID)

	// Combine category and keywords for tags
	tags := make([]string, 0, len(analysis.Keywords)+1)
	tags = append(tags, strings.ToLower(analysis.Category))
	tags = append(tags, analysis.Keywords...)

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

	// Log the initial request
	c.logger.Info("Starting GPT analysis",
		zap.Int64("user_id", userID),
		zap.String("content", content))

	// Create a thread
	thread, err := c.client.CreateThread(ctx, openai.ThreadRequest{})
	if err != nil {
		c.logger.Error("Failed to create thread",
			zap.Error(err),
			zap.Int64("user_id", userID))
		return c.fallbackResponse(content)
	}
	c.logger.Debug("Created thread",
		zap.String("thread_id", thread.ID),
		zap.Int64("user_id", userID))

	// Add a message to the thread
	message, err := c.client.CreateMessage(ctx, thread.ID, openai.MessageRequest{
		Role:    "user",
		Content: content,
	})
	if err != nil {
		c.logger.Error("Failed to create message",
			zap.Error(err),
			zap.String("thread_id", thread.ID),
			zap.Int64("user_id", userID))
		return c.fallbackResponse(content)
	}
	c.logger.Debug("Created message",
		zap.String("message_id", message.ID),
		zap.String("thread_id", thread.ID),
		zap.Int64("user_id", userID))

	// Run the assistant
	run, err := c.client.CreateRun(ctx, thread.ID, openai.RunRequest{
		AssistantID: c.assistantID,
	})
	if err != nil {
		c.logger.Error("Failed to create run",
			zap.Error(err),
			zap.String("thread_id", thread.ID),
			zap.String("assistant_id", c.assistantID),
			zap.Int64("user_id", userID))
		return c.fallbackResponse(content)
	}
	c.logger.Debug("Created run",
		zap.String("run_id", run.ID),
		zap.String("thread_id", thread.ID),
		zap.Int64("user_id", userID))

	// Poll for completion
	startTime := time.Now()
	for {
		run, err = c.client.RetrieveRun(ctx, thread.ID, run.ID)
		if err != nil {
			c.logger.Error("Failed to retrieve run",
				zap.Error(err),
				zap.String("run_id", run.ID),
				zap.String("thread_id", thread.ID),
				zap.Int64("user_id", userID))
			return c.fallbackResponse(content)
		}

		if run.Status == "completed" {
			c.logger.Debug("Run completed",
				zap.String("run_id", run.ID),
				zap.Duration("duration", time.Since(startTime)),
				zap.Int64("user_id", userID))
			break
		}

		if run.Status == "failed" || run.Status == "expired" || run.Status == "cancelled" {
			c.logger.Error("Run failed",
				zap.String("status", string(run.Status)),
				zap.String("run_id", run.ID),
				zap.String("thread_id", thread.ID),
				zap.Int64("user_id", userID))
			return c.fallbackResponse(content)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Get the messages
	messages, err := c.client.ListMessage(ctx, thread.ID, nil, nil, nil, nil, nil)
	if err != nil {
		c.logger.Error("Failed to list messages",
			zap.Error(err),
			zap.String("thread_id", thread.ID),
			zap.Int64("user_id", userID))
		return c.fallbackResponse(content)
	}

	// Get the last assistant message
	var lastAssistantMessage string
	for _, msg := range messages.Messages {
		if msg.Role == "assistant" {
			lastAssistantMessage = msg.Content[0].Text.Value
			c.logger.Debug("Received assistant response",
				zap.String("message_id", msg.ID),
				zap.String("thread_id", thread.ID),
				zap.String("response", lastAssistantMessage),
				zap.Int64("user_id", userID))
			break
		}
	}

	if lastAssistantMessage == "" {
		c.logger.Error("No assistant response found",
			zap.String("thread_id", thread.ID),
			zap.Int64("user_id", userID))
		return c.fallbackResponse(content)
	}

	// Parse the response
	var gptResponse GPTResponse
	if err := json.Unmarshal([]byte(lastAssistantMessage), &gptResponse); err != nil {
		c.logger.Error("Failed to parse assistant response",
			zap.Error(err),
			zap.String("response", lastAssistantMessage),
			zap.String("thread_id", thread.ID),
			zap.Int64("user_id", userID))
		return c.fallbackResponse(content)
	}

	c.logger.Info("Successfully completed GPT analysis",
		zap.Any("response", gptResponse),
		zap.String("thread_id", thread.ID),
		zap.Duration("total_duration", time.Since(startTime)),
		zap.Int64("user_id", userID))

	// Clean up the thread
	_, err = c.client.DeleteThread(ctx, thread.ID)
	if err != nil {
		c.logger.Warn("Failed to delete thread",
			zap.Error(err),
			zap.String("thread_id", thread.ID),
			zap.Int64("user_id", userID))
	}

	return gptResponse
}

func (c *GPTClassifier) fallbackResponse(content string) GPTResponse {
	return GPTResponse{
		Category: "general",
		Keywords: []string{"unclassified"},
		Summary:  "I received your message but I'm having trouble analyzing it right now.",
		Links:    []string{},
	}
}

func (c *GPTClassifier) formatUserResponse(analysis GPTResponse) string {
	var response strings.Builder

	// Add summary
	response.WriteString(analysis.Summary)
	response.WriteString("\n\n")

	// Add category
	response.WriteString("Category: #")
	response.WriteString(strings.ToLower(strings.ReplaceAll(analysis.Category, " ", "_")))
	response.WriteString("\n")

	// Add keywords as hashtags
	if len(analysis.Keywords) > 0 {
		response.WriteString("Tags: ")
		for i, keyword := range analysis.Keywords {
			response.WriteString("#")
			response.WriteString(strings.ToLower(strings.ReplaceAll(keyword, " ", "_")))
			if i < len(analysis.Keywords)-1 {
				response.WriteString(" ")
			}
		}
		response.WriteString("\n")
	}

	// Add links if present
	if len(analysis.Links) > 0 {
		response.WriteString("\nLinks found:\n")
		for _, link := range analysis.Links {
			response.WriteString("• ")
			response.WriteString(link)
			response.WriteString("\n")
		}
	}

	return response.String()
}

func (c *GPTClassifier) GetUserResponse(content string, userID int64) string {
	analysis := c.GetStructuredAnalysis(content, userID)
	return c.formatUserResponse(analysis)
}
