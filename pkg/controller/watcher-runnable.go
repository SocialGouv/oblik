package controller

import (
	"context"

	"github.com/SocialGouv/oblik/pkg/cleaner"
	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/watcher"
)

type watcherRunnable struct {
	KubeClients *client.KubeClients
}

func (w *watcherRunnable) Start(ctx context.Context) error {
	watcher.CronScheduler.Start()

	go func() {
		cleaner.CleanUpVPAs(ctx, w.KubeClients)
	}()

	go func() {
		watcher.WatchWorkloads(ctx, w.KubeClients)
	}()

	go func() {
		watcher.WatchVPAs(ctx, w.KubeClients)
	}()

	go func() {
		watcher.WatchResourcesConfigs(ctx, w.KubeClients)
	}()

	<-ctx.Done()
	return nil
}

func (w *watcherRunnable) NeedLeaderElection() bool {
	return true
}
