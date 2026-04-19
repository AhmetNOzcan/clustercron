package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	HTTPPort    int
	NodeID      string
	LogLevel    string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		NodeID:      getEnvOrDefault("NODE_ID", hostnameOrRandom()),
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
	}

	port, err := getEnvAsInt("HTTP_PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_PORT %w", err)
	}
	cfg.HTTPPort = port

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s=%q as int: %w", key, v, err)
	}
	return n, nil
}

func hostnameOrRandom() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "node-unknown"
}
