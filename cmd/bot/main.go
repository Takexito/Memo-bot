package main

import (
	"github.com/xaenox/memo-bot/internal/bot"
	"github.com/xaenox/memo-bot/internal/classifier"
	"github.com/xaenox/memo-bot/internal/storage"
	"github.com/xaenox/memo-bot/pkg/config"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err), zap.String("path", "config.yaml"))
	}

	// Initialize storage
	var store storage.Storage
	if cfg.Database.UseInMemory {
		logger.Info("Using in-memory storage")
		store = storage.NewMemoryStorage()
	} else {
		logger.Info("Using PostgreSQL storage")
		dbConfig := storage.DatabaseConfig{
			Host:        cfg.Database.Host,
			Port:        cfg.Database.Port,
			User:        cfg.Database.User,
			Password:    cfg.Database.Password,
			DBName:      cfg.Database.DBName,
			SSLMode:     cfg.Database.SSLMode,
			UseInMemory: cfg.Database.UseInMemory,
		}
		store, err = storage.NewPostgresStorage(dbConfig, logger)
		if err != nil {
			logger.Fatal("Failed to initialize storage", zap.Error(err))
		}
	}
	defer store.Close()

	// Initialize classifier with storage
	clf := classifier.NewGPTClassifier(
		cfg.OpenAI.APIKey,
		cfg.OpenAI.AssistantID,
		cfg.OpenAI.Model,
		cfg.OpenAI.MaxTokens,
		cfg.OpenAI.Temperature,
		cfg.Classifier.MaxTags,
		store,
		logger,
	)

	// Initialize bot
	b, err := bot.New(cfg.Telegram.Token, store, clf, logger)
	if err != nil {
		logger.Fatal("Failed to create bot", zap.Error(err))
	}

	// Start the bot
	if err := b.Start(); err != nil {
		logger.Fatal("Bot error", zap.Error(err))
	}
}
