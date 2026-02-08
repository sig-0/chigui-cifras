package bot

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sig-0/fxrates/storage/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
	"github.com/sig-0/chigui-cifras/internal/storage"
)

const (
	testUSDRate  = 52.43
	testEURRate  = 57.10
	testUSDTRate = 51.90
)

// newRateServer returns an httptest server that serves all three
// rate endpoints (USD/VES, EUR/VES, USDT/VES) used by fetchDashboardMessage
func newRateServer(t *testing.T) *httptest.Server {
	t.Helper()

	asOf := time.Date(2026, time.February, 1, 8, 0, 0, 0, time.UTC)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var resp fxrates.PageExchangeRate

		switch r.URL.Path {
		case "/v1/rates/USD/VES":
			resp = fxrates.PageExchangeRate{
				Results: []fxrates.ExchangeRate{{
					Base: types.CurrencyUSD, Target: types.CurrencyVES,
					Rate: testUSDRate, RateType: types.RateTypeMID,
					Source: types.SourceBCV, AsOf: asOf,
				}},
				Total: 1,
			}
		case "/v1/rates/EUR/VES":
			resp = fxrates.PageExchangeRate{
				Results: []fxrates.ExchangeRate{{
					Base: types.CurrencyEUR, Target: types.CurrencyVES,
					Rate: testEURRate, RateType: types.RateTypeMID,
					Source: types.SourceBCV, AsOf: asOf,
				}},
				Total: 1,
			}
		case "/v1/rates/USDT/VES":
			resp = fxrates.PageExchangeRate{
				Results: []fxrates.ExchangeRate{{
					Base: types.CurrencyUSDT, Target: types.CurrencyVES,
					Rate: testUSDTRate, RateType: types.RateTypeBUY,
					Source: "P2P", AsOf: asOf,
				}},
				Total: 1,
			}
		default:
			w.WriteHeader(http.StatusNotFound)

			return
		}

		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
}

func newTestScheduler(store storage.Store, fxClient *fxrates.Client, sender MessageSender) *Scheduler {
	return NewScheduler(
		store,
		fxClient,
		sender,
		time.Minute,
		time.Minute,
		slog.Default(),
	)
}

func TestBroadcastDue_SendsToSubscribers(t *testing.T) {
	t.Parallel()

	var updatedChats []int64

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 1, Frequency: storage.FrequencyHourly},
				{ChatID: 2, Frequency: storage.FrequencyDaily},
			}, nil
		},
		updateNextSendFn: func(_ context.Context, chatID int64, _ time.Time) error {
			updatedChats = append(updatedChats, chatID)

			return nil
		},
	}

	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	var sent []sentMessage

	sender := &mockSender{
		sendHTMLMessageFn: func(_ context.Context, chatID int64, text string) error {
			sent = append(sent, sentMessage{ChatID: chatID, Text: text})

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.broadcastDue(context.Background())

	require.Len(t, sent, 2)
	assert.Equal(t, int64(1), sent[0].ChatID)
	assert.Equal(t, int64(2), sent[1].ChatID)
	assert.Contains(t, sent[0].Text, "USD/VES")
	assert.Contains(t, sent[0].Text, "EUR/VES")

	assert.Equal(t, []int64{1, 2}, updatedChats)
}

func TestBroadcastDue_NoDueSubscribers(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return nil, nil
		},
	}

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient("http://unused", time.Second), sender)

	sched.broadcastDue(context.Background())

	assert.Zero(t, sendCalls)
}

func TestBroadcastDue_StoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return nil, errors.New("db down")
		},
	}

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient("http://unused", time.Second), sender)

	sched.broadcastDue(context.Background())

	assert.Zero(t, sendCalls)
}

