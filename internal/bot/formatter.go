package bot

import (
	"fmt"
	"strings"
	"text/tabwriter"
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

// writeTabbed writes tabbed output directly into the provided builder
func writeTabbed(sb *strings.Builder, write func(w *tabwriter.Writer)) {
	w := tabwriter.NewWriter(sb, 0, 0, 2, ' ', 0)
	write(w)
	_ = w.Flush()
}

// writeTabLines writes lines separated by newlines without a trailing newline
func writeTabLines(w *tabwriter.Writer, lines []string) {
	for i, line := range lines {
		if i > 0 {
			fmt.Fprint(w, "\n")
		}

		fmt.Fprint(w, line)
	}
}

// FormatRate formats a single exchange rate for display
func FormatRate(rate fxrates.ExchangeRate, lang Language) string {
	emoji := getEmoji(rate.Base)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s %s → %s\n\n", emoji, rate.Base, rate.Target))

	if lang == LanguageEN {
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "Rate:\t%.4f\n", rate.Rate)
			fmt.Fprintf(w, "Source:\t%s\n", rate.Source)
			fmt.Fprintf(w, "Type:\t%s", rate.RateType)
		})
	} else {
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "Tasa:\t%.4f\n", rate.Rate)
			fmt.Fprintf(w, "Fuente:\t%s\n", rate.Source)
			fmt.Fprintf(w, "Tipo:\t%s", rate.RateType)
		})
	}

	sb.WriteString("\n\n")

	if lang == LanguageEN {
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "📅 As of:\t%s\n", formatTime(rate.AsOf))
			fmt.Fprintf(w, "🔄 Fetched:\t%s", formatTime(rate.FetchedAt))
		})
	} else {
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "📅 Fecha:\t%s\n", formatTime(rate.AsOf))
			fmt.Fprintf(w, "🔄 Actualizado:\t%s", formatTime(rate.FetchedAt))
		})
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

	writeTabbed(&sb, func(w *tabwriter.Writer) {
		if lang == LanguageEN {
			fmt.Fprint(w, "Target\tRate\tSource\tType")
		} else {
			fmt.Fprint(w, "Destino\tTasa\tFuente\tTipo")
		}

		for _, rate := range rates {
			fmt.Fprintf(w, "\n%s\t%.4f\t%s\t%s", rate.Target, rate.Rate, rate.Source, rate.RateType)
		}
	})

	if lang == LanguageEN {
		sb.WriteString(fmt.Sprintf("\n📅 As of: %s", formatTime(rates[0].AsOf)))
	} else {
		sb.WriteString(fmt.Sprintf("\n📅 Fecha: %s", formatTime(rates[0].AsOf)))
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

	writeTabbed(&sb, func(w *tabwriter.Writer) {
		if lang == LanguageEN {
			fmt.Fprint(w, "Currency\tEmoji")
		} else {
			fmt.Fprint(w, "Moneda\tEmoji")
		}

		for _, currency := range currencies {
			fmt.Fprintf(w, "\n%s\t%s", currency, getEmoji(currency))
		}
	})

	return sb.String()
}

// StartMessage returns the welcome message
func StartMessage(lang Language) string {
	if lang == LanguageEN {
		var sb strings.Builder
		sb.WriteString("👋 Hello!\n\n")
		sb.WriteString("I provide real-time exchange rates for VES (Venezuelan Bolivar).\n\n")
		sb.WriteString("Quick commands:\n")
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			writeTabLines(w, []string{
				"• /dolar\tUSD/VES rate",
				"• /euro\tEUR/VES rate",
				"• /usdt\tUSDT/VES rate",
			})
		})
		sb.WriteString("\n\nMore options:\n")
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			writeTabLines(w, []string{
				"• /rate <base> [target]\tGet a specific rate",
				"• /rates <base>\tAll rates for a currency",
				"• /currencies\tList available currencies",
			})
		})
		sb.WriteString("\n\nType /help to see all commands.")

		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString("👋 ¡Hola!\n\n")
	sb.WriteString("Ofrezco tasas de cambio en tiempo real para VES (Bolívar venezolano).\n\n")
	sb.WriteString("Comandos rápidos:\n") //nolint:misspell // Spanish copy
	writeTabbed(&sb, func(w *tabwriter.Writer) {
		writeTabLines(w, []string{
			"• /dolar\tTasa USD/VES",
			"• /euro\tTasa EUR/VES",
			"• /usdt\tTasa USDT/VES",
		})
	})
	sb.WriteString("\n\nMás opciones:\n")
	writeTabbed(&sb, func(w *tabwriter.Writer) {
		writeTabLines(w, []string{
			"• /tasa <base> [destino]\tObtener una tasa específica",
			"• /tasas <base>\tTodas las tasas de una moneda",
			"• /monedas\tListar monedas disponibles",
		})
	})
	sb.WriteString("\n\nEscribe /ayuda para ver todos los comandos.") //nolint:misspell // Spanish copy

	return sb.String()
}

// HelpMessage returns the help message
func HelpMessage(lang Language) string {
	if lang == LanguageEN {
		var sb strings.Builder

		sb.WriteString("📖 ChiguiCifras Commands\n\n")
		sb.WriteString("Rate queries:\n")
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			writeTabLines(w, []string{
				"• /rate <base> [target]\tGet an exchange rate",
				"• /rates <base>\tList all rates for a currency",
				"• /currencies\tList available currencies",
			})
		})

		sb.WriteString("\n\nVES shortcuts:\n")
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			writeTabLines(w, []string{
				"• /dolar\tUSD/VES",
				"• /euro\tEUR/VES",
				"• /usdt\tUSDT/VES",
				"• /rublo\tRUB/VES",
				"• /lira\tTRY/VES",
				"• /yuan\tCNY/VES",
			})
		})

		sb.WriteString("\n\nExamples:\n")
		writeTabbed(&sb, func(w *tabwriter.Writer) {
			writeTabLines(w, []string{
				"• /rate USD VES",
			})
		})

		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString("📖 Comandos de ChiguiCifras\n\n") //nolint:misspell // Spanish copy
	sb.WriteString("Consultas de tasas:\n")
	writeTabbed(&sb, func(w *tabwriter.Writer) {
		writeTabLines(w, []string{
			"• /tasa <base> [destino]\tObtener una tasa de cambio",
			"• /tasas <base>\tListar todas las tasas de una moneda",
			"• /monedas\tListar monedas disponibles",
		})
	})

	sb.WriteString("\n\nAtajos VES:\n")
	writeTabbed(&sb, func(w *tabwriter.Writer) {
		writeTabLines(w, []string{
			"• /dolar\tUSD/VES",
			"• /euro\tEUR/VES",
			"• /usdt\tUSDT/VES",
			"• /rublo\tRUB/VES",
			"• /lira\tTRY/VES",
			"• /yuan\tCNY/VES",
		})
	})

	sb.WriteString("\n\nEjemplos:\n")
	writeTabbed(&sb, func(w *tabwriter.Writer) {
		writeTabLines(w, []string{
			"• /tasa USD VES",
		})
	})

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
