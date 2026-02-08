package bot

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/sig-0/fxrates/provider/currencies"
	"github.com/sig-0/fxrates/storage/types"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
)

// currencyEmoji maps currency codes to emoji representations
var currencyEmoji = map[fxrates.Currency]string{
	currencies.USD:  "\U0001F4B5", // dollar
	currencies.EUR:  "\U0001F4B6", // euro
	currencies.VES:  "\U0001F1FB\U0001F1EA",
	currencies.USDT: "\U0001F4B2",
	currencies.RUB:  "\U0001F1F7\U0001F1FA",
	currencies.TRY:  "\U0001F1F9\U0001F1F7",
	currencies.CNY:  "\U0001F1E8\U0001F1F3",
}

var (
	caracasLocation = time.FixedZone("VET", -4*60*60)

	spanishMonths = [12]string{
		"ene", "feb", "mar", "abr", "may", "jun",
		"jul", "ago", "sep", "oct", "nov", "dic",
	}
)

func emojiForCurrency(currency fxrates.Currency) string {
	if e, ok := currencyEmoji[currency]; ok {
		return e
	}

	return "\U0001F4B1" // generic emoji
}

// spanishMonth returns the abbreviated Spanish month name for m (1-indexed)
func spanishMonth(m time.Month) string {
	if m < time.January || m > time.December {
		return ""
	}

	return spanishMonths[m-1]
}

// formatTime formats a UTC time into "DD month YYYY, HH:MM VET"
func formatTime(value time.Time) string {
	t := value.In(caracasLocation)

	return fmt.Sprintf("%02d %s %d, %02d:%02d VET",
		t.Day(), spanishMonth(t.Month()), t.Year(), t.Hour(), t.Minute())
}

// rateTypeLabel returns a Spanish label for the given rate type.
func rateTypeLabel(rt fxrates.RateType) string {
	switch rt {
	case types.RateTypeBUY:
		return "Compra"
	case types.RateTypeSELL:
		return "Venta"
	default:
		return "Tasa media"
	}
}

// FormatRate formats a single exchange rate as HTML.
func FormatRate(rate fxrates.ExchangeRate) string {
	emoji := emojiForCurrency(rate.Base)
	base := html.EscapeString(string(rate.Base))
	target := html.EscapeString(string(rate.Target))
	source := html.EscapeString(string(rate.Source))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s <b>%s → %s</b>\n\n", emoji, base, target))
	sb.WriteString(fmt.Sprintf("<code>%.2f</code> Bs\n\n", rate.Rate))
	sb.WriteString(fmt.Sprintf("\U0001F4CA %s · %s\n", source, rateTypeLabel(rate.RateType)))
	sb.WriteString(fmt.Sprintf("\U0001F4C5 %s", formatTime(rate.AsOf)))

	return sb.String()
}

// FormatRates formats multiple exchange rates as HTML.
func FormatRates(rates []fxrates.ExchangeRate) string {
	if len(rates) == 0 {
		return "No se encontraron tasas"
	}

	base := rates[0].Base
	emoji := emojiForCurrency(base)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s <b>Tasas de %s</b>\n\n", emoji, html.EscapeString(string(base))))

	for _, rate := range rates {
		sb.WriteString(fmt.Sprintf("• %s: <code>%.2f</code> (%s, %s)\n",
			html.EscapeString(string(rate.Target)), rate.Rate,
			html.EscapeString(string(rate.Source)), rateTypeLabel(rate.RateType)))
	}

	sb.WriteString(fmt.Sprintf("\n\U0001F4C5 %s", formatTime(rates[0].AsOf)))

	return sb.String()
}

