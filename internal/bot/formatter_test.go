package bot

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sig-0/fxrates/storage/types"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
)

func TestFormatter_FormatRate(t *testing.T) {
	t.Parallel()

	rate := fxrates.ExchangeRate{
		Base:      types.CurrencyUSD,
		Target:    types.CurrencyVES,
		Rate:      42,
		RateType:  types.RateTypeMID,
		Source:    types.SourceBCV,
		AsOf:      time.Date(2026, time.January, 2, 15, 4, 0, 0, time.UTC),
		FetchedAt: time.Date(2026, time.January, 2, 15, 5, 0, 0, time.UTC),
	}

	t.Run("english", func(t *testing.T) {
		t.Parallel()

		message := FormatRate(rate, LanguageEN)

		assert.Contains(t, message, "USD")
		assert.Contains(t, message, "VES")
		assert.Contains(t, message, "Rate:")
		assert.Contains(t, message, "42.00")
		assert.Contains(t, message, "Source:")
		assert.Contains(t, message, "BCV")
		assert.Contains(t, message, "Type:")
		assert.Contains(t, message, "MID")
		assert.Contains(t, message, "Effective:")
		assert.Contains(t, message, "2026-01-02 11:04 VET")
		assert.NotContains(t, message, "Fetched:")
	})

	t.Run("spanish", func(t *testing.T) {
		t.Parallel()

		message := FormatRate(rate, LanguageES)

		assert.Contains(t, message, "USD")
		assert.Contains(t, message, "VES")
		assert.Contains(t, message, "Tasa:")
		assert.Contains(t, message, "42.00")
		assert.Contains(t, message, "Fuente:")
		assert.Contains(t, message, "BCV")
		assert.Contains(t, message, "Tipo:")
		assert.Contains(t, message, "MID")
		assert.Contains(t, message, "Efectivo:")
		assert.Contains(t, message, "2026-01-02 11:04 VET")
		assert.NotContains(t, message, "Actualizado:")
	})
}

func TestFormatter_FormatRates(t *testing.T) {
	t.Parallel()

	var (
		rateTime = time.Date(2026, time.January, 2, 15, 4, 0, 0, time.UTC)
		rates    = []fxrates.ExchangeRate{
			{
				Base:      types.CurrencyUSD,
				Target:    types.CurrencyVES,
				Rate:      40,
				RateType:  types.RateTypeMID,
				Source:    types.SourceBCV,
				AsOf:      rateTime,
				FetchedAt: rateTime,
			},
			{
				Base:      types.CurrencyUSD,
				Target:    types.CurrencyEUR,
				Rate:      0.9,
				RateType:  types.RateTypeMID,
				Source:    types.SourceBCV,
				AsOf:      rateTime,
				FetchedAt: rateTime,
			},
		}
	)

	t.Run("empty english", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "No rates found", FormatRates(nil, LanguageEN))
	})

	t.Run("empty spanish", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "No se encontraron tasas", FormatRates(nil, LanguageES))
	})

	t.Run("english", func(t *testing.T) {
		t.Parallel()

		message := FormatRates(rates, LanguageEN)

		assert.Contains(t, message, "Rates for USD")
		assert.Contains(t, message, "VES")
		assert.Contains(t, message, "40.00")
		assert.Contains(t, message, "EUR")
		assert.Contains(t, message, "0.90")
		assert.Contains(t, message, "BCV")
		assert.Contains(t, message, "MID")
		assert.Contains(t, message, "Effective:")
		assert.Contains(t, message, "2026-01-02 11:04 VET")
	})

	t.Run("spanish", func(t *testing.T) {
		t.Parallel()

		message := FormatRates(rates, LanguageES)

		assert.Contains(t, message, "Tasas de USD")
		assert.Contains(t, message, "VES")
		assert.Contains(t, message, "40.00")
		assert.Contains(t, message, "EUR")
		assert.Contains(t, message, "0.90")
		assert.Contains(t, message, "BCV")
		assert.Contains(t, message, "MID")
		assert.Contains(t, message, "Efectivo:")
		assert.Contains(t, message, "2026-01-02 11:04 VET")
	})
}

func TestFormatter_FormatCurrencies(t *testing.T) {
	t.Parallel()

	currencies := []fxrates.Currency{
		types.CurrencyUSD,
		types.CurrencyVES,
		types.CurrencyEUR,
	}

	t.Run("english", func(t *testing.T) {
		t.Parallel()

		message := FormatCurrencies(currencies, LanguageEN)

		assert.Contains(t, message, "Supported currencies")
		assert.Contains(t, message, "USD")
		assert.Contains(t, message, "VES")
		assert.Contains(t, message, "EUR")
	})

	t.Run("spanish", func(t *testing.T) {
		t.Parallel()

		message := FormatCurrencies(currencies, LanguageES)

		assert.Contains(t, message, "Monedas soportadas")
		assert.Contains(t, message, "USD")
		assert.Contains(t, message, "VES")
		assert.Contains(t, message, "EUR")
	})
}

func TestFormatter_StartMessage(t *testing.T) {
	t.Parallel()

	assert.Contains(t, StartMessage(LanguageEN), "Hello")
	assert.Contains(t, StartMessage(LanguageES), "Hola")
}

func TestFormatter_HelpMessage(t *testing.T) {
	t.Parallel()

	assert.Contains(t, HelpMessage(LanguageEN), "Commands")
	assert.Contains(t, HelpMessage(LanguageES), "Comandos") //nolint:misspell // Spanish copy
}

func TestFormatter_ErrorMessage(t *testing.T) {
	t.Parallel()

	err := errors.New("boom")

	assert.Contains(t, ErrorMessage(err, LanguageEN), "Error: boom")
	assert.Contains(t, ErrorMessage(err, LanguageES), "Error: boom")
}

func TestFormatter_InvalidUsageMessage(t *testing.T) {
	t.Parallel()

	assert.Contains(t, InvalidUsageMessage("/rate <base>", LanguageEN), "Usage: /rate <base>")
	assert.Contains(t, InvalidUsageMessage("/tasa <base>", LanguageES), "Uso: /tasa <base>")
}

func TestFormatter_GetEmoji(t *testing.T) {
	t.Parallel()

	emoji := getEmoji(types.CurrencyUSD)

	require.NotEmpty(t, emoji)
}
