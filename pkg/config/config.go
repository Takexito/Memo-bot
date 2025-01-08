package config

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Telegram struct {
		Token string
	}
	Database struct {
		Host     string
		 Port     int
		 User     string
		 Password string
		 DBName   string
		 SSLMode  string
		 URL      string // For Vercel deployment
		 UseInMemory bool `mapstructure:"use_in_memory"` // Flag for using in-memory database
	}
	Classifier struct {
		MinConfidence float64 `mapstructure:"min_confidence"`
		MaxTags      int     `mapstructure:"max_tags"`
	}
	OpenAI struct {
		APIKey      string `mapstructure:"api_key"`
		Model       string `mapstructure:"model"`
		MaxTokens   int    `mapstructure:"max_tokens"`
		Temperature float64
	}
}

func LoadConfig(path string) (*Config, error) {
	var config Config

	// Try loading from file first
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err == nil {
		if err := viper.Unmarshal(&config); err != nil {
			return nil, err
		}
	}

	// Override with environment variables if they exist
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		config.Telegram.Token = token
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.Database.URL = dbURL
	}

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}

	if maxTags := os.Getenv("MAX_TAGS"); maxTags != "" {
		if val, err := strconv.Atoi(maxTags); err == nil {
			config.Classifier.MaxTags = val
		}
	}

	if minConf := os.Getenv("MIN_CONFIDENCE"); minConf != "" {
		if val, err := strconv.ParseFloat(minConf, 64); err == nil {
			config.Classifier.MinConfidence = val
		}
	}

	if useInMemory := os.Getenv("USE_IN_MEMORY_DB"); useInMemory != "" {
		config.Database.UseInMemory = useInMemory == "true"
	}

	// Set defaults for OpenAI if not set
	if config.OpenAI.Model == "" {
		config.OpenAI.Model = "gpt-3.5-turbo"
	}
	if config.OpenAI.MaxTokens == 0 {
		config.OpenAI.MaxTokens = 150
	}
	if config.OpenAI.Temperature == 0 {
		config.OpenAI.Temperature = 0.3
	}

	return &config, nil
} 