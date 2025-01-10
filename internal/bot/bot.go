package bot

import (
    "context"
    "fmt"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/google/uuid"
    "github.com/xaenox/memo-bot/internal/classifier"
    "github.com/xaenox/memo-bot/internal/models"
    "github.com/xaenox/memo-bot/internal/storage"
    "go.uber.org/zap"
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

    // Create a new message ID
    messageID := uuid.New().String()

    // Get GPT analysis response
    gptResponse := b.classifier.GetStructuredAnalysis(content, message.From.ID)

    // Create and save the message
    msg := &models.Message{
        ID:        messageID,
        UserID:    message.From.ID,
        Content:   content,
        Category:  gptResponse.Category,
        Tags:      gptResponse.Keywords,
        CreatedAt: time.Now(),
    }

    if err := b.storage.SaveMessage(ctx, msg); err != nil {
        b.logger.Error("Failed to save message",
            zap.Error(err),
            zap.String("message_id", messageID),
            zap.Int64("user_id", message.From.ID))
        b.sendErrorMessage(message.Chat.ID, "Sorry, I couldn't save your message. Please try again.")
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

	response := "*Your tags:*\n"
	for _, tag := range metadata.Tags {
		formattedTag := "#" + strings.ReplaceAll(tag, " ", "_")
		response += escapeMarkdown(formattedTag) + "\n"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ParseMode = "MarkdownV2"
	b.api.Send(msg)
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

	response := "*Your categories:*\n"
	for _, category := range metadata.Categories {
		formattedCategory := "#" + strings.ReplaceAll(category, " ", "_")
		response += escapeMarkdown(formattedCategory) + "\n"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	msg.ParseMode = "MarkdownV2"
	b.api.Send(msg)
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
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send message",
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
    case "history":
        b.handleHistory(ctx, message)
    default:
        b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
    }
}

func (b *Bot) handleHistory(ctx context.Context, message *tgbotapi.Message) {
    messages, err := b.storage.GetUserMessages(ctx, message.From.ID, 5, 0)
    if err != nil {
        b.logger.Error("Failed to get user messages",
            zap.Error(err),
            zap.Int64("user_id", message.From.ID))
        b.sendErrorMessage(message.Chat.ID, "Sorry, I couldn't retrieve your message history.")
        return
    }

    if len(messages) == 0 {
        b.sendMessage(message.Chat.ID, "You don't have any messages yet.")
        return
    }

    response := "*Your recent messages:*\n\n"
    for _, msg := range messages {
        response += fmt.Sprintf("*%s*\n", escapeMarkdown(msg.Category))
        response += fmt.Sprintf("_%s_\n", escapeMarkdown(msg.Content))
        if len(msg.Tags) > 0 {
            tags := make([]string, len(msg.Tags))
            for i, tag := range msg.Tags {
                tags[i] = "#" + escapeMarkdown(strings.ReplaceAll(tag, " ", "_"))
            }
            response += fmt.Sprintf("Tags: %s\n", strings.Join(tags, " "))
        }
        response += "\n"
    }

    msg := tgbotapi.NewMessage(message.Chat.ID, response)
    msg.ParseMode = "MarkdownV2"
    if _, err := b.api.Send(msg); err != nil {
        b.logger.Error("Failed to send history message",
            zap.Error(err),
            zap.Int64("chat_id", message.Chat.ID))
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

func (b *Bot) sendErrorMessage(chatID int64, text string) {
    msg := tgbotapi.NewMessage(chatID, "⚠️ "+text)
    if _, err := b.api.Send(msg); err != nil {
        b.logger.Error("Failed to send error message",
            zap.Error(err),
            zap.Int64("chat_id", chatID))
    }
}
