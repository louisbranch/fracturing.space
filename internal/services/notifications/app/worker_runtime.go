package server

import (
	"context"
	"log"
)

// RunEmailDeliveryWorker starts the notifications email-delivery worker runtime.
//
// The worker lifecycle is intentionally separate from gRPC API serving so
// deployments can run queue processing independently from API transport.
func RunEmailDeliveryWorker(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	srvEnv := loadServerEnv()
	store, err := openNotificationsStore(srvEnv.DBPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			log.Printf("close notifications store: %v", closeErr)
		}
	}()

	worker := buildEmailDeliveryWorker(store, srvEnv)
	worker.Run(ctx)
	return nil
}
