package bot

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sig-0/fxrates/provider/currencies"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
)

// FxHandler holds command handler and their dependencies
type FxHandler struct {
	fxClient *fxrates.Client
}

// NewHandlers creates a new FxHandler instance
func NewHandlers(fxClient *fxrates.Client) *FxHandler {
	return &FxHandler{
		fxClient: fxClient,
	}
}

// Start handles the /inicio command
func (h *FxHandler) Start(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := h.languageForCommand(update.Message.Text)

	h.reply(ctx, b, update, StartMessage(lang))
}

// Help handles the /ayuda command
func (h *FxHandler) Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := h.languageForCommand(update.Message.Text)

	h.reply(ctx, b, update, HelpMessage(lang))
}

// Rate handles the /tasa command
func (h *FxHandler) Rate(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := h.languageForCommand(update.Message.Text)

	args := h.parseArgs(update.Message.Text)

	if len(args) < 1 {
		usage := "/tasa <base> [destino]"
		if lang == LanguageEN {
			usage = "/rate <base> [target]"
		}

		h.reply(ctx, b, update, InvalidUsageMessage(usage, lang))

		return
	}

	base := strings.ToUpper(args[0])
	target := currencies.VES.String()

	if len(args) >= 2 {
		target = strings.ToUpper(args[1])
	}

	rates, err := h.fxClient.Rate(ctx, base, target)
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err, lang))

		return
	}

	if len(rates.Results) == 0 {
		if lang == LanguageEN {
			h.reply(ctx, b, update, "No rates found for "+base+"/"+target)
		} else {
			h.reply(ctx, b, update, "No se encontraron tasas para "+base+"/"+target)
		}

		return
	}

	h.reply(ctx, b, update, FormatRate(rates.Results[0], lang))
}

// Rates handles the /tasas command
func (h *FxHandler) Rates(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := h.languageForCommand(update.Message.Text)

	args := h.parseArgs(update.Message.Text)

	if len(args) < 1 {
		usage := "/tasas <base>"
		if lang == LanguageEN {
			usage = "/rates <base>"
		}

		h.reply(ctx, b, update, InvalidUsageMessage(usage, lang))

		return
	}

	base := strings.ToUpper(args[0])

	rates, err := h.fxClient.Rates(ctx, base)
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err, lang))

		return
	}

	if len(rates.Results) == 0 {
		if lang == LanguageEN {
			h.reply(ctx, b, update, "No rates found for "+base)
		} else {
			h.reply(ctx, b, update, "No se encontraron tasas para "+base)
		}

		return
	}

	h.reply(ctx, b, update, FormatRates(rates.Results, lang))
}

// Currencies handles the /monedas command
func (h *FxHandler) Currencies(ctx context.Context, b *bot.Bot, update *models.Update) {
	lang := h.languageForCommand(update.Message.Text)

	availableCurrencies, err := h.fxClient.Currencies(ctx)
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err, lang))

		return
	}

	h.reply(ctx, b, update, FormatCurrencies(availableCurrencies.Results, lang))
}

// Dolar handles the /dolar shortcut
func (h *FxHandler) Dolar(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, "USD")
}

// Euro handles the /euro shortcut
func (h *FxHandler) Euro(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, "EUR")
}

// USDT handles the /usdt shortcut
func (h *FxHandler) USDT(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, "USDT")
}

// Rublo handles the /rublo shortcut
func (h *FxHandler) Rublo(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, "RUB")
}

// Lira handles the /lira shortcut
func (h *FxHandler) Lira(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, "TRY")
}

// Yuan handles the /yuan shortcut
func (h *FxHandler) Yuan(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, "CNY")
}

func (h *FxHandler) rateShortcut(ctx context.Context, b *bot.Bot, update *models.Update, base string) {
	target := currencies.VES.String()

	rates, err := h.fxClient.Rate(ctx, base, target)
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err, LanguageES))

		return
	}

	if len(rates.Results) == 0 {
		h.reply(ctx, b, update, "No se encontraron tasas para "+base+"/"+target)

		return
	}

	h.reply(ctx, b, update, FormatRate(rates.Results[0], LanguageES))
}

func (h *FxHandler) parseArgs(text string) []string {
	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return nil
	}

	return parts[1:]
}

func (h *FxHandler) commandName(text string) string {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return ""
	}

	command := strings.ToLower(parts[0])
	if at := strings.Index(command, "@"); at != -1 {
		command = command[:at]
	}

	return command
}

func (h *FxHandler) languageForCommand(text string) Language {
	switch h.commandName(text) {
	case "/start", "/help", "/rate", "/rates", "/currencies":
		return LanguageEN
	default:
		return LanguageES
	}
}

func (h *FxHandler) reply(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
	if err != nil {
		return
	}
}
