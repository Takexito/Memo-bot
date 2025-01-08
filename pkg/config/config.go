package config

import (
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

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	// Map environment variables
	viper.BindEnv("telegram.token", "TELEGRAM_TOKEN")
	viper.BindEnv("database.host", "PGHOST")
	viper.BindEnv("database.port", "PGPORT")
	viper.BindEnv("database.user", "PGUSER")
	viper.BindEnv("database.password", "PGPASSWORD")
	viper.BindEnv("database.dbname", "PGDATABASE")
	viper.BindEnv("openai.api_key", "OPENAI_API_KEY")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
} 