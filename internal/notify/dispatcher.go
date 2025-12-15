package notify

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/notifyd-eng/notifyd/internal/config"
	"github.com/notifyd-eng/notifyd/internal/store"
	"github.com/rs/zerolog/log"
)

type Sender interface {
	Send(ctx context.Context, n *store.Notification) error
	Channel() string
}

type Dispatcher struct {
	store   *store.Store
	cfg     config.RetryConfig
	senders map[string]Sender
	mu      sync.RWMutex
	quit    chan struct{}
}

func NewDispatcher(s *store.Store, cfg config.RetryConfig) *Dispatcher {
	return &Dispatcher{
		store:   s,
		cfg:     cfg,
		senders: make(map[string]Sender),
		quit:    make(chan struct{}),
	}
}

func (d *Dispatcher) Register(s Sender) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.senders[s.Channel()] = s
	log.Info().Str("channel", s.Channel()).Msg("registered sender")
}

func (d *Dispatcher) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Dur("interval", interval).Msg("dispatcher started")

	for {
		select {
		case <-ticker.C:
			d.processBatch()
		case <-d.quit:
			log.Info().Msg("dispatcher stopped")
			return
		}
	}
}

func (d *Dispatcher) Stop() {
	close(d.quit)
}

func (d *Dispatcher) processBatch() {
	pending, err := d.store.PendingBatch(50)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch pending notifications")
		return
	}

	if len(pending) == 0 {
		return
	}

	log.Debug().Int("count", len(pending)).Msg("processing batch")

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, n := range pending {
		wg.Add(1)
		sem <- struct{}{}

		go func(n *store.Notification) {
			defer wg.Done()
			defer func() { <-sem }()
			d.deliver(n)
		}(n)
	}

	wg.Wait()
}

func (d *Dispatcher) deliver(n *store.Notification) {
	d.mu.RLock()
	sender, ok := d.senders[n.Channel]
	d.mu.RUnlock()

	if !ok {
		log.Warn().Str("channel", n.Channel).Str("id", n.ID).Msg("no sender for channel")
		d.store.MarkFailed(n.ID, fmt.Sprintf("no sender registered for channel %q", n.Channel))
		return
	}

	var lastErr error
	for attempt := 0; attempt < d.cfg.MaxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		err := sender.Send(ctx, n)
		cancel()

		if err == nil {
			if err := d.store.MarkSent(n.ID); err != nil {
				log.Error().Err(err).Str("id", n.ID).Msg("failed to mark as sent")
			}
			log.Info().Str("id", n.ID).Str("channel", n.Channel).Int("attempt", attempt+1).Msg("delivered")
			return
		}

		lastErr = err
		backoff := d.backoff(attempt)
		log.Warn().Err(err).Str("id", n.ID).Int("attempt", attempt+1).Dur("backoff", backoff).Msg("delivery failed, retrying")

		select {
		case <-time.After(backoff):
		case <-d.quit:
			return
		}
	}

	d.store.MarkFailed(n.ID, lastErr.Error())
	log.Error().Str("id", n.ID).Str("channel", n.Channel).Msg("delivery permanently failed")
}

func (d *Dispatcher) backoff(attempt int) time.Duration {
	wait := float64(d.cfg.InitialWait) * math.Pow(d.cfg.Multiplier, float64(attempt))
	if wait > float64(d.cfg.MaxWait) {
		wait = float64(d.cfg.MaxWait)
	}
	return time.Duration(wait)
}
