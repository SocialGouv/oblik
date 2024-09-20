package controller

import (
	"context"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/webhook"
)

type webhookServerRunnable struct {
	KubeClients *client.KubeClients
}

func (w *webhookServerRunnable) Start(ctx context.Context) error {
	// Start your webhook server
	return webhook.Server(ctx, w.KubeClients)
}

func (w *webhookServerRunnable) NeedLeaderElection() bool {
	return false
}
