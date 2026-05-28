package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BotToken          string
	DatabaseURL       string
	DefaultTimezone   string
	WorkerTickSeconds int
	LogLevel          string
	Environment       string
	ControlHTTPAddr   string
	PollTimeout       int
}

func Load() (Config, error) {
	_ = loadDotEnv(".env")
	cfg := Config{
		BotToken:          os.Getenv("TRACKMATE__BOT_TOKEN"),
		DatabaseURL:       strings.TrimSpace(os.Getenv("TRACKMATE__DATABASE_URL")),
		DefaultTimezone:   getEnvDefault("TRACKMATE__DEFAULT_TIMEZONE", "Europe/Moscow"),
		WorkerTickSeconds: getEnvIntDefault("TRACKMATE__WORKER_TICK_SECONDS", 5),
		LogLevel:          getEnvDefault("TRACKMATE__LOG_LEVEL", "INFO"),
		Environment:       getEnvDefault("TRACKMATE__ENVIRONMENT", "development"),
		ControlHTTPAddr:   os.Getenv("TRACKMATE__CONTROL_HTTP_ADDR"),
		PollTimeout:       getEnvIntDefault("TRACKMATE__POLL_TIMEOUT_SECONDS", 25),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("TRACKMATE__DATABASE_URL is required")
	}
	if _, err := time.LoadLocation(cfg.DefaultTimezone); err != nil {
		return Config{}, fmt.Errorf("invalid TRACKMATE__DEFAULT_TIMEZONE: %w", err)
	}
	return cfg, nil
}

func (c Config) RequireBotToken() error {
	if c.BotToken == "" || c.BotToken == "replace-me" {
		return fmt.Errorf("TRACKMATE__BOT_TOKEN is required")
	}
	return nil
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production") || strings.EqualFold(c.Environment, "prod")
}

func (c Config) ControlEnabled() bool {
	return c.ControlHTTPAddr != "" && !c.IsProduction()
}

func getEnvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvIntDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}
