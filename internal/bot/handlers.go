package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sig-0/fxrates/provider/currencies"
	"github.com/sig-0/fxrates/storage/types"

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

	rate := selectPreferredRate(rates.Results)
	if rate == nil {
		if lang == LanguageEN {
			h.reply(ctx, b, update, "No rates found for "+base+"/"+target)
		} else {
			h.reply(ctx, b, update, "No se encontraron tasas para "+base+"/"+target)
		}

		return
	}

	h.reply(ctx, b, update, FormatRate(*rate, lang))
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

// InlineQuery handles inline mode requests
func (h *FxHandler) InlineQuery(ctx context.Context, b *bot.Bot, update *models.Update) {
	inlineQuery := update.InlineQuery
	if inlineQuery == nil {
		return
	}

	lang := h.languageForInline(inlineQuery)

	base, target, ok := parseInlineQuery(inlineQuery.Query)
	if !ok {
		h.answerInlineHelp(ctx, b, inlineQuery, lang)

		return
	}

	rates, err := h.fxClient.Rate(ctx, base, target)
	if err != nil {
		h.answerInlineError(ctx, b, inlineQuery, lang)

		return
	}

	rate := selectPreferredRate(rates.Results)
	if rate == nil {
		h.answerInlineEmpty(ctx, b, inlineQuery, lang, base, target)

		return
	}

	title := fmt.Sprintf("%s/%s", rate.Base, rate.Target)
	description := fmt.Sprintf("%.4f (%s, %s)", rate.Rate, rate.Source, rate.RateType)
	message := FormatRate(*rate, lang)

	h.answerInlineResults(ctx, b, inlineQuery, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:          inlineResultID(title),
			Title:       title,
			Description: description,
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: message,
			},
		},
	})
}

func (h *FxHandler) rateShortcut(ctx context.Context, b *bot.Bot, update *models.Update, base string) {
	target := currencies.VES.String()

	rates, err := h.fxClient.Rate(ctx, base, target)
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err, LanguageES))

		return
	}

	rate := selectPreferredRate(rates.Results)
	if rate == nil {
		h.reply(ctx, b, update, "No se encontraron tasas para "+base+"/"+target)

		return
	}

	h.reply(ctx, b, update, FormatRate(*rate, LanguageES))
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

func (h *FxHandler) languageForInline(query *models.InlineQuery) Language {
	if query == nil || query.From == nil {
		return LanguageES
	}

	if strings.HasPrefix(strings.ToLower(query.From.LanguageCode), "en") {
		return LanguageEN
	}

	return LanguageES
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

func (h *FxHandler) answerInlineHelp(
	ctx context.Context,
	b *bot.Bot,
	query *models.InlineQuery,
	lang Language,
) {
	title := "Ayuda"
	description := "Escribe: USD VES (destino VES por defecto)"
	message := "Usa: USD VES o solo USD"

	if lang == LanguageEN {
		title = "Help"
		description = "Type: USD VES (default target VES)"
		message = "Use: USD VES or just USD"
	}

	h.answerInlineResults(ctx, b, query, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:          "help",
			Title:       title,
			Description: description,
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: message,
			},
		},
	})
}

func (h *FxHandler) answerInlineEmpty(
	ctx context.Context,
	b *bot.Bot,
	query *models.InlineQuery,
	lang Language,
	base string,
	target string,
) {
	title := "Sin resultados"
	message := "No se encontraron tasas para " + base + "/" + target

	if lang == LanguageEN {
		title = "No results"
		message = "No rates found for " + base + "/" + target
	}

	h.answerInlineResults(ctx, b, query, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:    "empty",
			Title: title,
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: message,
			},
		},
	})
}

func (h *FxHandler) answerInlineError(
	ctx context.Context,
	b *bot.Bot,
	query *models.InlineQuery,
	lang Language,
) {
	title := "Error"
	message := "No se pudo obtener la tasa"

	if lang == LanguageEN {
		message = "Unable to fetch the rate"
	}

	h.answerInlineResults(ctx, b, query, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:    "error",
			Title: title,
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: message,
			},
		},
	})
}

func (h *FxHandler) answerInlineResults(
	ctx context.Context,
	b *bot.Bot,
	query *models.InlineQuery,
	results []models.InlineQueryResult,
) {
	if query == nil {
		return
	}

	_, err := b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		Results:       results,
		CacheTime:     5,
		IsPersonal:    true,
	})
	if err != nil {
		return
	}
}

func inlineResultID(title string) string {
	return strings.ReplaceAll(strings.ToLower(title), "/", "-")
}

// selectPreferredRate selects the best rate from the results based on the currency pair.
// For fiat currencies (USD, EUR, etc.), it prefers MID rate from BCV.
// For crypto (USDT), it prefers whatever is available (typically P2P)
func selectPreferredRate(rates []fxrates.ExchangeRate) *fxrates.ExchangeRate {
	if len(rates) == 0 {
		return nil
	}

	// For single result, just return it
	if len(rates) == 1 {
		return &rates[0]
	}

	base := rates[0].Base

	// For fiat currencies, prefer BCV MID rate
	isFiat := base == currencies.USD || base == currencies.EUR ||
		base == currencies.RUB || base == currencies.TRY || base == currencies.CNY

	if isFiat {
		// First try: BCV + MID
		for i := range rates {
			if rates[i].Source == types.SourceBCV && rates[i].RateType == types.RateTypeMID {
				return &rates[i]
			}
		}

		// Second try: any MID rate
		for i := range rates {
			if rates[i].RateType == types.RateTypeMID {
				return &rates[i]
			}
		}
	}

	// Default: return first result
	return &rates[0]
}

func parseInlineQuery(query string) (string, string, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(query))
	if normalized == "" {
		return "", "", false
	}

	normalized = strings.NewReplacer("/", " ", "-", " ").Replace(normalized)
	parts := strings.Fields(normalized)

	if len(parts) == 0 {
		return "", "", false
	}

	base := parts[0]
	target := currencies.VES.String()

	if len(parts) > 1 {
		target = parts[1]
	}

	return base, target, true
}
