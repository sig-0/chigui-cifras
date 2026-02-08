package bot

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sig-0/fxrates/storage/types"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
	"github.com/sig-0/chigui-cifras/internal/storage"
)

func TestFormatter_SpanishMonth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ene", spanishMonth(time.January))
	assert.Equal(t, "feb", spanishMonth(time.February))
	assert.Equal(t, "dic", spanishMonth(time.December))
	assert.Equal(t, "", spanishMonth(0))
	assert.Equal(t, "", spanishMonth(13))
}

func TestFormatter_RateTypeLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Tasa media", rateTypeLabel(types.RateTypeMID))
	assert.Equal(t, "Compra", rateTypeLabel(types.RateTypeBUY))
	assert.Equal(t, "Venta", rateTypeLabel(types.RateTypeSELL))
	assert.Equal(t, "Tasa media", rateTypeLabel("OTHER"))
}

func TestFormatter_FormatTime(t *testing.T) {
	t.Parallel()

	// 2026-01-02 15:04 UTC → 2026-01-02 11:04 VET
	ts := time.Date(2026, time.January, 2, 15, 4, 0, 0, time.UTC)
	result := formatTime(ts)

	assert.Equal(t, "02 ene 2026, 11:04 VET", result)
}

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

	message := FormatRate(rate)

	assert.Contains(t, message, "<b>USD → VES</b>")
	assert.Contains(t, message, "<code>42.00</code>")
	assert.Contains(t, message, "Bs")
	assert.Contains(t, message, "BCV")
	assert.Contains(t, message, "Tasa media")
	assert.Contains(t, message, "02 ene 2026, 11:04 VET")
	assert.Contains(t, message, "Mozaik Pay")
	assert.Contains(t, message, `href="https://mozaik.money"`)
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

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "No se encontraron tasas", FormatRates(nil))
	})

	t.Run("with rates", func(t *testing.T) {
		t.Parallel()

		message := FormatRates(rates)

		assert.Contains(t, message, "<b>Tasas de USD</b>")
		assert.Contains(t, message, "VES")
		assert.Contains(t, message, "<code>40.00</code>")
		assert.Contains(t, message, "EUR")
		assert.Contains(t, message, "<code>0.90</code>")
		assert.Contains(t, message, "BCV")
		assert.Contains(t, message, "Tasa media")
		assert.Contains(t, message, "02 ene 2026, 11:04 VET")
		assert.Contains(t, message, "Mozaik Pay")
	})
}

func TestFormatter_FormatDashboard(t *testing.T) {
	t.Parallel()

	bcvTime := time.Date(2026, time.February, 1, 12, 0, 0, 0, time.UTC)
	p2pTime := time.Date(2026, time.February, 1, 14, 30, 0, 0, time.UTC)

	usd := &fxrates.ExchangeRate{
		Base: types.CurrencyUSD, Target: types.CurrencyVES,
		Rate: 52.43, RateType: types.RateTypeMID, Source: types.SourceBCV, AsOf: bcvTime,
	}
	eur := &fxrates.ExchangeRate{
		Base: types.CurrencyEUR, Target: types.CurrencyVES,
		Rate: 56.12, RateType: types.RateTypeMID, Source: types.SourceBCV, AsOf: bcvTime,
	}
	usdtRates := []fxrates.ExchangeRate{
		{
			Base: types.CurrencyUSDT, Target: types.CurrencyVES,
			Rate: 51.80, RateType: types.RateTypeBUY, Source: "P2P", AsOf: p2pTime,
		},
		{
			Base: types.CurrencyUSDT, Target: types.CurrencyVES,
			Rate: 52.10, RateType: types.RateTypeSELL, Source: "P2P", AsOf: p2pTime,
		},
	}

	message := FormatDashboard(usd, eur, usdtRates)

	assert.Contains(t, message, "<b>Tasas del día</b>")
	assert.Contains(t, message, "USD/VES")
	assert.Contains(t, message, "<code>52.43</code>")
	assert.Contains(t, message, "EUR/VES")
	assert.Contains(t, message, "<code>56.12</code>")
	assert.Contains(t, message, "USDT/VES")
	assert.Contains(t, message, "Compra:")
	assert.Contains(t, message, "<code>51.80</code>")
	assert.Contains(t, message, "Venta:")
	assert.Contains(t, message, "<code>52.10</code>")
	// Separate timestamps for BCV and P2P
	assert.Contains(t, message, "BCV · 01 feb 2026, 08:00 VET")
	assert.Contains(t, message, "P2P · 01 feb 2026, 10:30 VET")
	assert.Contains(t, message, "Mozaik Pay")
}

