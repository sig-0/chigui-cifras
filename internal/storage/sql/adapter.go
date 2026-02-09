package sql

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sig-0/chigui-cifras/internal/storage"
	"github.com/sig-0/chigui-cifras/internal/storage/sql/gen"
)

// Adapter wraps SQLC-generated Queries and implements storage.Store
type Adapter struct {
	q *gen.Queries
}

// NewAdapter creates a new Adapter
func NewAdapter(db gen.DBTX) *Adapter {
	return &Adapter{
		q: gen.New(db),
	}
}

func (a *Adapter) Subscribe(ctx context.Context, chatID int64, frequency string, nextSendAt time.Time) error {
	return a.q.Subscribe(ctx, gen.SubscribeParams{
		ChatID:    chatID,
		Frequency: frequency,
		NextSendAt: pgtype.Timestamptz{
			Time:  nextSendAt,
			Valid: true,
		},
	})
}

func (a *Adapter) Unsubscribe(ctx context.Context, chatID int64) error {
	return a.q.Unsubscribe(ctx, chatID)
}

func (a *Adapter) DueSubscribers(ctx context.Context) ([]*storage.Subscriber, error) {
	rows, err := a.q.DueSubscribers(ctx)
	if err != nil {
		return nil, err
	}

	subs := make([]*storage.Subscriber, len(rows))
	for i, r := range rows {
		subs[i] = &storage.Subscriber{
			ChatID:     r.ChatID,
			Frequency:  r.Frequency,
			NextSendAt: r.NextSendAt.Time,
		}
	}

	return subs, nil
}

func (a *Adapter) UpdateNextSend(ctx context.Context, chatID int64, nextSendAt time.Time) error {
	return a.q.UpdateNextSend(ctx, gen.UpdateNextSendParams{
		ChatID: chatID,
		NextSendAt: pgtype.Timestamptz{
			Time:  nextSendAt,
			Valid: true,
		},
	})
}

func (a *Adapter) CreateAlert(
	ctx context.Context,
	id string,
	chatID int64,
	base, direction string,
	threshold float64,
) (*storage.Alert, error) {
	row, err := a.q.CreateAlert(ctx, gen.CreateAlertParams{
		ID:         id,
		ChatID:     chatID,
		Base:       base,
		Direction:  direction,
		Threshold:  float64ToNumeric(threshold),
		MaxPerChat: int64(storage.MaxAlertsPerChat),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrAlertLimitReached
		}

		return nil, err
	}

	alert, err := genAlertToDomain(row)
	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func (a *Adapter) AlertsByChat(ctx context.Context, chatID int64) ([]*storage.Alert, error) {
	rows, err := a.q.AlertsByChat(ctx, chatID)
	if err != nil {
		return nil, err
	}

	return genAlertsToDomain(rows)
}

func (a *Adapter) DeleteAlert(ctx context.Context, id string, chatID int64) error {
	return a.q.DeleteAlert(ctx, gen.DeleteAlertParams{
		ID:     id,
		ChatID: chatID,
	})
}

func (a *Adapter) ActiveAlerts(ctx context.Context) ([]*storage.Alert, error) {
	rows, err := a.q.ActiveAlerts(ctx)
	if err != nil {
		return nil, err
	}

	return genAlertsToDomain(rows)
}

func (a *Adapter) TriggerAlert(ctx context.Context, id string) error {
	return a.q.TriggerAlert(ctx, id)
}

// float64ToNumeric converts a float64 to pgtype.Numeric
func float64ToNumeric(f float64) pgtype.Numeric {
	// Use big.Float for accurate conversion
	bf := new(big.Float).SetFloat64(f)

	// Convert to string with 4 decimal places to match NUMERIC(20,4)
	text := bf.Text('f', 4)

	var n pgtype.Numeric

	if err := n.Scan(text); err != nil {
		return pgtype.Numeric{}
	}

	return n
}

// numericToFloat64 converts a pgtype.Numeric to float64
func numericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, fmt.Errorf("null numeric value")
	}

	f, _ := new(big.Float).SetInt(n.Int).Float64()

	// Apply exponent: value = Int * 10^Exp
	if n.Exp != 0 {
		exp := new(big.Float).SetFloat64(1)
		base := new(big.Float).SetFloat64(10)

		e := int(n.Exp)
		if e > 0 {
			for i := 0; i < e; i++ {
				exp.Mul(exp, base)
			}
		} else {
			for i := 0; i < -e; i++ {
				exp.Mul(exp, base)
			}

			exp.Quo(new(big.Float).SetFloat64(1), exp)
		}

		f, _ = new(big.Float).Mul(new(big.Float).SetFloat64(f), exp).Float64()
	}

	return f, nil
}

func genAlertToDomain(a gen.Alert) (storage.Alert, error) {
	threshold, err := numericToFloat64(a.Threshold)
	if err != nil {
		return storage.Alert{}, fmt.Errorf("invalid threshold: %w", err)
	}

	return storage.Alert{
		ID:        a.ID,
		ChatID:    a.ChatID,
		Base:      a.Base,
		Direction: a.Direction,
		Threshold: threshold,
		CreatedAt: a.CreatedAt.Time,
	}, nil
}

func genAlertsToDomain(rows []gen.Alert) ([]*storage.Alert, error) {
	alerts := make([]*storage.Alert, 0, len(rows))

	for _, row := range rows {
		alert, err := genAlertToDomain(row)
		if err != nil {
			return nil, err
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}
