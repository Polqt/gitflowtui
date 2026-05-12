package realtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/Polqt/gitflowtui/tui"
	"github.com/gorilla/websocket"
)

// Client subscribes to gitflow-tui realtime events with automatic reconnects.
type Client struct {
	URL        string
	MinBackoff time.Duration
	MaxBackoff time.Duration
	Dialer     *websocket.Dialer
}

// Listen connects to the websocket endpoint and emits decoded events until ctx is cancelled.
func (c Client) Listen(ctx context.Context) (<-chan tui.RealtimeEvent, <-chan error) {
	events := make(chan tui.RealtimeEvent, 32)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		minBackoff := c.MinBackoff
		if minBackoff <= 0 {
			minBackoff = 250 * time.Millisecond
		}
		maxBackoff := c.MaxBackoff
		if maxBackoff <= 0 {
			maxBackoff = 10 * time.Second
		}
		dialer := c.Dialer
		if dialer == nil {
			dialer = websocket.DefaultDialer
		}

		attempt := 0
		for ctx.Err() == nil {
			conn, resp, err := dialer.DialContext(ctx, c.URL, nil)
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			if err != nil {
				sendErr(ctx, errs, fmt.Errorf("connect realtime websocket: %w", err))
				if !sleepBackoff(ctx, minBackoff, maxBackoff, attempt) {
					return
				}
				attempt++
				continue
			}

			attempt = 0
			err = readEvents(ctx, conn, events)
			_ = conn.Close()
			if ctx.Err() != nil {
				return
			}
			sendErr(ctx, errs, fmt.Errorf("realtime websocket disconnected: %w", err))
			if !sleepBackoff(ctx, minBackoff, maxBackoff, attempt) {
				return
			}
			attempt++
		}
	}()

	return events, errs
}

func readEvents(ctx context.Context, conn *websocket.Conn, events chan<- tui.RealtimeEvent) error {
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var event tui.RealtimeEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode realtime event: %w", err)
		}
		select {
		case events <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func sleepBackoff(ctx context.Context, minBackoff, maxBackoff time.Duration, attempt int) bool {
	multiplier := math.Pow(2, float64(attempt))
	delay := time.Duration(float64(minBackoff) * multiplier)
	if delay > maxBackoff {
		delay = maxBackoff
	}
	jitter := randomJitter(delay / 4)
	timer := time.NewTimer(delay + jitter)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func randomJitter(maxDelay time.Duration) time.Duration {
	if maxDelay <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxDelay)))
	if err != nil {
		return 0
	}
	return time.Duration(n.Int64())
}

func sendErr(ctx context.Context, errs chan<- error, err error) {
	select {
	case errs <- err:
	case <-ctx.Done():
	default:
	}
}
