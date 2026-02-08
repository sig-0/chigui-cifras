package bot

import (
	"context"
	"log/slog"
	"time"

	"github.com/sig-0/fxrates/provider/currencies"
	"github.com/sig-0/fxrates/provider/ves"
	"golang.org/x/sync/errgroup"

	"github.com/sig-0/chigui-cifras/internal/fxrates"
	"github.com/sig-0/chigui-cifras/internal/storage"
)

// MessageSender sends HTML messages to Telegram chats
type MessageSender interface {
	SendHTMLMessage(ctx context.Context, chatID int64, text string) error
}

// Scheduler runs periodic broadcast and alert checks
type Scheduler struct {
	store             storage.Store
	fxClient          *fxrates.Client
	sender            MessageSender
	logger            *slog.Logger
	broadcastInterval time.Duration
	alertInterval     time.Duration
}

// NewScheduler creates a new Scheduler
func NewScheduler(
	store storage.Store,
	fxClient *fxrates.Client,
	sender MessageSender,
	broadcastInterval time.Duration,
	alertInterval time.Duration,
	logger *slog.Logger,
) *Scheduler {
	return &Scheduler{
		store:             store,
		fxClient:          fxClient,
		sender:            sender,
		broadcastInterval: broadcastInterval,
		alertInterval:     alertInterval,
		logger:            logger,
	}
}

// Run starts the scheduler loops, blocking until ctx is canceled
func (s *Scheduler) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		s.runBroadcastLoop(gCtx)

		return nil
	})

	g.Go(func() error {
		s.runAlertLoop(gCtx)

		return nil
	})

	return g.Wait()
}

func (s *Scheduler) runBroadcastLoop(ctx context.Context) {
	ticker := time.NewTicker(s.broadcastInterval)
	defer ticker.Stop()

	s.logger.Info("broadcast loop started", "poll_interval", s.broadcastInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.broadcastDue(ctx)
		}
	}
}

func (s *Scheduler) broadcastDue(ctx context.Context) {
	subscribers, err := s.store.DueSubscribers(ctx)
	if err != nil {
		s.logger.Error("failed to fetch due subscribers", "error", err)

		return
	}

	if len(subscribers) == 0 {
		return
	}

	message, err := s.fetchDashboardMessage(ctx)
	if err != nil {
		s.logger.Error("failed to fetch dashboard for broadcast", "error", err)

		return
	}

	for _, sub := range subscribers {
		if err := s.sender.SendHTMLMessage(ctx, sub.ChatID, message); err != nil {
			s.logger.Warn("broadcast send failed, unsubscribing",
				"chat_id", sub.ChatID, "error", err)

			if unsubErr := s.store.Unsubscribe(ctx, sub.ChatID); unsubErr != nil {
				s.logger.Error("failed to unsubscribe after send failure",
					"chat_id", sub.ChatID, "error", unsubErr)
			}

			continue
		}

		nextSendAt := time.Now().Add(intervalForFrequency(sub.Frequency))

		if err := s.store.UpdateNextSend(ctx, sub.ChatID, nextSendAt); err != nil {
			s.logger.Error("failed to update next_send_at",
				"chat_id", sub.ChatID, "error", err)
		}
	}

	s.logger.Info("broadcast cycle complete", "recipients", len(subscribers))
}

// intervalForFrequency returns the interval duration for a given frequency
func intervalForFrequency(frequency string) time.Duration {
	switch frequency {
	case storage.FrequencyHourly:
		return time.Hour
	case storage.FrequencyDaily:
		return 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func (s *Scheduler) fetchDashboardMessage(ctx context.Context) (string, error) {
	var (
		usdRate  *fxrates.ExchangeRate
		eurRate  *fxrates.ExchangeRate
		usdtPage *fxrates.PageExchangeRate
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		rates, err := s.fxClient.Rate(gCtx, currencies.USD.String(), currencies.VES.String(), ves.BCVSource.String())
		if err != nil {
			return err
		}

		usdRate = SelectPreferredRate(rates.Results)

		return nil
	})

	g.Go(func() error {
		rates, err := s.fxClient.Rate(gCtx, currencies.EUR.String(), currencies.VES.String(), ves.BCVSource.String())
		if err != nil {
			return err
		}

		eurRate = SelectPreferredRate(rates.Results)

		return nil
	})

	g.Go(func() error {
		rates, err := s.fxClient.Rate(gCtx, currencies.USDT.String(), currencies.VES.String(), "")
		if err != nil {
			return err
		}

		usdtPage = rates

		return nil
	})

	if err := g.Wait(); err != nil {
		return "", err
	}

	var usdtRates []fxrates.ExchangeRate
	if usdtPage != nil {
		usdtRates = usdtPage.Results
	}

	return FormatDashboard(usdRate, eurRate, usdtRates), nil
}

func (s *Scheduler) runAlertLoop(ctx context.Context) {
	ticker := time.NewTicker(s.alertInterval)
	defer ticker.Stop()

	s.logger.Info("alert loop started", "interval", s.alertInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAlerts(ctx)
		}
	}
}

func (s *Scheduler) checkAlerts(ctx context.Context) {
	alerts, err := s.store.ActiveAlerts(ctx)
	if err != nil {
		s.logger.Error("failed to fetch active alerts", "error", err)

		return
	}

	if len(alerts) == 0 {
		return
	}

	// Group alerts by base currency
	grouped := make(map[string][]*storage.Alert)
	for _, a := range alerts {
		grouped[a.Base] = append(grouped[a.Base], a)
	}

	for base, alertGroup := range grouped {
		source := SourceForCurrency(fxrates.Currency(base))

		rates, err := s.fxClient.Rate(ctx, base, currencies.VES.String(), source.String())
		if err != nil {
			s.logger.Error("failed to fetch rate for alerts", "base", base, "error", err)

			continue
		}

		rate := SelectPreferredRate(rates.Results)
		if rate == nil {
			continue
		}

		currentRate := rate.Rate

		for _, alert := range alertGroup {
			triggered := false

			if alert.Direction == storage.DirectionAbove && currentRate >= alert.Threshold {
				triggered = true
			} else if alert.Direction == storage.DirectionBelow && currentRate <= alert.Threshold {
				triggered = true
			}

			if !triggered {
				continue
			}

			msg := FormatAlertTriggered(*alert, currentRate)

			if err := s.sender.SendHTMLMessage(ctx, alert.ChatID, msg); err != nil {
				s.logger.Warn("alert send failed, triggering to prevent retry",
					"alert_id", alert.ID, "chat_id", alert.ChatID, "error", err)
			}

			if err := s.store.TriggerAlert(ctx, alert.ID); err != nil {
				s.logger.Error("failed to trigger alert", "alert_id", alert.ID, "error", err)
			}
		}
	}
}