func TestFormatter_FormatDashboard_Partial(t *testing.T) {
	t.Parallel()

	rateTime := time.Date(2026, time.February, 1, 12, 0, 0, 0, time.UTC)

	usd := &fxrates.ExchangeRate{
		Base: types.CurrencyUSD, Target: types.CurrencyVES,
		Rate: 52.43, RateType: types.RateTypeMID, Source: types.SourceBCV, AsOf: rateTime,
	}

	message := FormatDashboard(usd, nil, nil)

	assert.Contains(t, message, "USD/VES")
	assert.Contains(t, message, "<code>52.43</code>")
	assert.NotContains(t, message, "EUR/VES")
	assert.NotContains(t, message, "USDT/VES")
}

func TestFormatter_FormatCurrencies(t *testing.T) {
	t.Parallel()

	currencyList := []fxrates.Currency{
		types.CurrencyUSD,
		types.CurrencyVES,
		types.CurrencyEUR,
	}

	message := FormatCurrencies(currencyList)

	assert.Contains(t, message, "<b>Monedas soportadas</b>")
	assert.Contains(t, message, "USD")
	assert.Contains(t, message, "VES")
	assert.Contains(t, message, "EUR")
}

func TestFormatter_StartMessage(t *testing.T) {
	t.Parallel()

	message := StartMessage()

	assert.Contains(t, message, "Hola")
	assert.Contains(t, message, "/dolar")
	assert.Contains(t, message, "/ayuda")
	assert.Contains(t, message, "Mozaik Pay")
	assert.Contains(t, message, "mover tu dinero en LATAM")
	assert.Contains(t, message, "utm_campaign=chiguicifras")
	assert.Contains(t, message, "Únete al acceso anticipado")
}

func TestFormatter_HelpMessage(t *testing.T) {
	t.Parallel()

	message := HelpMessage()

	assert.Contains(t, message, "Comandos") //nolint:misspell // Spanish copy
	assert.Contains(t, message, "/tasa")
	assert.Contains(t, message, "/dolar")
	assert.Contains(t, message, "/monedas")
	assert.Contains(t, message, "/suscribir")
	assert.Contains(t, message, "/desuscribir")
	assert.Contains(t, message, "/alerta")
	assert.Contains(t, message, "/alertas")
	assert.Contains(t, message, "/borraralerta")
}

func TestFormatter_ErrorMessage(t *testing.T) {
	t.Parallel()

	err := errors.New("boom")

	assert.Contains(t, ErrorMessage(err), "Error: boom")
}

func TestFormatter_InvalidUsageMessage(t *testing.T) {
	t.Parallel()

	assert.Contains(t, InvalidUsageMessage("/tasa <base>"), "Uso: /tasa <base>")
}

func TestFormatter_GetEmoji(t *testing.T) {
	t.Parallel()

	emoji := emojiForCurrency(types.CurrencyUSD)

	require.NotEmpty(t, emoji)
}

func TestFormatter_FormatSubscribed(t *testing.T) {
	t.Parallel()

	t.Run("hourly", func(t *testing.T) {
		t.Parallel()

		msg := FormatSubscribed(storage.FrequencyHourly)
		assert.Contains(t, msg, "Suscrito")
		assert.Contains(t, msg, "horario")
	})

	t.Run("daily", func(t *testing.T) {
		t.Parallel()

		msg := FormatSubscribed(storage.FrequencyDaily)
		assert.Contains(t, msg, "Suscrito")
		assert.Contains(t, msg, "diario")
	})
}

