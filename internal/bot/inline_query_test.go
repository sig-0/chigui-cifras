package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sig-0/chigui-cifras/internal/fxrates"

	"github.com/sig-0/fxrates/storage/types"
)

func TestInlineQuery_ValidRateEnglish(t *testing.T) {
	t.Parallel()

	var (
		asOf      = time.Date(2026, time.January, 2, 15, 4, 0, 0, time.UTC)
		fetchedAt = time.Date(2026, time.January, 2, 15, 5, 0, 0, time.UTC)

		response = fxrates.PageExchangeRate{
			Results: []fxrates.ExchangeRate{
				{
					Base:      types.CurrencyUSD,
					Target:    types.CurrencyVES,
					Rate:      42.1234,
					RateType:  types.RateTypeMID,
					Source:    types.SourceBCV,
					AsOf:      asOf,
					FetchedAt: fetchedAt,
				},
			},
			Total: 1,
		}
	)

	var fxPath string

	fxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fxPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	t.Cleanup(fxServer.Close)

	tgServer, requests := newInlineServer(t)
	t.Cleanup(tgServer.Close)

	client := fxrates.NewClient(fxServer.URL, time.Second)
	h := NewHandlers(client)
	b := newTelegramBot(t, tgServer.URL)

	update := &models.Update{
		InlineQuery: &models.InlineQuery{
			ID:    "inline-1",
			Query: "USD",
			From: &models.User{
				LanguageCode: "en-US",
			},
		},
	}

	h.InlineQuery(context.Background(), b, update)

	request := awaitInlineRequest(t, requests)

	assert.Equal(t, "inline-1", request.InlineQueryID)
	assert.Equal(t, "/v1/rates/USD/VES", fxPath)

	require.Len(t, request.Results, 1)
	result := request.Results[0]

	assert.Equal(t, "article", resultString(result, "type"))
	assert.Equal(t, "USD/VES", resultString(result, "title"))
	assert.Contains(t, resultString(result, "description"), "42.1234")

	message := resultMessageText(t, result)
	assert.Contains(t, message, "Rate:")
	assert.Contains(t, message, "USD")
	assert.Contains(t, message, "VES")
	assert.Contains(t, message, "VET")
}

func TestInlineQuery_HelpEnglish(t *testing.T) {
	t.Parallel()

	tgServer, requests := newInlineServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil)
	b := newTelegramBot(t, tgServer.URL)

	update := &models.Update{
		InlineQuery: &models.InlineQuery{
			ID:    "inline-2",
			Query: "",
			From: &models.User{
				LanguageCode: "en",
			},
		},
	}

	h.InlineQuery(context.Background(), b, update)

	request := awaitInlineRequest(t, requests)

	require.Len(t, request.Results, 1)
	result := request.Results[0]

	assert.Equal(t, "Help", resultString(result, "title"))
	assert.Contains(t, resultString(result, "description"), "USD VES")
	assert.Contains(t, resultMessageText(t, result), "Use: USD VES")
}

func TestInlineQuery_NoResultsSpanish(t *testing.T) {
	t.Parallel()

	response := fxrates.PageExchangeRate{}

	fxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	t.Cleanup(fxServer.Close)

	tgServer, requests := newInlineServer(t)
	t.Cleanup(tgServer.Close)

	client := fxrates.NewClient(fxServer.URL, time.Second)
	h := NewHandlers(client)
	b := newTelegramBot(t, tgServer.URL)

	update := &models.Update{
		InlineQuery: &models.InlineQuery{
			ID:    "inline-3",
			Query: "USD VES",
			From: &models.User{
				LanguageCode: "es-VE",
			},
		},
	}

	h.InlineQuery(context.Background(), b, update)

	request := awaitInlineRequest(t, requests)

	require.Len(t, request.Results, 1)
	result := request.Results[0]

	assert.Equal(t, "Sin resultados", resultString(result, "title"))
	assert.Contains(t, resultMessageText(t, result), "No se encontraron tasas para USD/VES")
}

