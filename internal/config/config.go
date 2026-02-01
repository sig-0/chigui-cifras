package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
)

const (
	DefaultListenAddress = "0.0.0.0:8080"

	DefaultFXRatesURL = "https://api.ojoporciento.com"
	DefaultFXTimeout  = 10 * time.Second
)

var (
	errMissingTelegramToken      = errors.New("missing telegram token")
	errMissingListenAddr         = errors.New("missing listen address")
	errMissingWebhookSecretToken = errors.New("missing webhook secret token")
	errMissingFXRatesBaseURL     = errors.New("missing fxrates base url")
	errFXRatesTimeoutNonPositive = errors.New("fxrates timeout must be positive")
)

// Config holds all application configuration
type Config struct {
	ListenAddress string         `toml:"listen_address"`
	Telegram      TelegramConfig `toml:"telegram"`
	FXRates       FXRatesConfig  `toml:"fxrates"`
}

// TelegramConfig holds Telegram bot settings
type TelegramConfig struct {
	Token              string `toml:"token"`
	WebhookURL         string `toml:"webhook_url"`
	WebhookSecretToken string `toml:"webhook_secret_token"`
}

// FXRatesConfig holds fxrates API client settings
type FXRatesConfig struct {
	BaseURL string        `toml:"base_url"`
	Timeout time.Duration `toml:"timeout"`
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		ListenAddress: DefaultListenAddress,
		FXRates: FXRatesConfig{
			BaseURL: DefaultFXRatesURL,
			Timeout: DefaultFXTimeout,
		},
	}
}

// ValidateConfig validates the server configuration
func ValidateConfig(config *Config) error {
	if strings.TrimSpace(config.Telegram.Token) == "" {
		return errMissingTelegramToken
	}

	if strings.TrimSpace(config.ListenAddress) == "" {
		return errMissingListenAddr
	}

	if _, _, splitErr := net.SplitHostPort(config.ListenAddress); splitErr != nil {
		return fmt.Errorf("invalid listen address: %q", config.ListenAddress)
	}

	if strings.TrimSpace(config.FXRates.BaseURL) == "" {
		return errMissingFXRatesBaseURL
	}

	parsedFXURL, err := url.Parse(config.FXRates.BaseURL)
	if err != nil || !parsedFXURL.IsAbs() {
		return fmt.Errorf("invalid fxrates base url: %q", config.FXRates.BaseURL)
	}

	if parsedFXURL.Scheme != "https" && parsedFXURL.Scheme != "http" {
		return fmt.Errorf("fxrates base url must use http or https: %q", config.FXRates.BaseURL)
	}

	if config.FXRates.Timeout <= 0 {
		return errFXRatesTimeoutNonPositive
	}

	if strings.TrimSpace(config.Telegram.WebhookURL) == "" {
		return nil
	}

	parsedWebhookURL, err := url.Parse(config.Telegram.WebhookURL)
	if err != nil || !parsedWebhookURL.IsAbs() {
		return fmt.Errorf("invalid webhook url: %q", config.Telegram.WebhookURL)
	}

	if parsedWebhookURL.Scheme != "https" {
		return fmt.Errorf("webhook url must use https: %q", config.Telegram.WebhookURL)
	}

	if strings.TrimSpace(config.Telegram.WebhookSecretToken) == "" {
		return errMissingWebhookSecretToken
	}

	return nil
}

// Read reads the configuration from the given path
func Read(path string) (*Config, error) {
	// Read the config file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse it
	cfg := DefaultConfig()

	if err := toml.Unmarshal(content, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
