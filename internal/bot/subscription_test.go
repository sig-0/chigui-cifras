package bot

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sig-0/chigui-cifras/internal/storage"
)

func TestSubscribe_HourlySuccess(t *testing.T) {
	t.Parallel()

	var calledFreq string

	store := &mockStore{
		subscribeFn: func(_ context.Context, _ int64, frequency string, _ time.Time) error {
			calledFreq = frequency

			return nil
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/suscribir horario",
			Chat: models.Chat{ID: 42},
		},
	}

	h.Subscribe(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Equal(t, int64(42), msg.ChatID)
	assert.Contains(t, msg.Text, "Suscrito")
	assert.Contains(t, msg.Text, "horario")
	assert.Equal(t, storage.FrequencyHourly, calledFreq)
}

func TestSubscribe_InvalidFrequency(t *testing.T) {
	t.Parallel()

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), &mockStore{})

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/suscribir semanal",
			Chat: models.Chat{ID: 42},
		},
	}

	h.Subscribe(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "Uso inválido")
}

func TestSubscribe_NoArgs(t *testing.T) {
	t.Parallel()

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), &mockStore{})

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/suscribir",
			Chat: models.Chat{ID: 42},
		},
	}

	h.Subscribe(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "Uso inválido")
}

func TestUnsubscribe_Success(t *testing.T) {
	t.Parallel()

	var calledChatID int64

	store := &mockStore{
		unsubscribeFn: func(_ context.Context, chatID int64) error {
			calledChatID = chatID

			return nil
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/desuscribir",
			Chat: models.Chat{ID: 99},
		},
	}

	h.Unsubscribe(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "desuscrito")
	assert.Equal(t, int64(99), calledChatID)
}

func TestCreateAlert_Success(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		createAlertFn: func(
			_ context.Context, id string, chatID int64, base, direction string, threshold float64,
		) (*storage.Alert, error) {
			return &storage.Alert{
				ID: id, ChatID: chatID, Base: base,
				Direction: direction, Threshold: threshold,
			}, nil
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/alerta USD arriba 55",
			Chat: models.Chat{ID: 42},
		},
	}

	h.CreateAlert(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "Alerta creada")
	assert.Contains(t, msg.Text, "USD/VES")
	assert.Contains(t, msg.Text, "por encima de")
}

func TestCreateAlert_LimitReached(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		createAlertFn: func(context.Context, string, int64, string, string, float64) (*storage.Alert, error) {
			return nil, storage.ErrAlertLimitReached
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/alerta USD arriba 55",
			Chat: models.Chat{ID: 42},
		},
	}

	h.CreateAlert(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "5 alertas")
}

func TestCreateAlert_InvalidArgs(t *testing.T) {
	t.Parallel()

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), &mockStore{})

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/alerta USD",
			Chat: models.Chat{ID: 42},
		},
	}

	h.CreateAlert(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "Uso inválido")
}

func TestCreateAlert_InvalidThreshold(t *testing.T) {
	t.Parallel()

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), &mockStore{})

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/alerta USD arriba abc",
			Chat: models.Chat{ID: 42},
		},
	}

	h.CreateAlert(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "número positivo")
}

func TestListAlerts_WithAlerts(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		alertsByChatFn: func(_ context.Context, _ int64) ([]*storage.Alert, error) {
			return []*storage.Alert{
				{ID: "a1", Base: "USD", Direction: storage.DirectionAbove, Threshold: 55},
			}, nil
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/alertas",
			Chat: models.Chat{ID: 42},
		},
	}

	h.ListAlerts(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "alertas activas")
	assert.Contains(t, msg.Text, "a1")
}

func TestListAlerts_Empty(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		alertsByChatFn: func(_ context.Context, _ int64) ([]*storage.Alert, error) {
			return nil, nil
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/alertas",
			Chat: models.Chat{ID: 42},
		},
	}

	h.ListAlerts(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "No tienes alertas activas")
}

func TestDeleteAlert_Success(t *testing.T) {
	t.Parallel()

	var deletedID string

	store := &mockStore{
		deleteAlertFn: func(_ context.Context, id string, _ int64) error {
			deletedID = id

			return nil
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/borraralerta abc123",
			Chat: models.Chat{ID: 42},
		},
	}

	h.DeleteAlert(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "Alerta eliminada")
	assert.Equal(t, "abc123", deletedID)
}

func TestDeleteAlert_NoArgs(t *testing.T) {
	t.Parallel()

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), &mockStore{})

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/borraralerta",
			Chat: models.Chat{ID: 42},
		},
	}

	h.DeleteAlert(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "Uso inválido")
}

func TestSubscribe_StoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		subscribeFn: func(_ context.Context, _ int64, _ string, _ time.Time) error {
			return errors.New("db down")
		},
	}

	tgServer, messages := newMessageServer(t)
	t.Cleanup(tgServer.Close)

	h := NewHandlers(nil, slog.Default(), store)

	b, err := tgbot.New("test-token", tgbot.WithServerURL(tgServer.URL), tgbot.WithSkipGetMe())
	require.NoError(t, err)

	update := &models.Update{
		Message: &models.Message{
			Text: "/suscribir diario",
			Chat: models.Chat{ID: 42},
		},
	}

	h.Subscribe(context.Background(), b, update)

	msg := awaitMessage(t, messages)
	assert.Contains(t, msg.Text, "Error")
	assert.Contains(t, msg.Text, "db down")
}
