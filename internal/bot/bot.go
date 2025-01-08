package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xaenox/memo-bot/internal/classifier"
	"github.com/xaenox/memo-bot/internal/models"
	"github.com/xaenox/memo-bot/internal/storage"
	"go.uber.org/zap"
	"strings"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	storage    storage.Storage
	classifier classifier.Classifier
	logger     *zap.Logger
}

func New(token string, storage storage.Storage, classifier classifier.Classifier, logger *zap.Logger) (*Bot, error) {
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
		case "list":
			b.handleList(message)
		default:
			b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
		}
		return
	}

	// Process the message content
	note := &models.Note{
		UserID:  message.From.ID,
		Content: message.Text,
		Type:    models.TextContent,
	}

	// Handle different types of content
	switch {
	case message.Photo != nil:
		note.Type = models.ImageContent
		note.FileID = message.Photo[len(message.Photo)-1].FileID
		if message.Caption != "" {
			note.Content = message.Caption
		}
	case message.Video != nil:
		note.Type = models.VideoContent
		note.FileID = message.Video.FileID
		if message.Caption != "" {
			note.Content = message.Caption
		}
	case message.Document != nil:
		note.Type = models.DocumentContent
		note.FileID = message.Document.FileID
		if message.Caption != "" {
			note.Content = message.Caption
		}
	}

	// Classify content and get tags
	note.Tags = b.classifier.ClassifyContent(note.Content)

	// Store the note
	if err := b.storage.CreateNote(note); err != nil {
		b.logger.Error("Failed to store note", zap.Error(err))
		b.sendMessage(message.Chat.ID, "Sorry, failed to save your note. Please try again later.")
		return
	}

	// Send confirmation with tags
	response := fmt.Sprintf("Note saved with tags: %s", strings.Join(note.Tags, ", "))
	b.sendMessage(message.Chat.ID, response)
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
/list - List your recent notes
/list #tag - List notes with specific tag

You can send:
- Text messages
- Photos with captions
- Documents
- Videos

I'll automatically classify your content and add relevant tags!`

	b.sendMessage(message.Chat.ID, help)
}

func (b *Bot) handleList(message *tgbotapi.Message) {
	args := strings.Fields(message.Text)
	var notes []*models.Note
	var err error

	if len(args) > 1 && strings.HasPrefix(args[1], "#") {
		// List notes by tag
		tag := strings.TrimPrefix(args[1], "#")
		notes, err = b.storage.GetNotesByTag(message.From.ID, tag)
	} else {
		// List all notes
		notes, err = b.storage.GetNotesByUserID(message.From.ID)
	}

	if err != nil {
		b.logger.Error("Failed to get notes", zap.Error(err))
		b.sendMessage(message.Chat.ID, "Sorry, failed to retrieve your notes. Please try again later.")
		return
	}

	if len(notes) == 0 {
		b.sendMessage(message.Chat.ID, "No notes found.")
		return
	}

	// Format and send notes
	var response strings.Builder
	for i, note := range notes {
		if i >= 10 {
			response.WriteString("\n... and more notes.")
			break
		}

		response.WriteString(fmt.Sprintf("\n📝 %s\nTags: %s\n",
			note.Content,
			strings.Join(note.Tags, ", ")))
	}

	b.sendMessage(message.Chat.ID, response.String())
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send message",
			zap.Error(err),
			zap.Int64("chat_id", chatID))
	}
}
