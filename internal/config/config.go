// Package config centralizes runtime configuration.
//
// Values are read from environment variables (12-factor style) with safe
// defaults for local development. Load returns a typed, validated struct so
// that the rest of the codebase never reads os.Getenv directly.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env             string
	HTTPAddr        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	LogLevel  string
	LogFormat string // "json" or "text"

	RateLimitRPS   float64
	RateLimitBurst int

	RedisAddr     string
	RedisPassword string

	DatabaseDSN     string
	DBMaxOpenConns  int
	DBMaxIdleConns  int
	DBConnMaxIdle   time.Duration
	DBConnMaxLife   time.Duration

	SessionAuthKey string
	SessionEncKey  string
}

func Load() (Config, error) {
	c := Config{
		Env:             getString("APP_ENV", "development"),
		HTTPAddr:        getString("HTTP_ADDR", ":8080"),
		ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 15*time.Second),

		LogLevel:  getString("LOG_LEVEL", "info"),
		LogFormat: getString("LOG_FORMAT", "json"),

		RateLimitRPS:   getFloat("RATE_LIMIT_RPS", 50),
		RateLimitBurst: getInt("RATE_LIMIT_BURST", 100),

		RedisAddr:     getString("REDIS_ADDR", "redis:6379"),
		RedisPassword: getString("REDIS_PASSWORD", ""),

		DatabaseDSN:    getString("DATABASE_DSN", ""),
		DBMaxOpenConns: getInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: getInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxIdle:  getDuration("DB_CONN_MAX_IDLE", 5*time.Minute),
		DBConnMaxLife:  getDuration("DB_CONN_MAX_LIFE", 30*time.Minute),

		SessionAuthKey: getString("SESSION_AUTH_KEY", ""),
		SessionEncKey:  getString("SESSION_ENC_KEY", ""),
	}

	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) validate() error {
	if c.Env == "production" {
		if c.SessionAuthKey == "" {
			return fmt.Errorf("SESSION_AUTH_KEY is required in production")
		}
		if c.LogFormat != "json" {
			return fmt.Errorf("LOG_FORMAT must be json in production")
		}
	}
	if c.RateLimitRPS <= 0 {
		return fmt.Errorf("RATE_LIMIT_RPS must be > 0")
	}
	return nil
}

func getString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
