package bot

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
)

// Bot wraps the Telegram bot with handler
type Bot struct {
	bot     *bot.Bot
	handler *FxHandler
	logger  *slog.Logger
}

// Settings contains optional Telegram bot settings
type Settings struct {
	WebhookSecretToken string
}

// New creates a new Bot instance
func New(
	token string,
	fxClient *fxrates.Client,
	logger *slog.Logger,
	settings Settings,
) (*Bot, error) {
	handlers := NewHandlers(fxClient, logger)

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.InlineQuery != nil {
				handlers.InlineQuery(ctx, b, update)
			}
		}),
	}

	if settings.WebhookSecretToken != "" {
		opts = append(opts, bot.WithWebhookSecretToken(settings.WebhookSecretToken))
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create telegram bot: %w", err)
	}

	tgBot := &Bot{
		bot:     b,
		handler: handlers,
		logger:  logger,
	}

	tgBot.registerHandlers()

	return tgBot, nil
}

func (b *Bot) registerHandlers() {
	// Core commands
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/inicio", bot.MatchTypePrefix, b.handler.Start)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, b.handler.Start)

	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/ayuda", bot.MatchTypePrefix, b.handler.Help)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypePrefix, b.handler.Help)

	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/tasa", bot.MatchTypePrefix, b.handler.Rate)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/rate", bot.MatchTypePrefix, b.handler.Rate)

	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/tasas", bot.MatchTypePrefix, b.handler.Rates)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/rates", bot.MatchTypePrefix, b.handler.Rates)

	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/monedas", bot.MatchTypePrefix, b.handler.Currencies)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/currencies", bot.MatchTypePrefix, b.handler.Currencies)

	// VES shortcuts
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/dolar", bot.MatchTypePrefix, b.handler.Dolar)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/euro", bot.MatchTypePrefix, b.handler.Euro)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/usdt", bot.MatchTypePrefix, b.handler.USDT)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/rublo", bot.MatchTypePrefix, b.handler.Rublo)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/lira", bot.MatchTypePrefix, b.handler.Lira)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/yuan", bot.MatchTypePrefix, b.handler.Yuan)
}

// StartWebhook begins webhook mode dispatching for updates
func (b *Bot) StartWebhook(ctx context.Context) {
	b.bot.StartWebhook(ctx)
}

// StartPolling begins long polling mode dispatching for updates
func (b *Bot) StartPolling(ctx context.Context) {
	b.bot.Start(ctx)
}

// WebhookHandler returns an HTTP handler for Telegram webhook updates
func (b *Bot) WebhookHandler() http.Handler {
	return b.bot.WebhookHandler()
}

// SetWebhook registers the webhook URL with Telegram
func (b *Bot) SetWebhook(ctx context.Context, url, secretToken string) (bool, error) {
	return b.bot.SetWebhook(ctx, &bot.SetWebhookParams{
		URL:         url,
		SecretToken: secretToken,
	})
}

// DeleteWebhook deletes the current webhook
func (b *Bot) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) (bool, error) {
	return b.bot.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
		DropPendingUpdates: dropPendingUpdates,
	})
}

// SendMessage sends a message to a chat
func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})

	return err
}
