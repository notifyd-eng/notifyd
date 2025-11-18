package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Store    StoreConfig    `mapstructure:"store"`
	Channels ChannelConfig  `mapstructure:"channels"`
	Retry    RetryConfig    `mapstructure:"retry"`
}

type ServerConfig struct {
	Listen      string        `mapstructure:"listen"`
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
	APIKey      string        `mapstructure:"api_key"`
}

type StoreConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type RetryConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	InitialWait time.Duration `mapstructure:"initial_wait"`
	MaxWait     time.Duration `mapstructure:"max_wait"`
	Multiplier  float64       `mapstructure:"multiplier"`
}

type ChannelConfig struct {
	Email   *EmailConfig   `mapstructure:"email"`
	Slack   *SlackConfig   `mapstructure:"slack"`
	Webhook *WebhookConfig `mapstructure:"webhook"`
}

type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

type SlackConfig struct {
	Token      string `mapstructure:"token"`
	DefaultCh  string `mapstructure:"default_channel"`
}

type WebhookConfig struct {
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetDefault("server.listen", ":8400")
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 120*time.Second)
	v.SetDefault("store.driver", "sqlite3")
	v.SetDefault("store.dsn", "/var/lib/notifyd/notifyd.db")
	v.SetDefault("retry.max_attempts", 5)
	v.SetDefault("retry.initial_wait", 1*time.Second)
	v.SetDefault("retry.max_wait", 5*time.Minute)
	v.SetDefault("retry.multiplier", 2.0)
	v.SetDefault("channels.webhook.timeout", 10*time.Second)
	v.SetDefault("channels.webhook.max_retries", 3)

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/notifyd")
		v.AddConfigPath("$HOME/.notifyd")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("NOTIFYD")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		fmt.Fprintln(os.Stderr, "no config file found, using defaults")
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