func TestInlineQuery_ErrorSpanish(t *testing.T) {
	t.Parallel()

	fxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(fxServer.Close)

	tgServer, requests := newInlineServer(t)
	t.Cleanup(tgServer.Close)

	client := fxrates.NewClient(fxServer.URL, time.Second)
	h := NewHandlers(client)
	b := newTelegramBot(t, tgServer.URL)

	update := &models.Update{
		InlineQuery: &models.InlineQuery{
			ID:    "inline-4",
			Query: "USD VES",
			From: &models.User{
				LanguageCode: "es",
			},
		},
	}

	h.InlineQuery(context.Background(), b, update)

	request := awaitInlineRequest(t, requests)

	require.Len(t, request.Results, 1)
	result := request.Results[0]

	assert.Equal(t, "Error", resultString(result, "title"))
	assert.Contains(t, resultMessageText(t, result), "No se pudo obtener la tasa")
}

func TestInlineQuery_ParseInlineQuery(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name   string
		query  string
		base   string
		target string
		ok     bool
	}{
		{
			name:   "base only",
			query:  "USD",
			base:   "USD",
			target: "VES",
			ok:     true,
		},
		{
			name:   "base and target",
			query:  "USD EUR",
			base:   "USD",
			target: "EUR",
			ok:     true,
		},
		{
			name:   "slash",
			query:  "usd/ves",
			base:   "USD",
			target: "VES",
			ok:     true,
		},
		{
			name:   "dash",
			query:  "usd-eur",
			base:   "USD",
			target: "EUR",
			ok:     true,
		},
		{
			name:  "empty",
			query: " ",
			ok:    false,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base, target, ok := parseInlineQuery(testCase.query)

			assert.Equal(t, testCase.ok, ok)

			if testCase.ok {
				assert.Equal(t, testCase.base, base)
				assert.Equal(t, testCase.target, target)
			}
		})
	}
}

func TestInlineQuery_LanguageForInline(t *testing.T) {
	t.Parallel()

	h := NewHandlers(nil)

	assert.Equal(t, LanguageES, h.languageForInline(nil))
	assert.Equal(t, LanguageES, h.languageForInline(&models.InlineQuery{}))
	assert.Equal(t, LanguageEN, h.languageForInline(&models.InlineQuery{From: &models.User{LanguageCode: "en"}}))
	assert.Equal(t, LanguageEN, h.languageForInline(&models.InlineQuery{From: &models.User{LanguageCode: "en-US"}}))
	assert.Equal(t, LanguageES, h.languageForInline(&models.InlineQuery{From: &models.User{LanguageCode: "es-VE"}}))
}

type inlineRequest struct {
	InlineQueryID string
	Results       []map[string]any
}

func newInlineServer(t *testing.T) (*httptest.Server, <-chan inlineRequest) {
	t.Helper()

	requests := make(chan inlineRequest, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot"+"test-token"+"/answerInlineQuery" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}

		resultsRaw := r.FormValue("results")

		var results []map[string]any
		if resultsRaw != "" {
			if err := json.Unmarshal([]byte(resultsRaw), &results); err != nil {
				t.Errorf("decode results: %v", err)
			}
		}

		requests <- inlineRequest{
			InlineQueryID: r.FormValue("inline_query_id"),
			Results:       results,
		}

		w.Header().Set("Content-Type", "application/json")

		if _, err := w.Write([]byte(`{"ok":true,"result":true}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))

	return srv, requests
}

func newTelegramBot(t *testing.T, serverURL string) *tgbot.Bot {
	t.Helper()

	b, err := tgbot.New(
		"test-token",
		tgbot.WithServerURL(serverURL),
		tgbot.WithSkipGetMe(),
	)
	require.NoError(t, err)

	return b
}

func awaitInlineRequest(t *testing.T, requests <-chan inlineRequest) inlineRequest {
	t.Helper()

	select {
	case req := <-requests:
		return req
	case <-time.After(5 * time.Second):
		t.Fatal("inline request not received")
	}

	return inlineRequest{}
}

func resultString(result map[string]any, key string) string {
	value, _ := result[key].(string)

	return value
}

func resultMessageText(t *testing.T, result map[string]any) string {
	t.Helper()

	content, ok := result["input_message_content"].(map[string]any)
	require.True(t, ok)

	text, ok := content["message_text"].(string)
	require.True(t, ok)

	return text
}
