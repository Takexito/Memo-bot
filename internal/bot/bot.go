package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xaenox/memo-bot/internal/classifier"
	"github.com/xaenox/memo-bot/internal/models"
	"github.com/xaenox/memo-bot/internal/storage"
	"go.uber.org/zap"
)

type MessageSender interface {
	SendMessage(chatID int64, text string) (tgbotapi.Message, error)
	DeleteMessage(chatID int64, messageID int) error
	SendReplyMessage(chatID int64, text string, replyToID int) (tgbotapi.Message, error)
}

func (b *Bot) handleAddCategory(ctx context.Context, message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())
	if len(args) == 0 {
		b.sendMessage(message.Chat.ID, "Please provide a category name.\nUsage: /addcategory <category_name>")
		return
	}

	category := strings.ToLower(args[0])
	if err := b.storage.AddCategory(ctx, message.From.ID, category); err != nil {
		b.logger.Error("Failed to add category",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID),
			zap.String("category", category))
		b.sendErrorMessage(message.Chat.ID, "Failed to add category. Please try again.")
		return
	}

	response := fmt.Sprintf("Added category: #%s", escapeMarkdown(category))
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ParseMode = "MarkdownV2"
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send add category confirmation",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID))
	}
}

func (b *Bot) handleRemoveCategory(ctx context.Context, message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())
	if len(args) == 0 {
		b.sendMessage(message.Chat.ID, "Please provide a category name.\nUsage: /removecategory <category_name>")
		return
	}

	category := strings.ToLower(args[0])
	if err := b.storage.RemoveCategory(ctx, message.From.ID, category); err != nil {
		b.logger.Error("Failed to remove category",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID),
			zap.String("category", category))
		b.sendErrorMessage(message.Chat.ID, "Failed to remove category. Please try again.")
		return
	}

	response := fmt.Sprintf("Removed category: #%s", escapeMarkdown(category))
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ParseMode = "MarkdownV2"
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send remove category confirmation",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID))
	}
}

func (b *Bot) handleMaxTags(ctx context.Context, message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())
	if len(args) == 0 {
		b.sendMessage(message.Chat.ID, "Please provide the maximum number of tags.\nUsage: /maxtags <number>")
		return
	}

	maxTags, err := strconv.Atoi(args[0])
	if err != nil || maxTags < 1 {
		b.sendMessage(message.Chat.ID, "Please provide a valid positive number.")
		return
	}

	if err := b.storage.UpdateUserMaxTags(ctx, message.From.ID, maxTags); err != nil {
		b.logger.Error("Failed to update max tags",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID),
			zap.Int("max_tags", maxTags))
		b.sendErrorMessage(message.Chat.ID, "Failed to update maximum tags. Please try again.")
		return
	}

	response := fmt.Sprintf("Updated maximum tags to: %d", maxTags)
	b.sendMessage(message.Chat.ID, response)
}

type TelegramMessageSender struct {
	api *tgbotapi.BotAPI
}

func NewTelegramMessageSender(api *tgbotapi.BotAPI) *TelegramMessageSender {
	return &TelegramMessageSender{api: api}
}

func (t *TelegramMessageSender) SendMessage(chatID int64, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	return t.api.Send(msg)
}

func (t *TelegramMessageSender) DeleteMessage(chatID int64, messageID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := t.api.Request(deleteMsg)
	return err
}

func (t *TelegramMessageSender) SendReplyMessage(chatID int64, text string, replyToID int) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToID
	return t.api.Send(msg)
}

const (
	errMsgGeneral    = "Sorry, something went wrong. Please try again later."
	errMsgSave       = "Sorry, I couldn't save your message. Please try again."
	errMsgRetrieval  = "Sorry, I couldn't retrieve the information. Please try again later."
	errMsgClassify   = "Sorry, I had trouble analyzing your message. Please try again."
	errMsgPermission = "Sorry, you don't have permission to do that."
)

