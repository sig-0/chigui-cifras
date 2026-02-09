package storage

import (
	"context"
	"errors"
	"time"
)

// Frequency constants for subscription broadcasts
const (
	FrequencyHourly = "hourly"
	FrequencyDaily  = "daily"
)

// Direction constants for price alerts
const (
	DirectionAbove = "above"
	DirectionBelow = "below"
)

// MaxAlertsPerChat is the maximum number of active alerts a user can have
const MaxAlertsPerChat = 5

// ErrAlertLimitReached is returned when a user has hit the alert limit
var ErrAlertLimitReached = errors.New("alert limit reached")

// Store defines the interface for subscription and alert persistence
type Store interface {
	Subscribe(ctx context.Context, chatID int64, frequency string, nextSendAt time.Time) error
	Unsubscribe(ctx context.Context, chatID int64) error
	DueSubscribers(ctx context.Context) ([]*Subscriber, error)
	UpdateNextSend(ctx context.Context, chatID int64, nextSendAt time.Time) error

	CreateAlert(ctx context.Context, id string, chatID int64, base, direction string, threshold float64) (*Alert, error)
	AlertsByChat(ctx context.Context, chatID int64) ([]*Alert, error)
	DeleteAlert(ctx context.Context, id string, chatID int64) error
	ActiveAlerts(ctx context.Context) ([]*Alert, error)
	TriggerAlert(ctx context.Context, id string) error
}

// Subscriber represents a user subscribed to broadcasts
type Subscriber struct {
	NextSendAt time.Time
	Frequency  string // FrequencyHourly or FrequencyDaily
	ChatID     int64
}

// Alert represents a price threshold alert
type Alert struct {
	CreatedAt time.Time
	ID        string
	Base      string
	Direction string // DirectionAbove or DirectionBelow
	ChatID    int64
	Threshold float64
}
