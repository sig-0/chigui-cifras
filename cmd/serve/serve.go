package serve

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/sync/errgroup"

	"github.com/sig-0/chigui-cifras/cmd/env"
	"github.com/sig-0/chigui-cifras/internal/bot"
	"github.com/sig-0/chigui-cifras/internal/config"
	"github.com/sig-0/chigui-cifras/internal/fxrates"
)

// serveCfg wraps the serve configuration
type serveCfg struct {
	config *config.Config

	configPath        string
	webhookListenAddr string
}

// NewServeCmd creates the serve subcommand
func NewServeCmd() *ffcli.Command {
	cfg := &serveCfg{
		config: config.DefaultConfig(),
	}

	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfg.registerFlags(fs)

	return &ffcli.Command{
		Name:       "serve",
		ShortUsage: "serve [flags]",
		LongHelp:   "Starts the ChiguiCifras Telegram bot",
		FlagSet:    fs,
		Exec:       cfg.exec,
	}
}

func (c *serveCfg) registerFlags(fs *flag.FlagSet) {
	fs.StringVar(
		&c.configPath,
		"config",
		"",
		"the path to the server TOML configuration, if any",
	)

	fs.StringVar(
		&c.webhookListenAddr,
		"listen",
		"",
		"webhook listen address",
	)
}

func (c *serveCfg) exec(ctx context.Context, _ []string) error {
	// Read the server configuration, if any
	if c.configPath != "" {
		serverCfg, err := config.Read(c.configPath)
		if err != nil {
			return fmt.Errorf("unable to read server config, %w", err)
		}

		c.config = serverCfg
	}

	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)

	// Load .env
	if err := godotenv.Load(); err != nil {
		logger.Warn("unable to load .env file")
	}

	if err := applyEnv(c.config); err != nil {
		return err
	}

	// Set the webhook listen address (for Telegram)
	if c.webhookListenAddr != "" {
		c.config.Telegram.WebhookListenAddr = c.webhookListenAddr
	}

	if err := config.ValidateConfig(c.config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Initialize fxrates client
	fxClient := fxrates.NewClient(c.config.FXRates.BaseURL, c.config.FXRates.Timeout)

	// Initialize the Telegram bot
	tgBot, err := bot.New(
		c.config.Telegram.Token,
		fxClient,
		logger,
		bot.Settings{
			WebhookSecretToken: c.config.Telegram.WebhookSecretToken,
		},
	)
	if err != nil {
		return fmt.Errorf("unable to create bot: %w", err)
	}

	// Setup run ctx
	runCtx, cancelFn := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancelFn()

	group, gCtx := errgroup.WithContext(runCtx)

	_, setErr := tgBot.SetWebhook(
		gCtx,
		c.config.Telegram.WebhookURL,
		c.config.Telegram.WebhookSecretToken,
	)
	if setErr != nil {
		return fmt.Errorf("unable to set webhook: %w", setErr)
	}

	parsedWebhookURL, err := url.Parse(c.config.Telegram.WebhookURL)
	if err != nil || !parsedWebhookURL.IsAbs() {
		return fmt.Errorf("invalid webhook url: %q", c.config.Telegram.WebhookURL)
	}

	webhookPath := parsedWebhookURL.Path
	if webhookPath == "" {
		webhookPath = "/"
	}

	// Set up the mux handlers
	mux := http.NewServeMux()
	mux.Handle(webhookPath, tgBot.WebhookHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:              c.config.Telegram.WebhookListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	group.Go(func() error {
		defer logger.Info("server shut down")

		logger.Info(
			"starting webhook listener",
			"listen_addr", c.config.Telegram.WebhookListenAddr,
			"webhook_url", c.config.Telegram.WebhookURL,
			"webhook_path", webhookPath,
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	group.Go(func() error {
		<-gCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	})

	group.Go(func() error {
		logger.Info("starting telegram bot in webhook mode")
		tgBot.Start(gCtx)

		return nil
	})

	return group.Wait()
}

func applyEnv(cfg *config.Config) error {
	if v, ok := os.LookupEnv(env.Prefix + "_" + env.TelegramTokenSuffix); ok {
		cfg.Telegram.Token = v
	}

	if v, ok := os.LookupEnv(env.Prefix + "_" + env.WebhookURLSuffix); ok {
		cfg.Telegram.WebhookURL = v
	}

	if v, ok := os.LookupEnv(env.Prefix + "_" + env.WebhookListenAddrSuffix); ok {
		cfg.Telegram.WebhookListenAddr = v
	}

	if v, ok := os.LookupEnv(env.Prefix + "_" + env.WebhookSecretTokenSuffix); ok {
		cfg.Telegram.WebhookSecretToken = v
	}

	if v, ok := os.LookupEnv(env.Prefix + "_" + env.FXRatesURLSuffix); ok {
		cfg.FXRates.BaseURL = v
	}

	if v, ok := os.LookupEnv(env.Prefix + "_" + env.FXRatesTimeoutSuffix); ok {
		timeout, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid %s_%s: %w", env.Prefix, env.FXRatesTimeoutSuffix, err)
		}

		cfg.FXRates.Timeout = timeout
	}

	return nil
}
