package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xaenox/memo-bot/internal/classifier"
	"github.com/xaenox/memo-bot/internal/storage"
	"go.uber.org/zap"
	"strings"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	storage    storage.Storage
	classifier *classifier.GPTClassifier
	logger     *zap.Logger
}

func New(token string, storage storage.Storage, classifier *classifier.GPTClassifier, logger *zap.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return &Bot{
		api:        api,
		storage:    storage,
		classifier: classifier,
		logger:     logger,
	}, nil
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		go b.handleMessage(update.Message)
	}

	return nil
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	// Handle commands
	if message.IsCommand() {
		switch message.Command() {
		case "start":
			b.handleStart(message)
		case "help":
			b.handleHelp(message)
		case "tags":
			b.handleTags(message)
		case "categories":
			b.handleCategories(message)
		default:
			b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
		}
		return
	}

	// Get content from message
	content := message.Text
	if message.Caption != "" {
		content = message.Caption
	}

	// Get GPT analysis response
	gptResponse := b.classifier.GetStructuredAnalysis(content, message.From.ID)

	// Save category and tags to user metadata
	if err := b.storage.AddUserCategory(message.From.ID, gptResponse.Category); err != nil {
		b.logger.Error("Failed to save category",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID),
			zap.String("category", gptResponse.Category))
	}

	// Save each keyword as a tag
	for _, tag := range gptResponse.Keywords {
		if err := b.storage.AddUserTag(message.From.ID, tag); err != nil {
			b.logger.Error("Failed to save tag",
				zap.Error(err),
				zap.Int64("user_id", message.From.ID),
				zap.String("tag", tag))
		}
	}

	// Format and send the response
	// Add # to each keyword
	formattedTags := make([]string, len(gptResponse.Keywords))
	for i, tag := range gptResponse.Keywords {
		formattedTags[i] = "#" + tag
	}

	response := fmt.Sprintf("*Category:* %s\n*Tags:* %s\n\n*Summary:* %s",
		gptResponse.Category,
		strings.Join(formattedTags, ", "),
		gptResponse.Summary)

	// Send the formatted response with Markdown and reply to the original message
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ParseMode = "Markdown"
	msg.ReplyToMessageID = message.MessageID
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send response",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID))
	}
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	welcome := `Welcome to MemoBot! 📝
I can help you organize your notes, images, and files with automatic classification.

Just send me any message, photo, or document, and I'll save it with relevant tags.
Use /help to see all available commands.`

	b.sendMessage(message.Chat.ID, welcome)
}

func (b *Bot) handleHelp(message *tgbotapi.Message) {
	help := `Available commands:
/start - Start the bot
/help - Show this help message
/tags - Show your tags
/categories - Show your categories

You can send:
- Text messages
- Photos with captions
- Documents
- Videos

I'll automatically classify your content and add relevant tags!`

	b.sendMessage(message.Chat.ID, help)
}

func (b *Bot) handleTags(message *tgbotapi.Message) {
	metadata, err := b.storage.GetUserMetadata(message.From.ID)
	if err != nil {
		b.logger.Error("Failed to get user metadata",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID))
		b.sendMessage(message.Chat.ID, "Sorry, failed to retrieve your tags. Please try again later.")
		return
	}

	if len(metadata.Tags) == 0 {
		b.sendMessage(message.Chat.ID, "You don't have any tags yet.")
		return
	}

	response := "Your tags:\n"
	for _, tag := range metadata.Tags {
		response += fmt.Sprintf("#%s\n", tag)
	}

	b.sendMessage(message.Chat.ID, response)
}

func (b *Bot) handleCategories(message *tgbotapi.Message) {
	metadata, err := b.storage.GetUserMetadata(message.From.ID)
	if err != nil {
		b.logger.Error("Failed to get user metadata",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID))
		b.sendMessage(message.Chat.ID, "Sorry, failed to retrieve your categories. Please try again later.")
		return
	}

	if len(metadata.Categories) == 0 {
		b.sendMessage(message.Chat.ID, "You don't have any categories yet.")
		return
	}

	response := "Your categories:\n"
	for _, category := range metadata.Categories {
		response += fmt.Sprintf("📁 %s\n", category)
	}

	b.sendMessage(message.Chat.ID, response)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send message",
			zap.Error(err),
			zap.Int64("chat_id", chatID))
	}
}
