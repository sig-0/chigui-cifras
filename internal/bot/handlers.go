package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/sig-0/fxrates/provider/currencies"
	"github.com/sig-0/fxrates/provider/ves"
	"github.com/sig-0/fxrates/storage/types"
	"golang.org/x/sync/errgroup"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
)

// FxHandler holds command handler and their dependencies
type FxHandler struct {
	fxClient *fxrates.Client
	logger   *slog.Logger
}

// NewHandlers creates a new FxHandler instance
func NewHandlers(fxClient *fxrates.Client, logger *slog.Logger) *FxHandler {
	return &FxHandler{
		fxClient: fxClient,
		logger:   logger,
	}
}

// Start handles the /start command
func (h *FxHandler) Start(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.reply(ctx, b, update, StartMessage())
}

// Help handles the /ayuda command
func (h *FxHandler) Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.reply(ctx, b, update, HelpMessage())
}

// Rate handles the /tasa command.
// With no args, it shows the dashboard; with one arg, shows pair vs VES; two args, shows explicit pair
func (h *FxHandler) Rate(ctx context.Context, b *bot.Bot, update *models.Update) {
	args := h.parseArgs(update.Message.Text)

	if len(args) < 1 {
		h.dashboard(ctx, b, update)

		return
	}

	base := strings.ToUpper(args[0])
	target := currencies.VES.String()

	if len(args) >= 2 {
		target = strings.ToUpper(args[1])
	}

	source := sourceForCurrency(fxrates.Currency(base))

	rates, err := h.fxClient.Rate(ctx, base, target, source.String())
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err))

		return
	}

	rate := selectPreferredRate(rates.Results)
	if rate == nil {
		h.reply(ctx, b, update, "No se encontraron tasas para "+base+"/"+target)

		return
	}

	h.reply(ctx, b, update, FormatRate(*rate))
}

// dashboard fetches USD, EUR, and USDT rates in parallel and renders the dashboard
func (h *FxHandler) dashboard(ctx context.Context, b *bot.Bot, update *models.Update) {
	var (
		usdRate  *fxrates.ExchangeRate
		eurRate  *fxrates.ExchangeRate
		usdtPage *fxrates.PageExchangeRate
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		rates, err := h.fxClient.Rate(gctx, currencies.USD.String(), currencies.VES.String(), ves.BCVSource.String())
		if err != nil {
			return err
		}

		usdRate = selectPreferredRate(rates.Results)

		return nil
	})

	g.Go(func() error {
		rates, err := h.fxClient.Rate(gctx, currencies.EUR.String(), currencies.VES.String(), ves.BCVSource.String())
		if err != nil {
			return err
		}

		eurRate = selectPreferredRate(rates.Results)

		return nil
	})

	g.Go(func() error {
		rates, err := h.fxClient.Rate(gctx, currencies.USDT.String(), currencies.VES.String(), "")
		if err != nil {
			return err
		}

		usdtPage = rates

		return nil
	})

	if err := g.Wait(); err != nil {
		h.reply(ctx, b, update, ErrorMessage(err))

		return
	}

	var usdtRates []fxrates.ExchangeRate
	if usdtPage != nil {
		usdtRates = usdtPage.Results
	}

	h.reply(ctx, b, update, FormatDashboard(usdRate, eurRate, usdtRates))
}

// Currencies handles the /monedas command
func (h *FxHandler) Currencies(ctx context.Context, b *bot.Bot, update *models.Update) {
	availableCurrencies, err := h.fxClient.Currencies(ctx)
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err))

		return
	}

	h.reply(ctx, b, update, FormatCurrencies(availableCurrencies.Results))
}

// Dolar handles the /dolar shortcut
func (h *FxHandler) Dolar(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, currencies.USD.String())
}

// Euro handles the /euro shortcut
func (h *FxHandler) Euro(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.rateShortcut(ctx, b, update, currencies.EUR.String())
}

