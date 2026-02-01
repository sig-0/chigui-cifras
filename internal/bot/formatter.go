package bot

import (
	"fmt"
	"strings"

	"github.com/sig-0/fxrates/provider/currencies"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
)

// Language indicates the output language for user-facing messages
type Language string

const (
	LanguageES Language = "es"
	LanguageEN Language = "en"
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

func getEmoji(currency fxrates.Currency) string {
	if e, ok := currencyEmoji[currency]; ok {
		return e
	}

	return "\U0001F4B1" // generic currency
}

// FormatRate formats a single exchange rate for display
func FormatRate(rate fxrates.ExchangeRate, lang Language) string {
	emoji := getEmoji(rate.Base)
	if lang == LanguageEN {
		return fmt.Sprintf(`%s %s → %s

Rate: %.4f
Source: %s
Type: %s

📅 As of: %s
🔄 Fetched: %s`,
			emoji,
			rate.Base,
			rate.Target,
			rate.Rate,
			rate.Source,
			rate.RateType,
			rate.AsOf.Format("2006-01-02 15:04 MST"),
			rate.FetchedAt.Format("2006-01-02 15:04 MST"),
		)
	}

	return fmt.Sprintf(`%s %s → %s

Tasa: %.4f
Fuente: %s
Tipo: %s

📅 Fecha: %s
🔄 Actualizado: %s`,
		emoji,
		rate.Base,
		rate.Target,
		rate.Rate,
		rate.Source,
		rate.RateType,
		rate.AsOf.Format("2006-01-02 15:04 MST"),
		rate.FetchedAt.Format("2006-01-02 15:04 MST"),
	)
}

// FormatRates formats multiple exchange rates for display
func FormatRates(rates []fxrates.ExchangeRate, lang Language) string {
	if len(rates) == 0 {
		if lang == LanguageEN {
			return "No rates found"
		}

		return "No se encontraron tasas"
	}

	base := rates[0].Base
	emoji := getEmoji(base)

	var sb strings.Builder

	if lang == LanguageEN {
		sb.WriteString(fmt.Sprintf("%s Rates for %s\n\n", emoji, base))
	} else {
		sb.WriteString(fmt.Sprintf("%s Tasas de %s\n\n", emoji, base))
	}

	for _, rate := range rates {
		sb.WriteString(fmt.Sprintf("• %s: %.4f (%s, %s)\n",
			rate.Target,
			rate.Rate,
			rate.Source,
			rate.RateType,
		))
	}

	if lang == LanguageEN {
		sb.WriteString(fmt.Sprintf("\n📅 As of: %s", rates[0].AsOf.Format("2006-01-02 15:04 MST")))
	} else {
		sb.WriteString(fmt.Sprintf("\n📅 Fecha: %s", rates[0].AsOf.Format("2006-01-02 15:04 MST")))
	}

	return sb.String()
}

// FormatCurrencies formats the list of currencies for display
func FormatCurrencies(currencies []fxrates.Currency, lang Language) string {
	var sb strings.Builder
	if lang == LanguageEN {
		sb.WriteString("💱 Supported currencies\n\n")
	} else {
		sb.WriteString("💱 Monedas soportadas\n\n")
	}

	for _, currency := range currencies {
		emoji := getEmoji(currency)
		sb.WriteString(fmt.Sprintf("%s %s\n", emoji, currency))
	}

	return sb.String()
}

// StartMessage returns the welcome message
func StartMessage(lang Language) string {
	if lang == LanguageEN {
		return `👋 Hello!

I provide real-time exchange rates for VES (Venezuelan Bolivar).

Quick commands:
• /dolar - USD/VES rate
• /euro - EUR/VES rate
• /usdt - USDT/VES rate

More options:
• /rate <base> [target] - Get a specific rate
• /rates <base> - All rates for a currency
• /currencies - List available currencies

	Type /help to see all commands.`
	}

	//nolint:misspell // Spanish copy
	return `👋 ¡Hola!

Ofrezco tasas de cambio en tiempo real para VES (Bolívar venezolano).

Comandos rápidos:
• /dolar - Tasa USD/VES
• /euro - Tasa EUR/VES
• /usdt - Tasa USDT/VES

Más opciones:
• /tasa <base> [destino] - Obtener una tasa específica
• /tasas <base> - Todas las tasas de una moneda
• /monedas - Listar monedas disponibles

Escribe /ayuda para ver todos los comandos.`
}

// HelpMessage returns the help message
func HelpMessage(lang Language) string {
	if lang == LanguageEN {
		return `📖 ChiguiCifras Commands

Rate queries:
• /rate <base> [target] - Get an exchange rate
• /rates <base> - List all rates for a currency
• /currencies - List available currencies

VES shortcuts:
• /dolar - USD/VES
• /euro - EUR/VES
• /usdt - USDT/VES
• /rublo - RUB/VES
• /lira - TRY/VES
• /yuan - CNY/VES

	Examples:
• /rate USD VES`
	}

	//nolint:misspell // Spanish copy
	return `📖 Comandos de ChiguiCifras

Consultas de tasas:
• /tasa <base> [destino] - Obtener una tasa de cambio
• /tasas <base> - Listar todas las tasas de una moneda
• /monedas - Listar monedas disponibles

Atajos VES:
• /dolar - USD/VES
• /euro - EUR/VES
• /usdt - USDT/VES
• /rublo - RUB/VES
• /lira - TRY/VES
• /yuan - CNY/VES

Ejemplos:
• /tasa USD VES`
}

// ErrorMessage formats an error message
func ErrorMessage(err error, lang Language) string {
	if lang == LanguageEN {
		return fmt.Sprintf("❌ Error: %v", err)
	}

	return fmt.Sprintf("❌ Error: %v", err)
}

// InvalidUsageMessage returns an invalid usage message
func InvalidUsageMessage(usage string, lang Language) string {
	if lang == LanguageEN {
		return fmt.Sprintf("❌ Invalid usage.\n\nUsage: %s", usage)
	}

	return fmt.Sprintf("❌ Uso inválido.\n\nUso: %s", usage)
}
