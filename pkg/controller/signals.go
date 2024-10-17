package controller

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"
)

func handleSignals(ctx context.Context, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	klog.Info("Received termination signal, shutting down gracefully...")
	cancel()
	os.Exit(0)
}