type Bot struct {
	api        *tgbotapi.BotAPI
	sender     MessageSender
	storage    storage.Storage
	classifier *classifier.GPTClassifier
	logger     *zap.Logger
}

func New(token string, storage storage.Storage, classifier *classifier.GPTClassifier, logger *zap.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	sender := NewTelegramMessageSender(api)

	return &Bot{
		api:        api,
		sender:     sender,
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
	ctx := context.Background()

	// Handle commands
	if message.IsCommand() {
		b.handleCommand(ctx, message)
		return
	}

	// Get content from message
	content := message.Text
	if message.Caption != "" {
		content = message.Caption
	}

	// Send loading message
	loadingMsg, err := b.sender.SendReplyMessage(
		message.Chat.ID,
		"🤔 Analyzing your message...",
		message.MessageID,
	)
	if err != nil {
		b.logger.Error("Failed to send loading message",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID))
	}

	// Get GPT analysis response
	gptResponse := b.classifier.GetStructuredAnalysis(content, message.From.ID)

	// Delete loading message
	if err := b.sender.DeleteMessage(message.Chat.ID, loadingMsg.MessageID); err != nil {
		b.logger.Error("Failed to delete loading message",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID),
			zap.Int("message_id", loadingMsg.MessageID))
	}

	if gptResponse.Category == "" {
		b.logger.Error("Failed to get GPT analysis",
			zap.Int64("user_id", message.From.ID))
		b.sendErrorMessage(message.Chat.ID, errMsgClassify)
		return
	}

	// Update user metadata with new category and tags
	if err := b.storage.AddCategory(ctx, message.From.ID, gptResponse.Category); err != nil {
		b.logger.Error("Failed to save category",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID),
			zap.String("category", gptResponse.Category))
	}

	for _, tag := range gptResponse.Keywords {
		if err := b.storage.AddTag(ctx, message.From.ID, tag); err != nil {
			b.logger.Error("Failed to save tag",
				zap.Error(err),
				zap.Int64("user_id", message.From.ID),
				zap.String("tag", tag))
		}
	}

	// Send the response
	b.sendClassificationResponse(message.Chat.ID, message.MessageID, &gptResponse)
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	ctx := context.Background()

	// Initialize user in storage if needed
	user := &models.User{
		ID:         message.From.ID,
		LastUsedAt: time.Now(),
	}

	if err := b.storage.UpdateUser(ctx, user); err != nil {
		b.logger.Error("Failed to initialize user",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID))
	}

	welcome := `Welcome to MemoBot! 📝
I can help you organize your notes, images, and files with automatic classification.

Just send me any message, photo, or document, and I'll:
• Save it securely
• Classify it automatically
• Add relevant tags
• Make it easily searchable

Available commands:
/help - Show all commands
/tags - View your tags
/categories - View your categories
/history - View recent messages

Send me something to get started!`

	b.sendMessage(message.Chat.ID, welcome)
}

func (b *Bot) handleHelp(message *tgbotapi.Message) {
	help := `*Available Commands:*
/start \- Start the bot
/help \- Show this help message
/tags \- Show your tags
/categories \- Show your categories
/addcategory \- Add a new category
/removecategory \- Remove a category
/maxtags \- Set maximum number of tags per message
/history \- View recent messages

*Usage:*
/addcategory <category\_name>
/removecategory <category\_name>
/maxtags <number>

*I can process:*
• Text messages
• Photos with captions
• Documents
• Videos

Each message will be automatically:
• Classified into a category
• Tagged with relevant keywords
• Summarized for easy reference

*Tips:*
• Use hashtags in your messages for custom tags
• Long press any message to forward it to me
• Reply to my classification with corrections

Need help? Just send /help again\!`

	msg := tgbotapi.NewMessage(message.Chat.ID, help)
	msg.ParseMode = "MarkdownV2"
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send help message",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID))
	}
}