func TestBroadcastDue_SendFailureUnsubscribes(t *testing.T) {
	t.Parallel()

	var unsubscribedChats []int64

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 1, Frequency: storage.FrequencyDaily},
			}, nil
		},
		unsubscribeFn: func(_ context.Context, chatID int64) error {
			unsubscribedChats = append(unsubscribedChats, chatID)

			return nil
		},
	}

	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			return errors.New("chat not found")
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.broadcastDue(context.Background())

	assert.Equal(t, []int64{1}, unsubscribedChats)
}

func TestBroadcastDue_APIErrorSkipsSend(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 1, Frequency: storage.FrequencyDaily},
			}, nil
		},
	}

	// Server that always returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.broadcastDue(context.Background())

	assert.Zero(t, sendCalls)
}

func TestBroadcastDue_UpdateNextSendUsesCorrectInterval(t *testing.T) {
	t.Parallel()

	recorded := make(map[int64]time.Time)

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 10, Frequency: storage.FrequencyHourly},
				{ChatID: 20, Frequency: storage.FrequencyDaily},
			}, nil
		},
		updateNextSendFn: func(_ context.Context, chatID int64, nextSendAt time.Time) error {
			recorded[chatID] = nextSendAt

			return nil
		},
	}

	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), &mockSender{})

	before := time.Now()

	sched.broadcastDue(context.Background())

	after := time.Now()

	require.Len(t, recorded, 2)

	// Hourly subscriber: next_send_at should be ~1h from now
	hourlyNext := recorded[int64(10)]
	assert.True(t, hourlyNext.After(before.Add(time.Hour-time.Second)))
	assert.True(t, hourlyNext.Before(after.Add(time.Hour+time.Second)))

	// Daily subscriber: next_send_at should be ~24h from now
	dailyNext := recorded[int64(20)]
	assert.True(t, dailyNext.After(before.Add(24*time.Hour-time.Second)))
	assert.True(t, dailyNext.Before(after.Add(24*time.Hour+time.Second)))
}

func TestCheckAlerts_TriggersAbove(t *testing.T) {
	t.Parallel()

	var triggeredIDs []string

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return []*storage.Alert{
				{ID: "a1", ChatID: 42, Base: "USD", Direction: storage.DirectionAbove, Threshold: 50},
			}, nil
		},
		triggerAlertFn: func(_ context.Context, id string) error {
			triggeredIDs = append(triggeredIDs, id)

			return nil
		},
	}

	// Rate is 52.43, above threshold of 50
	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	var sent []sentMessage

	sender := &mockSender{
		sendHTMLMessageFn: func(_ context.Context, chatID int64, text string) error {
			sent = append(sent, sentMessage{ChatID: chatID, Text: text})

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.checkAlerts(context.Background())

	require.Len(t, sent, 1)
	assert.Equal(t, int64(42), sent[0].ChatID)
	assert.Contains(t, sent[0].Text, "¡Alerta!")
	assert.Contains(t, sent[0].Text, "52.43")

	assert.Equal(t, []string{"a1"}, triggeredIDs)
}

func TestCheckAlerts_TriggersBelow(t *testing.T) {
	t.Parallel()

	var triggeredIDs []string

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return []*storage.Alert{
				{ID: "b1", ChatID: 99, Base: "USD", Direction: storage.DirectionBelow, Threshold: 55},
			}, nil
		},
		triggerAlertFn: func(_ context.Context, id string) error {
			triggeredIDs = append(triggeredIDs, id)

			return nil
		},
	}

	// Rate is 52.43, below threshold of 55
	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	var sent []sentMessage

	sender := &mockSender{
		sendHTMLMessageFn: func(_ context.Context, chatID int64, text string) error {
			sent = append(sent, sentMessage{ChatID: chatID, Text: text})

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.checkAlerts(context.Background())

	require.Len(t, sent, 1)
	assert.Equal(t, int64(99), sent[0].ChatID)
	assert.Contains(t, sent[0].Text, "bajó de")

	assert.Equal(t, []string{"b1"}, triggeredIDs)
}

func TestCheckAlerts_DoesNotTriggerWhenNotMet(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return []*storage.Alert{
				{ID: "a1", ChatID: 42, Base: "USD", Direction: storage.DirectionAbove, Threshold: 60},
				{ID: "b1", ChatID: 42, Base: "USD", Direction: storage.DirectionBelow, Threshold: 40},
			}, nil
		},
	}

	// Rate is 52.43: not >= 60 and not <= 40
	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.checkAlerts(context.Background())

	assert.Zero(t, sendCalls)
}

