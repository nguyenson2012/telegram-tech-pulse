package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application loaded from environment variables.
type Config struct {
	ServerPort           string
	DatabaseURL          string
	GeminiAPIKey         string
	GeminiModel          string
	GeminiEmbeddingModel string
	TelegramBotToken     string
	TelegramChatID       string
	CronSchedule         string
	RunPipelineOnStartup bool
	WorkerCount          int
	QueueBufferSize      int
	TopArticlesLimit     int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/aitechpulse?sslmode=disable")
	geminiKey := getEnv("GEMINI_API_KEY", "")
	telegramToken := getEnv("TELEGRAM_BOT_TOKEN", "")
	telegramChatID := getEnv("TELEGRAM_CHAT_ID", "")

	cfg := &Config{
		ServerPort:           getEnv("PORT", getEnv("SERVER_PORT", "8080")),
		DatabaseURL:          dbURL,
		GeminiAPIKey:         geminiKey,
		GeminiModel:          getEnv("GEMINI_MODEL", "gemini-3.6-flash"),
		GeminiEmbeddingModel: getEnv("GEMINI_EMBEDDING_MODEL", "gemini-embedding-2"),
		TelegramBotToken:     telegramToken,
		TelegramChatID:       telegramChatID,
		CronSchedule:         getEnv("CRON_SCHEDULE", "0 8 * * *"),
		RunPipelineOnStartup: getEnvAsBool("RUN_PIPELINE_ON_STARTUP", false),
		WorkerCount:          getEnvAsInt("WORKER_COUNT", 3),
		QueueBufferSize:      getEnvAsInt("QUEUE_BUFFER_SIZE", 100),
		TopArticlesLimit:     getEnvAsInt("TOP_ARTICLES_LIMIT", 5),
	}

	return cfg, nil
}

// Validate checks for critical configuration requirements.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	// Note: In development or test environments, GEMINI_API_KEY and TELEGRAM credentials
	// can be optional for building and unit testing, but warning/validation can be surfaced.
	return nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}
