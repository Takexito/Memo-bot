package config

import (
	"fmt"
	"net/url"
	"strings"
	"github.com/spf13/viper"
)

type Config struct {
	Telegram   TelegramConfig   `mapstructure:"telegram"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Classifier ClassifierConfig `mapstructure:"classifier"`
	OpenAI     OpenAIConfig     `mapstructure:"openai"`
}

type TelegramConfig struct {
	Token string `mapstructure:"token"`
}

type DatabaseConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	DBName      string `mapstructure:"dbname"`
	SSLMode     string `mapstructure:"sslmode"`
	UseInMemory bool   `mapstructure:"use_in_memory"`
}

type ClassifierConfig struct {
	MinConfidence float64 `mapstructure:"min_confidence"`
	MaxTags       int     `mapstructure:"max_tags"`
}

type OpenAIConfig struct {
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

func parseDatabaseURL(dbURL string) (DatabaseConfig, error) {
	u, err := url.Parse(dbURL)
	if err != nil {
		return DatabaseConfig{}, err
	}

	password, _ := u.User.Password()
	port := 5432 // default PostgreSQL port
	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &port)
	}

	// Remove leading slash from path to get database name
	dbName := strings.TrimPrefix(u.Path, "/")

	return DatabaseConfig{
		Host:     u.Hostname(),
		Port:     port,
		User:     u.User.Username(),
		Password: password,
		DBName:   dbName,
		SSLMode:  "disable",
	}, nil
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	
	// Set default values
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.use_in_memory", false)
	v.SetDefault("classifier.min_confidence", 0.7)
	v.SetDefault("classifier.max_tags", 5)
	v.SetDefault("openai.model", "gpt-3.5-turbo")
	v.SetDefault("openai.max_tokens", 150)
	v.SetDefault("openai.temperature", 0.7)

	// Enable environment variable support
	v.AutomaticEnv()

	// Read the config file
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Check for DATABASE_URL environment variable
	if dbURL := v.GetString("DATABASE_URL"); dbURL != "" {
		dbConfig, err := parseDatabaseURL(dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DATABASE_URL: %v", err)
		}
		config.Database = dbConfig
	}

	// Get other environment variables
	if token := v.GetString("TELEGRAM_TOKEN"); token != "" {
		config.Telegram.Token = token
	}

	if apiKey := v.GetString("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}

	return &config, nil
} 