func TestFormatter_FormatUnsubscribed(t *testing.T) {
	t.Parallel()

	msg := FormatUnsubscribed()
	assert.Contains(t, msg, "desuscrito")
}

func TestFormatter_FormatAlertCreated(t *testing.T) {
	t.Parallel()

	alert := storage.Alert{
		ID:        "test123",
		ChatID:    42,
		Base:      "USD",
		Direction: storage.DirectionAbove,
		Threshold: 55.5,
		CreatedAt: time.Now(),
	}

	msg := FormatAlertCreated(alert)

	assert.Contains(t, msg, "Alerta creada")
	assert.Contains(t, msg, "USD/VES")
	assert.Contains(t, msg, "por encima de")
	assert.Contains(t, msg, "55.50")
	assert.Contains(t, msg, "test123")
}

func TestFormatter_FormatAlertCreated_Below(t *testing.T) {
	t.Parallel()

	alert := storage.Alert{
		ID:        "abc",
		Base:      "EUR",
		Direction: storage.DirectionBelow,
		Threshold: 50,
	}

	msg := FormatAlertCreated(alert)

	assert.Contains(t, msg, "por debajo de")
	assert.Contains(t, msg, "EUR/VES")
}

func TestFormatter_FormatAlerts(t *testing.T) {
	t.Parallel()

	alerts := []*storage.Alert{
		{ID: "a1", Base: "USD", Direction: storage.DirectionAbove, Threshold: 55},
		{ID: "a2", Base: "EUR", Direction: storage.DirectionBelow, Threshold: 50},
	}

	msg := FormatAlerts(alerts)

	assert.Contains(t, msg, "alertas activas")
	assert.Contains(t, msg, "USD/VES")
	assert.Contains(t, msg, "arriba")
	assert.Contains(t, msg, "55.00")
	assert.Contains(t, msg, "EUR/VES")
	assert.Contains(t, msg, "abajo")
	assert.Contains(t, msg, "50.00")
	assert.Contains(t, msg, "a1")
	assert.Contains(t, msg, "a2")
}

func TestFormatter_FormatAlertTriggered(t *testing.T) {
	t.Parallel()

	alert := storage.Alert{
		ID:        "x1",
		Base:      "USD",
		Direction: storage.DirectionAbove,
		Threshold: 55,
	}

	msg := FormatAlertTriggered(alert, 56.5)

	assert.Contains(t, msg, "Alerta")
	assert.Contains(t, msg, "USD/VES")
	assert.Contains(t, msg, "superó")
	assert.Contains(t, msg, "55.00")
	assert.Contains(t, msg, "56.50")
}

func TestFormatter_FormatAlertTriggered_Below(t *testing.T) {
	t.Parallel()

	alert := storage.Alert{
		ID:        "x2",
		Base:      "EUR",
		Direction: storage.DirectionBelow,
		Threshold: 50,
	}

	msg := FormatAlertTriggered(alert, 49.5)

	assert.Contains(t, msg, "bajó de")
}

func TestFormatter_FormatNoAlerts(t *testing.T) {
	t.Parallel()

	msg := FormatNoAlerts()
	assert.Contains(t, msg, "No tienes alertas activas")
	assert.Contains(t, msg, "/alerta")
}

func TestFormatter_FormatAlertDeleted(t *testing.T) {
	t.Parallel()

	msg := FormatAlertDeleted()
	assert.Contains(t, msg, "Alerta eliminada")
}

func TestFormatter_FormatAlertLimitReached(t *testing.T) {
	t.Parallel()

	msg := FormatAlertLimitReached()
	assert.Contains(t, msg, "5 alertas")
	assert.Contains(t, msg, "/borraralerta")
}