func (b *Bot) handleTags(ctx context.Context, message *tgbotapi.Message) {
	tags, err := b.storage.GetUserTags(ctx, message.From.ID)
	if err != nil {
		b.logger.Error("Failed to get user tags",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID))
		b.sendErrorMessage(message.Chat.ID, errMsgRetrieval)
		return
	}

	if len(tags) == 0 {
		b.sendMessage(message.Chat.ID, "You don't have any tags yet.")
		return
	}

	response := "*Your tags:*\n"
	for _, tag := range tags {
		formattedTag := "#" + strings.ReplaceAll(tag, " ", "_")
		response += escapeMarkdown(formattedTag) + "\n"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ParseMode = "MarkdownV2"
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send tags message",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID))
		b.sendErrorMessage(message.Chat.ID, errMsgGeneral)
	}
}

func (b *Bot) handleCategories(ctx context.Context, message *tgbotapi.Message) {
	categories, err := b.storage.GetUserCategories(ctx, message.From.ID)
	if err != nil {
		b.logger.Error("Failed to get user categories",
			zap.Error(err),
			zap.Int64("user_id", message.From.ID))
		b.sendErrorMessage(message.Chat.ID, errMsgRetrieval)
		return
	}

	if len(categories) == 0 {
		b.sendMessage(message.Chat.ID, "You don't have any categories yet.")
		return
	}

	response := "*Your categories:*\n"
	for _, category := range categories {
		formattedCategory := "#" + strings.ReplaceAll(category, " ", "_")
		response += escapeMarkdown(formattedCategory) + "\n"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ParseMode = "MarkdownV2"
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send categories message",
			zap.Error(err),
			zap.Int64("chat_id", message.Chat.ID))
		b.sendErrorMessage(message.Chat.ID, errMsgGeneral)
	}
}

// Add this helper function to escape special characters for MarkdownV2
func escapeMarkdown(text string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	escaped := text
	for _, char := range specialChars {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}
	return escaped
}

func (b *Bot) sendMessage(chatID int64, text string) {
	_, err := b.sender.SendMessage(chatID, text)
	if err != nil {
		b.logger.Error("Failed to send message",
			zap.Error(err),
			zap.Int64("chat_id", chatID),
			zap.String("text", text))
	}
}

func (b *Bot) sendErrorMessage(chatID int64, text string) {
	_, err := b.sender.SendMessage(chatID, "⚠️ "+text)
	if err != nil {
		b.logger.Error("Failed to send error message",
			zap.Error(err),
			zap.Int64("chat_id", chatID))
	}
}
func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.handleStart(message)
	case "help":
		b.handleHelp(message)
	case "tags":
		b.handleTags(ctx, message)
	case "categories":
		b.handleCategories(ctx, message)
	case "addcategory":
		b.handleAddCategory(ctx, message)
	case "removecategory":
		b.handleRemoveCategory(ctx, message)
	case "maxtags":
		b.handleMaxTags(ctx, message)
	default:
		b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
	}
}

func (b *Bot) sendClassificationResponse(chatID int64, replyToID int, response *classifier.GPTResponse) {
	// Format category and tags
	formattedCategory := "#" + strings.ReplaceAll(response.Category, " ", "_")
	formattedTags := make([]string, len(response.Keywords))
	for i, tag := range response.Keywords {
		formattedTags[i] = "#" + strings.ReplaceAll(tag, " ", "_")
	}

	// Escape special characters for Markdown
	formattedCategory = escapeMarkdown(formattedCategory)
	formattedSummary := escapeMarkdown(response.Summary)
	for i, tag := range formattedTags {
		formattedTags[i] = escapeMarkdown(tag)
	}

	// Build response message
	text := fmt.Sprintf("*Category:* %s\n", formattedCategory)
	if len(formattedTags) > 0 {
		text += fmt.Sprintf("*Tags:* %s\n", strings.Join(formattedTags, " "))
	}
	text += fmt.Sprintf("\n*Summary:* %s", formattedSummary)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "MarkdownV2"
	msg.ReplyToMessageID = replyToID

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send classification response",
			zap.Error(err),
			zap.Int64("chat_id", chatID))
	}
}