// USDT handles the /usdt shortcut, showing both BUY and SELL rates
func (h *FxHandler) USDT(ctx context.Context, b *bot.Bot, update *models.Update) {
	target := currencies.VES.String()

	rates, err := h.fxClient.Rate(ctx, currencies.USDT.String(), target, "")
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err))

		return
	}

	if len(rates.Results) == 0 {
		h.reply(ctx, b, update, "No se encontraron tasas para USDT/"+target)

		return
	}

	h.reply(ctx, b, update, FormatRates(rates.Results))
}

// InlineQuery handles inline mode requests
func (h *FxHandler) InlineQuery(ctx context.Context, b *bot.Bot, update *models.Update) {
	inlineQuery := update.InlineQuery
	if inlineQuery == nil {
		return
	}

	base, target, ok := parseInlineQuery(inlineQuery.Query)
	if !ok {
		h.answerInlineHelp(ctx, b, inlineQuery)

		return
	}

	source := sourceForCurrency(fxrates.Currency(base))

	rates, err := h.fxClient.Rate(ctx, base, target, source.String())
	if err != nil {
		h.answerInlineError(ctx, b, inlineQuery)

		return
	}

	rate := selectPreferredRate(rates.Results)
	if rate == nil {
		h.answerInlineEmpty(ctx, b, inlineQuery, base, target)

		return
	}

	title := fmt.Sprintf("%s/%s", rate.Base, rate.Target)
	description := fmt.Sprintf("%.4f (%s, %s)", rate.Rate, rate.Source, rate.RateType)
	message := FormatRate(*rate)

	h.answerInlineResults(ctx, b, inlineQuery, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:          inlineResultID(title),
			Title:       title,
			Description: description,
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: message,
				ParseMode:   models.ParseModeHTML,
			},
		},
	})
}

func (h *FxHandler) rateShortcut(ctx context.Context, b *bot.Bot, update *models.Update, base string) {
	target := currencies.VES.String()
	source := sourceForCurrency(fxrates.Currency(base))

	rates, err := h.fxClient.Rate(ctx, base, target, source.String())
	if err != nil {
		h.reply(ctx, b, update, ErrorMessage(err))

		return
	}

	rate := selectPreferredRate(rates.Results)
	if rate == nil {
		h.reply(ctx, b, update, "No se encontraron tasas para "+base+"/"+target)

		return
	}

	h.reply(ctx, b, update, FormatRate(*rate))
}

func (h *FxHandler) parseArgs(text string) []string {
	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return nil
	}

	return parts[1:]
}

func (h *FxHandler) reply(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	h.logger.Debug("sending reply",
		"chat_id", update.Message.Chat.ID,
		"text_length", len(text),
	)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		h.logger.Error("failed to send message",
			"chat_id", update.Message.Chat.ID,
			"error", err,
		)
	}
}

func (h *FxHandler) answerInlineHelp(
	ctx context.Context,
	b *bot.Bot,
	query *models.InlineQuery,
) {
	h.answerInlineResults(ctx, b, query, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:          "help",
			Title:       "Ayuda",
			Description: "Escribe: USD VES (destino VES por defecto)",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "Usa: USD VES o solo USD",
			},
		},
	})
}

func (h *FxHandler) answerInlineEmpty(
	ctx context.Context,
	b *bot.Bot,
	query *models.InlineQuery,
	base string,
	target string,
) {
	h.answerInlineResults(ctx, b, query, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:    "empty",
			Title: "Sin resultados",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "No se encontraron tasas para " + base + "/" + target,
			},
		},
	})
}

func (h *FxHandler) answerInlineError(
	ctx context.Context,
	b *bot.Bot,
	query *models.InlineQuery,
) {
	h.answerInlineResults(ctx, b, query, []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:    "error",
			Title: "Error",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "No se pudo obtener la tasa",
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

// sourceForCurrency returns the preferred source for a given base currency.
// For fiat currencies, we use BCV
func sourceForCurrency(base fxrates.Currency) fxrates.Source {
	switch base {
	case currencies.USD, currencies.EUR, currencies.RUB, currencies.TRY, currencies.CNY:
		return ves.BCVSource
	default:
		return ""
	}
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
