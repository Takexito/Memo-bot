package main

import (
	"log"
	"path/filepath"

	"github.com/xaenox/memo-bot/internal/bot"
	"github.com/xaenox/memo-bot/internal/classifier"
	"github.com/xaenox/memo-bot/internal/storage"
	"github.com/xaenox/memo-bot/pkg/config"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	configPath := filepath.Join(".", "config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Fatal("Failed to load config",
			zap.Error(err),
			zap.String("path", configPath))
	}

	// Initialize storage
	var store storage.Storage
	if cfg.Database.UseInMemory {
		logger.Info("Using in-memory storage")
		store = storage.NewMemoryStorage()
	} else {
		logger.Info("Using PostgreSQL storage")
		store, err = storage.NewPostgresStorage(
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DBName,
			cfg.Database.SSLMode,
		)
		if err != nil {
			logger.Fatal("Failed to initialize storage", zap.Error(err))
		}
	}
	defer store.Close()

	// Initialize GPT classifier
	clf := classifier.NewGPTClassifier(
		cfg.OpenAI.APIKey,
			cfg.OpenAI.Model,
			cfg.OpenAI.MaxTokens,
			cfg.OpenAI.Temperature,
			cfg.Classifier.MaxTags,
			logger,
	)

	// Initialize and start bot
	b, err := bot.New(cfg.Telegram.Token, store, clf, logger)
	if err != nil {
		logger.Fatal("Failed to create bot", zap.Error(err))
	}

	logger.Info("Bot started")
	if err := b.Start(); err != nil {
		logger.Fatal("Bot error", zap.Error(err))
	}
} 