package processor

import (
	"context"
	"testing"
	"time"
)

func TestCheckContext_Active(t *testing.T) {
	ctx := context.Background()
	if err := checkContext(ctx); err != nil {
		t.Errorf("checkContext() with active context returned error: %v", err)
	}
}

func TestCheckContext_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := checkContext(ctx)
	if err == nil {
		t.Error("checkContext() with canceled context should return error")
	}
	if err != context.Canceled {
		t.Errorf("checkContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestCheckContext_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout

	err := checkContext(ctx)
	if err == nil {
		t.Error("checkContext() with timed out context should return error")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("checkContext() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestCheckContext_NotYetExpired(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := checkContext(ctx); err != nil {
		t.Errorf("checkContext() with not-yet-expired context returned error: %v", err)
	}
}
