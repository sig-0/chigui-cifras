package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ValidateConfig(t *testing.T) {
	t.Parallel()

	const (
		testWebhookURL  = "https://example.com/webhook"
		testSecretToken = "secret"
	)

	validConfig := func() *Config {
		cfg := DefaultConfig()

		cfg.Telegram.Token = "token"
		cfg.FXRates.BaseURL = "https://api.example.com"
		cfg.FXRates.Timeout = time.Second

		return cfg
	}

	testTable := []struct {
		name        string
		mutate      func(*Config)
		err         error
		errContains string
	}{
		{
			name: "missing telegram token",
			mutate: func(cfg *Config) {
				cfg.Telegram.Token = ""
			},
			err: errMissingTelegramToken,
		},
		{
			name: "missing listen address",
			mutate: func(cfg *Config) {
				cfg.ListenAddress = ""
			},
			err: errMissingListenAddr,
		},
		{
			name: "invalid listen address",
			mutate: func(cfg *Config) {
				cfg.ListenAddress = "invalid"
			},
			errContains: "invalid listen address",
		},
		{
			name: "invalid webhook url",
			mutate: func(cfg *Config) {
				cfg.Telegram.WebhookURL = "not-a-url"
			},
			errContains: "invalid webhook url",
		},
		{
			name: "webhook url not https",
			mutate: func(cfg *Config) {
				cfg.Telegram.WebhookURL = "http://example.com/webhook"
			},
			errContains: "webhook url must use https",
		},
		{
			name: "missing webhook secret token",
			mutate: func(cfg *Config) {
				cfg.Telegram.WebhookURL = testWebhookURL
				cfg.Telegram.WebhookSecretToken = ""
			},
			err: errMissingWebhookSecretToken,
		},
		{
			name: "valid webhook configuration",
			mutate: func(cfg *Config) {
				cfg.Telegram.WebhookURL = testWebhookURL
				cfg.Telegram.WebhookSecretToken = testSecretToken
			},
		},
		{
			name: "missing fxrates base url",
			mutate: func(cfg *Config) {
				cfg.FXRates.BaseURL = ""
			},
			err: errMissingFXRatesBaseURL,
		},
		{
			name: "invalid fxrates base url",
			mutate: func(cfg *Config) {
				cfg.FXRates.BaseURL = "not-a-url"
			},
			errContains: "invalid fxrates base url",
		},
		{
			name: "fxrates base url invalid scheme",
			mutate: func(cfg *Config) {
				cfg.FXRates.BaseURL = "ftp://example.com"
			},
			errContains: "fxrates base url must use http or https",
		},
		{
			name: "fxrates timeout non positive",
			mutate: func(cfg *Config) {
				cfg.FXRates.Timeout = 0
			},
			err: errFXRatesTimeoutNonPositive,
		},
		{
			name: "valid configuration",
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			if testCase.mutate != nil {
				testCase.mutate(cfg)
			}

			err := ValidateConfig(cfg)

			switch {
			case testCase.err != nil:
				assert.ErrorIs(t, err, testCase.err)
			case testCase.errContains != "":
				require.Error(t, err)
				assert.ErrorContains(t, err, testCase.errContains)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Read(t *testing.T) {
	t.Parallel()

	configBody := `
[telegram]
token = "token"
webhook_url = "https://example.com/webhook"
webhook_secret_token = "secret"

[fxrates]
base_url = "http://example.com"
timeout = "12s"
`

	path := filepath.Join(t.TempDir(), "config.toml")

	require.NoError(t, os.WriteFile(path, []byte(configBody), 0o600))

	cfg, err := Read(path)

	require.NoError(t, err)

	assert.Equal(t, DefaultListenAddress, cfg.ListenAddress)
	assert.Equal(t, "token", cfg.Telegram.Token)
	assert.Equal(t, "https://example.com/webhook", cfg.Telegram.WebhookURL)
	assert.Equal(t, "secret", cfg.Telegram.WebhookSecretToken)

	assert.Equal(t, "http://example.com", cfg.FXRates.BaseURL)
	assert.Equal(t, 12*time.Second, cfg.FXRates.Timeout)
}
