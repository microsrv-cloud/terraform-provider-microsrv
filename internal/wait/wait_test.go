package wait_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/wait"
)

func TestUntilReady(t *testing.T) {
	t.Parallel()
	n := 0
	err := wait.UntilReady(context.Background(), wait.Config{Timeout: time.Second, Interval: time.Millisecond}, func(ctx context.Context) (string, error) {
		n++
		if n < 3 {
			return client.StatePending, nil
		}
		return client.StateReady, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUntilReadyFailed(t *testing.T) {
	t.Parallel()
	err := wait.UntilReady(context.Background(), wait.Config{Timeout: time.Second, Interval: time.Millisecond}, func(ctx context.Context) (string, error) {
		return client.StateFailed, nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUntilGone(t *testing.T) {
	t.Parallel()
	n := 0
	err := wait.UntilGone(context.Background(), wait.Config{Timeout: time.Second, Interval: time.Millisecond}, func(ctx context.Context) (string, error) {
		n++
		if n < 2 {
			return client.StateDeleting, nil
		}
		return "", &client.APIError{StatusCode: http.StatusNotFound}
	})
	if err != nil {
		t.Fatal(err)
	}
}
