package server

import (
	"context"
	"testing"
)

func TestRunEmailDeliveryWorkerStopsOnCancel(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_NOTIFICATIONS_DB_PATH", t.TempDir()+"/notifications.db")
	t.Setenv("FRACTURING_SPACE_NOTIFICATIONS_EMAIL_DELIVERY_WORKER_POLL_INTERVAL", "5ms")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := RunEmailDeliveryWorker(ctx); err != nil {
		t.Fatalf("run email delivery worker: %v", err)
	}
}

func TestRunEmailDeliveryWorkerReturnsOpenError(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_NOTIFICATIONS_DB_PATH", t.TempDir())

	if err := RunEmailDeliveryWorker(context.Background()); err == nil {
		t.Fatal("expected open store error")
	}
}
