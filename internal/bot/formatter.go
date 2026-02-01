package bot

import (
	"fmt"
	"strings"
	"time"

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

var caracasLocation = time.FixedZone("VET", -4*60*60)

func getEmoji(currency fxrates.Currency) string {
	if e, ok := currencyEmoji[currency]; ok {
		return e
	}

	return "\U0001F4B1" // generic currency
}

// formatTime formats the time to display VET (Venezuela time)
func formatTime(value time.Time) string {
	return value.In(caracasLocation).Format("2006-01-02 15:04 MST")
}

// FormatRate formats a single exchange rate for display
func FormatRate(rate fxrates.ExchangeRate, lang Language) string {
	emoji := getEmoji(rate.Base)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s %s → %s\n\n", emoji, rate.Base, rate.Target))

	if lang == LanguageEN {
		sb.WriteString(fmt.Sprintf("Rate: %.2f\n", rate.Rate))
		sb.WriteString(fmt.Sprintf("Source: %s\n", rate.Source))
		sb.WriteString(fmt.Sprintf("Type: %s\n\n", rate.RateType))
		sb.WriteString(fmt.Sprintf("📅 Effective: %s", formatTime(rate.AsOf)))
	} else {
		sb.WriteString(fmt.Sprintf("Tasa: %.2f\n", rate.Rate))
		sb.WriteString(fmt.Sprintf("Fuente: %s\n", rate.Source))
		sb.WriteString(fmt.Sprintf("Tipo: %s\n\n", rate.RateType))
		sb.WriteString(fmt.Sprintf("📅 Efectivo: %s", formatTime(rate.AsOf)))
	}

	return sb.String()
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
		sb.WriteString(fmt.Sprintf("• %s: %.2f (%s, %s)\n", rate.Target, rate.Rate, rate.Source, rate.RateType))
	}

	if lang == LanguageEN {
		sb.WriteString(fmt.Sprintf("\n📅 Effective: %s", formatTime(rates[0].AsOf)))
	} else {
		sb.WriteString(fmt.Sprintf("\n📅 Efectivo: %s", formatTime(rates[0].AsOf)))
	}

	return sb.String()
}

// FormatCurrencies formats the list of currencies for display
func FormatCurrencies(currencyList []fxrates.Currency, lang Language) string {
	var sb strings.Builder
	if lang == LanguageEN {
		sb.WriteString("💱 Supported currencies\n\n")
	} else {
		sb.WriteString("💱 Monedas soportadas\n\n")
	}

	for _, currency := range currencyList {
		sb.WriteString(fmt.Sprintf("%s %s\n", getEmoji(currency), currency))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// StartMessage returns the welcome message
func StartMessage(lang Language) string {
	if lang == LanguageEN {
		var sb strings.Builder
		sb.WriteString("👋 Hello!\n\n")
		sb.WriteString("I provide real-time exchange rates for VES (Venezuelan Bolivar).\n\n")
		sb.WriteString("Quick commands:\n")
		sb.WriteString("• /dolar - USD/VES rate\n")
		sb.WriteString("• /euro - EUR/VES rate\n")
		sb.WriteString("• /usdt - USDT/VES rate\n")
		sb.WriteString("\nMore options:\n")
		sb.WriteString("• /rate <base> [target] - Get a specific rate\n")
		sb.WriteString("• /rates <base> - All rates for a currency\n")
		sb.WriteString("• /currencies - List available currencies\n")
		sb.WriteString("\nType /help to see all commands.")

		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString("👋 ¡Hola!\n\n")
	sb.WriteString("Ofrezco tasas de cambio en tiempo real para VES (Bolívar venezolano).\n\n")
	sb.WriteString("Comandos rápidos:\n") //nolint:misspell // Spanish copy
	sb.WriteString("• /dolar - Tasa USD/VES\n")
	sb.WriteString("• /euro - Tasa EUR/VES\n")
	sb.WriteString("• /usdt - Tasa USDT/VES\n")
	sb.WriteString("\nMás opciones:\n")
	sb.WriteString("• /tasa <base> [destino] - Obtener una tasa específica\n")
	sb.WriteString("• /tasas <base> - Todas las tasas de una moneda\n")
	sb.WriteString("• /monedas - Listar monedas disponibles\n")
	sb.WriteString("\nEscribe /ayuda para ver todos los comandos.") //nolint:misspell // Spanish copy

	return sb.String()
}

// HelpMessage returns the help message
func HelpMessage(lang Language) string {
	if lang == LanguageEN {
		var sb strings.Builder

		sb.WriteString("📖 ChiguiCifras Commands\n\n")
		sb.WriteString("Rate queries:\n")
		sb.WriteString("• /rate <base> [target] - Get an exchange rate\n")
		sb.WriteString("• /rates <base> - List all rates for a currency\n")
		sb.WriteString("• /currencies - List available currencies\n")

		sb.WriteString("\nVES shortcuts:\n")
		sb.WriteString("• /dolar - USD/VES\n")
		sb.WriteString("• /euro - EUR/VES\n")
		sb.WriteString("• /usdt - USDT/VES\n")
		sb.WriteString("• /rublo - RUB/VES\n")
		sb.WriteString("• /lira - TRY/VES\n")
		sb.WriteString("• /yuan - CNY/VES\n")

		sb.WriteString("\nExamples:\n")
		sb.WriteString("• /rate USD VES")

		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString("📖 Comandos de ChiguiCifras\n\n") //nolint:misspell // Spanish copy
	sb.WriteString("Consultas de tasas:\n")
	sb.WriteString("• /tasa <base> [destino] - Obtener una tasa de cambio\n")
	sb.WriteString("• /tasas <base> - Listar todas las tasas de una moneda\n")
	sb.WriteString("• /monedas - Listar monedas disponibles\n")

	sb.WriteString("\nAtajos VES:\n")
	sb.WriteString("• /dolar - USD/VES\n")
	sb.WriteString("• /euro - EUR/VES\n")
	sb.WriteString("• /usdt - USDT/VES\n")
	sb.WriteString("• /rublo - RUB/VES\n")
	sb.WriteString("• /lira - TRY/VES\n")
	sb.WriteString("• /yuan - CNY/VES\n")

	sb.WriteString("\nEjemplos:\n")
	sb.WriteString("• /tasa USD VES")

	return sb.String()
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
