package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sig-0/chigui-cifras/internal/storage"
)

type (
	subscribeDelegate       func(context.Context, int64, string, time.Time) error
	unsubscribeDelegate     func(context.Context, int64) error
	dueSubscribersDelegate  func(context.Context) ([]*storage.Subscriber, error)
	updateNextSendDelegate  func(context.Context, int64, time.Time) error
	createAlertDelegate     func(context.Context, string, int64, string, string, float64) (*storage.Alert, error)
	alertsByChatDelegate    func(context.Context, int64) ([]*storage.Alert, error)
	deleteAlertDelegate     func(context.Context, string, int64) error
	activeAlertsDelegate    func(context.Context) ([]*storage.Alert, error)
	triggerAlertDelegate    func(context.Context, string) error
	sendHTMLMessageDelegate func(context.Context, int64, string) error
)

// mockStore implements storage.Store for testing
type mockStore struct {
	subscribeFn      subscribeDelegate
	unsubscribeFn    unsubscribeDelegate
	dueSubscribersFn dueSubscribersDelegate
	updateNextSendFn updateNextSendDelegate
	createAlertFn    createAlertDelegate
	alertsByChatFn   alertsByChatDelegate
	deleteAlertFn    deleteAlertDelegate
	activeAlertsFn   activeAlertsDelegate
	triggerAlertFn   triggerAlertDelegate
}

func (m *mockStore) Subscribe(ctx context.Context, chatID int64, frequency string, nextSendAt time.Time) error {
	if m.subscribeFn != nil {
		return m.subscribeFn(ctx, chatID, frequency, nextSendAt)
	}

	return nil
}

func (m *mockStore) Unsubscribe(ctx context.Context, chatID int64) error {
	if m.unsubscribeFn != nil {
		return m.unsubscribeFn(ctx, chatID)
	}

	return nil
}

func (m *mockStore) DueSubscribers(ctx context.Context) ([]*storage.Subscriber, error) {
	if m.dueSubscribersFn != nil {
		return m.dueSubscribersFn(ctx)
	}

	return nil, nil
}

func (m *mockStore) UpdateNextSend(ctx context.Context, chatID int64, nextSendAt time.Time) error {
	if m.updateNextSendFn != nil {
		return m.updateNextSendFn(ctx, chatID, nextSendAt)
	}

	return nil
}

func (m *mockStore) CreateAlert(
	ctx context.Context, id string, chatID int64, base, direction string, threshold float64,
) (*storage.Alert, error) {
	if m.createAlertFn != nil {
		return m.createAlertFn(ctx, id, chatID, base, direction, threshold)
	}

	return nil, nil
}

func (m *mockStore) AlertsByChat(ctx context.Context, chatID int64) ([]*storage.Alert, error) {
	if m.alertsByChatFn != nil {
		return m.alertsByChatFn(ctx, chatID)
	}

	return nil, nil
}

func (m *mockStore) DeleteAlert(ctx context.Context, id string, chatID int64) error {
	if m.deleteAlertFn != nil {
		return m.deleteAlertFn(ctx, id, chatID)
	}

	return nil
}

func (m *mockStore) ActiveAlerts(ctx context.Context) ([]*storage.Alert, error) {
	if m.activeAlertsFn != nil {
		return m.activeAlertsFn(ctx)
	}

	return nil, nil
}

func (m *mockStore) TriggerAlert(ctx context.Context, id string) error {
	if m.triggerAlertFn != nil {
		return m.triggerAlertFn(ctx, id)
	}

	return nil
}

// mockSender implements MessageSender for testing
type mockSender struct {
	sendHTMLMessageFn sendHTMLMessageDelegate
}

func (m *mockSender) SendHTMLMessage(ctx context.Context, chatID int64, text string) error {
	if m.sendHTMLMessageFn != nil {
		return m.sendHTMLMessageFn(ctx, chatID, text)
	}

	return nil
}

// sentMessage captures a message sent via mockSender or the Telegram test server
type sentMessage struct {
	Text   string
	ChatID int64
}

// newMessageServer returns an httptest server that captures Telegram bot
// SendMessage calls and a channel to receive them
func newMessageServer(t *testing.T) (*httptest.Server, <-chan sentMessage) {
	t.Helper()

	messages := make(chan sentMessage, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}

		text := r.FormValue("text")
		chatIDStr := r.FormValue("chat_id")

		var chatID int64
		if chatIDStr != "" {
			require.NoError(t, json.Unmarshal([]byte(chatIDStr), &chatID))
		}

		messages <- sentMessage{ChatID: chatID, Text: text}

		w.Header().Set("Content-Type", "application/json")

		_, err := w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":` +
			chatIDStr + `,"type":"private"}}}`))
		require.NoError(t, err)
	}))

	return srv, messages
}

func awaitMessage(t *testing.T, messages <-chan sentMessage) sentMessage {
	t.Helper()

	select {
	case msg := <-messages:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatal("message not received")
	}

	return sentMessage{}
}
