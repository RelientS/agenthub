package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the AgentHub server.
type Config struct {
	Port string `mapstructure:"port"`
	Env  string `mapstructure:"env"`

	DB           DBConfig           `mapstructure:"db"`
	Redis        RedisConfig        `mapstructure:"redis"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	WS           WSConfig           `mapstructure:"ws"`
	Orchestrator OrchestratorConfig `mapstructure:"orchestrator"`
	Sync         SyncConfig         `mapstructure:"sync"`
}

// DBConfig holds database connection settings.
type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URL string `mapstructure:"url"`
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret string        `mapstructure:"secret"`
	Expire time.Duration `mapstructure:"expire"`
}

// WSConfig holds WebSocket settings.
type WSConfig struct {
	PingInterval time.Duration `mapstructure:"ping_interval"`
	PongTimeout  time.Duration `mapstructure:"pong_timeout"`
}

// OrchestratorConfig holds task orchestrator settings.
type OrchestratorConfig struct {
	CheckInterval  time.Duration `mapstructure:"check_interval"`
	StaleTaskHours int           `mapstructure:"stale_task_hours"`
}

// SyncConfig holds synchronization settings.
type SyncConfig struct {
	LogRetentionDays int `mapstructure:"log_retention_days"`
}

// Load reads configuration from environment variables and returns a Config.
// Environment variables use the AGENTHUB_ prefix (e.g., AGENTHUB_PORT, AGENTHUB_DB_HOST).
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("port", "8080")
	v.SetDefault("env", "development")

	// Database defaults
	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.name", "agenthub")
	v.SetDefault("db.user", "agenthub")
	v.SetDefault("db.password", "")

	// Redis defaults
	v.SetDefault("redis.url", "redis://localhost:6379/0")

	// JWT defaults
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expire", "720h")

	// WebSocket defaults
	v.SetDefault("ws.ping_interval", "30s")
	v.SetDefault("ws.pong_timeout", "10s")

	// Orchestrator defaults
	v.SetDefault("orchestrator.check_interval", "5m")
	v.SetDefault("orchestrator.stale_task_hours", 24)

	// Sync defaults
	v.SetDefault("sync.log_retention_days", 30)

	// Read from environment variables with AGENTHUB_ prefix
	v.SetEnvPrefix("AGENTHUB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsDevelopment returns true if the server is running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction returns true if the server is running in production mode.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