// FormatDashboard formats the dashboard view showing popular VES rates as HTML.
func FormatDashboard(usd, eur *fxrates.ExchangeRate, usdtRates []fxrates.ExchangeRate) string {
	var sb strings.Builder
	sb.WriteString("\U0001F4CA <b>Tasas del día</b>\n\n")

	if usd != nil {
		sb.WriteString(fmt.Sprintf("\U0001F4B5 USD/VES  <code>%.2f</code>\n", usd.Rate))
	}

	if eur != nil {
		sb.WriteString(fmt.Sprintf("\U0001F4B6 EUR/VES  <code>%.2f</code>\n", eur.Rate))
	}

	if len(usdtRates) > 0 {
		sb.WriteString("\U0001F4B2 USDT/VES\n")

		for _, r := range usdtRates {
			switch r.RateType {
			case types.RateTypeBUY:
				sb.WriteString(fmt.Sprintf("  ↗ Compra: <code>%.2f</code>\n", r.Rate))
			case types.RateTypeSELL:
				sb.WriteString(fmt.Sprintf("  ↘ Venta: <code>%.2f</code>\n", r.Rate))
			default:
				sb.WriteString(fmt.Sprintf("  • <code>%.2f</code>\n", r.Rate))
			}
		}
	}

	// Show separate timestamps for BCV and USDT
	// since they update at different frequencies
	sb.WriteString("\n")

	if usd != nil {
		sb.WriteString(fmt.Sprintf("\U0001F4C5 %s · %s\n",
			html.EscapeString(string(usd.Source)), formatTime(usd.AsOf)))
	} else if eur != nil {
		sb.WriteString(fmt.Sprintf("\U0001F4C5 %s · %s\n",
			html.EscapeString(string(eur.Source)), formatTime(eur.AsOf)))
	}

	if len(usdtRates) > 0 {
		sb.WriteString(fmt.Sprintf("\U0001F4C5 %s · %s",
			html.EscapeString(string(usdtRates[0].Source)), formatTime(usdtRates[0].AsOf)))
	}

	return sb.String()
}

// FormatCurrencies formats the list of currencies as HTML.
func FormatCurrencies(currencyList []fxrates.Currency) string {
	var sb strings.Builder
	sb.WriteString("\U0001F4B1 <b>Monedas soportadas</b>\n\n")

	for _, currency := range currencyList {
		sb.WriteString(fmt.Sprintf("%s %s\n", emojiForCurrency(currency), html.EscapeString(string(currency))))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// StartMessage returns the welcome message as HTML.
func StartMessage() string {
	var sb strings.Builder
	sb.WriteString("👋 <b>¡Hola!</b>\n\n")
	sb.WriteString("Ofrezco tasas de cambio en tiempo real ")
	sb.WriteString("para VES (Bolívar venezolano).\n\n")
	sb.WriteString("<b>Comandos rápidos:</b>\n") //nolint:misspell // Spanish
	sb.WriteString("• /dolar - Tasa USD/VES\n")
	sb.WriteString("• /euro - Tasa EUR/VES\n")
	sb.WriteString("• /usdt - Tasa USDT/VES\n")
	sb.WriteString("\n<b>Más opciones:</b>\n")
	sb.WriteString("• /tasa - Tasas del día\n")
	sb.WriteString("• /tasa USD - Tasa de una moneda\n")
	sb.WriteString("• /monedas - Listar monedas disponibles\n")
	sb.WriteString("\nEscribe /ayuda para ver todos los comandos.") //nolint:misspell // Spanish

	return sb.String()
}

// HelpMessage returns the help message as HTML.
func HelpMessage() string {
	var sb strings.Builder
	sb.WriteString("\U0001F4D6 <b>Comandos de ChiguiCifras</b>\n\n") //nolint:misspell // Spanish
	sb.WriteString("<b>Consultas de tasas:</b>\n")
	sb.WriteString("• /tasa - Tasas del día\n")
	sb.WriteString("• /tasa &lt;base&gt; [destino] - Obtener una tasa de cambio\n")
	sb.WriteString("• /monedas - Listar monedas disponibles\n")

	sb.WriteString("\n<b>Atajos VES:</b>\n")
	sb.WriteString("• /dolar - USD/VES\n")
	sb.WriteString("• /euro - EUR/VES\n")
	sb.WriteString("• /usdt - USDT/VES\n")

	sb.WriteString("\n<b>Ejemplos:</b>\n")
	sb.WriteString("• /tasa USD VES")

	return sb.String()
}

// ErrorMessage formats an error message as HTML
func ErrorMessage(err error) string {
	return fmt.Sprintf("❌ Error: %s", html.EscapeString(err.Error()))
}

// InvalidUsageMessage returns an invalid usage message as HTML
func InvalidUsageMessage(usage string) string {
	return fmt.Sprintf("❌ Uso inválido.\n\nUso: %s", usage)
}
