package controller

import (
	"context"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/watcher"
)

type watcherRunnable struct {
	KubeClients *client.KubeClients
}

func (w *watcherRunnable) Start(ctx context.Context) error {
	watcher.CronScheduler.Start()

	go func() {
		watcher.WatchVPAs(ctx, w.KubeClients.Clientset, w.KubeClients.DynamicClient, w.KubeClients.VpaClientset)
	}()

	<-ctx.Done()
	return nil
}

func (w *watcherRunnable) NeedLeaderElection() bool {
	return true
}
