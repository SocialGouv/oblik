package controller

import (
	"context"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/server"
)

type serverRunnable struct {
	KubeClients *client.KubeClients
}

func (s *serverRunnable) Start(ctx context.Context) error {
	return server.Server(ctx, s.KubeClients)
}

func (s *serverRunnable) NeedLeaderElection() bool {
	return false
}
