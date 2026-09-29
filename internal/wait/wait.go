package wait

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
)

// Config controls polling behavior.
type Config struct {
	Timeout  time.Duration
	Interval time.Duration
}

func (c Config) withDefaults() Config {
	if c.Timeout <= 0 {
		c.Timeout = 15 * time.Minute
	}
	if c.Interval <= 0 {
		c.Interval = 2 * time.Second
	}
	return c
}

// StateFunc returns the current lifecycle state, or an APIError.
type StateFunc func(ctx context.Context) (state string, err error)

// UntilReady polls until state is READY/CHECKPOINTED or FAILED.
// CHECKPOINTED is platform scale-to-zero (serverless dormant / checkpointed) and
// is operationally ready (handled by IsReadyState).
func UntilReady(ctx context.Context, cfg Config, get StateFunc) error {
	cfg = cfg.withDefaults()
	deadline := time.Now().Add(cfg.Timeout)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		state, err := get(ctx)
		if err != nil {
			return err
		}
		switch state {
		case client.StateReady, client.StateCheckpointed:
			return nil
		case client.StateFailed:
			return fmt.Errorf("resource entered FAILED state")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for READY (last state=%s)", state)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// UntilGone polls until GET returns 404.
func UntilGone(ctx context.Context, cfg Config, get StateFunc) error {
	cfg = cfg.withDefaults()
	deadline := time.Now().Add(cfg.Timeout)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		_, err := get(ctx)
		if err != nil {
			var apiErr *client.APIError
			if errors.As(err, &apiErr) && apiErr.IsNotFound() {
				return nil
			}
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for resource deletion")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