func TestCheckAlerts_NoActiveAlerts(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return nil, nil
		},
	}

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient("http://unused", time.Second), sender)

	sched.checkAlerts(context.Background())

	assert.Zero(t, sendCalls)
}

func TestCheckAlerts_StoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return nil, errors.New("db down")
		},
	}

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient("http://unused", time.Second), sender)

	sched.checkAlerts(context.Background())

	assert.Zero(t, sendCalls)
}

func TestCheckAlerts_SendFailureStillTriggers(t *testing.T) {
	t.Parallel()

	var triggeredIDs []string

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return []*storage.Alert{
				{ID: "a1", ChatID: 42, Base: "USD", Direction: storage.DirectionAbove, Threshold: 50},
			}, nil
		},
		triggerAlertFn: func(_ context.Context, id string) error {
			triggeredIDs = append(triggeredIDs, id)

			return nil
		},
	}

	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			return errors.New("chat blocked")
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.checkAlerts(context.Background())

	// Alert should still be marked as triggered even though send failed
	assert.Equal(t, []string{"a1"}, triggeredIDs)
}

func TestCheckAlerts_APIErrorSkipsBase(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return []*storage.Alert{
				{ID: "a1", ChatID: 42, Base: "USD", Direction: storage.DirectionAbove, Threshold: 50},
			}, nil
		},
	}

	// Server that always returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.checkAlerts(context.Background())

	assert.Zero(t, sendCalls)
}

func TestCheckAlerts_MultipleBaseCurrencies(t *testing.T) {
	t.Parallel()

	var triggeredIDs []string

	store := &mockStore{
		activeAlertsFn: func(_ context.Context) ([]*storage.Alert, error) {
			return []*storage.Alert{
				{ID: "u1", ChatID: 42, Base: "USD", Direction: storage.DirectionAbove, Threshold: 50},
				{ID: "e1", ChatID: 42, Base: "EUR", Direction: storage.DirectionAbove, Threshold: 55},
			}, nil
		},
		triggerAlertFn: func(_ context.Context, id string) error {
			triggeredIDs = append(triggeredIDs, id)

			return nil
		},
	}

	// USD=52.43 (above 50), EUR=57.10 (above 55)
	srv := newRateServer(t)
	t.Cleanup(srv.Close)

	var sent []sentMessage

	sender := &mockSender{
		sendHTMLMessageFn: func(_ context.Context, chatID int64, text string) error {
			sent = append(sent, sentMessage{ChatID: chatID, Text: text})

			return nil
		},
	}
	sched := newTestScheduler(store, fxrates.NewClient(srv.URL, time.Second), sender)

	sched.checkAlerts(context.Background())

	assert.Len(t, sent, 2)
	assert.Len(t, triggeredIDs, 2)
	assert.Contains(t, triggeredIDs, "u1")
	assert.Contains(t, triggeredIDs, "e1")
}

func TestSchedulerRun_StopsOnCancel(t *testing.T) {
	t.Parallel()

	sched := NewScheduler(
		&mockStore{},
		fxrates.NewClient("http://unused", time.Second),
		&mockSender{},
		time.Hour, // long intervals so tickers don't fire
		time.Hour,
		slog.Default(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := sched.Run(ctx)
	assert.NoError(t, err)
}

func TestIntervalForFrequency(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Hour, intervalForFrequency(storage.FrequencyHourly))
	assert.Equal(t, 24*time.Hour, intervalForFrequency(storage.FrequencyDaily))
	assert.Equal(t, 24*time.Hour, intervalForFrequency("unknown"))
}
