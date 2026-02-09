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

	now := time.Now()

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 1, Frequency: storage.FrequencyHourly, NextSendAt: now},
				{ChatID: 2, Frequency: storage.FrequencyDaily, NextSendAt: now},
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
				{ChatID: 1, Frequency: storage.FrequencyDaily, NextSendAt: time.Now()},
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
				{ChatID: 1, Frequency: storage.FrequencyDaily, NextSendAt: time.Now()},
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

	// NextSendAt just before now simulates a normal on-time tick
	justNow := time.Now().Add(-time.Second)

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 10, Frequency: storage.FrequencyHourly, NextSendAt: justNow},
				{ChatID: 20, Frequency: storage.FrequencyDaily, NextSendAt: justNow},
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

	sched.broadcastDue(context.Background())

	require.Len(t, recorded, 2)

	// Hourly subscriber
	assert.Equal(t, justNow.Add(time.Hour), recorded[int64(10)])

	// Daily subscriber
	assert.Equal(t, justNow.Add(24*time.Hour), recorded[int64(20)])
}

func TestBroadcastDue_UsesSingleCycleNow(t *testing.T) {
	t.Parallel()

	recorded := make(map[int64]time.Time)
	now := time.Date(2026, time.February, 9, 12, 0, 0, 0, time.UTC)
	justBeforeNow := now.Add(-time.Second)

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 10, Frequency: storage.FrequencyHourly, NextSendAt: justBeforeNow},
				{ChatID: 20, Frequency: storage.FrequencyDaily, NextSendAt: justBeforeNow},
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

	var nowCalls int

	sched.nowFn = func() time.Time {
		nowCalls++

		return now
	}

	sched.broadcastDue(context.Background())

	assert.Equal(t, 1, nowCalls)
	require.Len(t, recorded, 2)
	assert.Equal(t, justBeforeNow.Add(time.Hour), recorded[int64(10)])
	assert.Equal(t, justBeforeNow.Add(24*time.Hour), recorded[int64(20)])
}

func TestBroadcastDue_SkipsStaleSubscribers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.February, 9, 12, 0, 0, 0, time.UTC)
	recorded := make(map[int64]time.Time)

	store := &mockStore{
		dueSubscribersFn: func(_ context.Context) ([]*storage.Subscriber, error) {
			return []*storage.Subscriber{
				{ChatID: 1, Frequency: storage.FrequencyHourly, NextSendAt: now.Add(-2 * time.Hour)},
				{ChatID: 2, Frequency: storage.FrequencyDaily, NextSendAt: now.Add(-48 * time.Hour)},
			}, nil
		},
		updateNextSendFn: func(_ context.Context, chatID int64, nextSendAt time.Time) error {
			recorded[chatID] = nextSendAt

			return nil
		},
	}

	var sendCalls int

	sender := &mockSender{
		sendHTMLMessageFn: func(context.Context, int64, string) error {
			sendCalls++

			return nil
		},
	}

	// Invalid URL on purpose, if stale subscribers are skipped correctly,
	// no API request is needed and updates still occur
	sched := newTestScheduler(store, fxrates.NewClient("http://unused", time.Second), sender)
	sched.nowFn = func() time.Time { return now }

	sched.broadcastDue(context.Background())

	assert.Zero(t, sendCalls)
	require.Len(t, recorded, 2)
	assert.Equal(t, now.Add(time.Hour), recorded[int64(1)])
	assert.Equal(t, now.Add(24*time.Hour), recorded[int64(2)])
}

func TestNextSendAfter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.February, 9, 12, 0, 0, 0, time.UTC)

	testTable := []struct {
		lastSendAt time.Time
		want       time.Time
		name       string
		interval   time.Duration
	}{
		{
			name:       "hourly one second late",
			lastSendAt: now.Add(-time.Second),
			interval:   time.Hour,
			want:       now.Add(time.Hour - time.Second), // 12:59:59
		},
		{
			name:       "daily five minutes late",
			lastSendAt: now.Add(-5 * time.Minute),
			interval:   24 * time.Hour,
			want:       now.Add(24*time.Hour - 5*time.Minute), // tomorrow 11:55
		},

		{
			name:       "hourly exactly on boundary",
			lastSendAt: now.Add(-time.Hour),
			interval:   time.Hour,
			want:       now, // not before now, so no fast-forward needed
		},
		{
			name:       "daily exactly on boundary",
			lastSendAt: now.Add(-24 * time.Hour),
			interval:   24 * time.Hour,
			want:       now,
		},

		{
			name:       "hourly one hour one second late",
			lastSendAt: now.Add(-time.Hour - time.Second),
			interval:   time.Hour,
			want:       now.Add(time.Hour - time.Second), // skips the missed one
		},

		{
			name:       "hourly two and a half hours late",
			lastSendAt: now.Add(-2*time.Hour - 30*time.Minute),
			interval:   time.Hour,
			want:       now.Add(30 * time.Minute), // 12:30, next aligned slot
		},
		{
			name:       "daily one and a half days late",
			lastSendAt: now.Add(-36 * time.Hour),
			interval:   24 * time.Hour,
			want:       now.Add(12 * time.Hour), // today at midnight
		},

		{
			name:       "hourly three days late",
			lastSendAt: now.Add(-72 * time.Hour),
			interval:   time.Hour,
			want:       now.Add(time.Hour), // 13:00, skips all 72 missed
		},
		{
			name:       "daily ten days late",
			lastSendAt: now.Add(-10 * 24 * time.Hour),
			interval:   24 * time.Hour,
			want:       now.Add(24 * time.Hour), // tomorrow, skips all 10 missed
		},
		{
			name:       "daily ten days and six hours late",
			lastSendAt: now.Add(-10*24*time.Hour - 6*time.Hour),
			interval:   24 * time.Hour,
			want:       now.Add(18 * time.Hour), // 11 intervals from lastSendAt
		},

		{
			name:       "hourly exactly five hours late",
			lastSendAt: now.Add(-5 * time.Hour),
			interval:   time.Hour,
			want:       now.Add(time.Hour), // n=5, next = lastSendAt + 6h
		},
		{
			name:       "daily exactly seven days late",
			lastSendAt: now.Add(-7 * 24 * time.Hour),
			interval:   24 * time.Hour,
			want:       now.Add(24 * time.Hour), // n=7, next = lastSendAt + 8d
		},

		{
			name:       "hourly one millisecond late",
			lastSendAt: now.Add(-time.Millisecond),
			interval:   time.Hour,
			want:       now.Add(time.Hour - time.Millisecond),
		},
		{
			name:       "hourly one hour and one millisecond late",
			lastSendAt: now.Add(-time.Hour - time.Millisecond),
			interval:   time.Hour,
			want:       now.Add(time.Hour - time.Millisecond), // fast-forwards past the missed one
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := nextSendAfter(now, testCase.lastSendAt, testCase.interval)
			assert.Equal(t, testCase.want, got)
			assert.False(t, got.Before(now), "next send must not be in the past")
		})
	}
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
