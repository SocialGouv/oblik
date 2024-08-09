package controller

import (
	"context"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/watcher"
	"github.com/SocialGouv/oblik/pkg/webhook"
	"k8s.io/klog/v2"
)

func Run() {
	kubeClients := client.NewKubeClients()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.CronScheduler.Start()

	go watcher.WatchVPAs(ctx, kubeClients.Clientset, kubeClients.DynamicClient, kubeClients.VpaClientset)
	go webhook.Server(ctx, kubeClients)

	klog.Info("Starting VPA Operator...")
	<-ctx.Done()
	klog.Info("Shutting down VPA Operator...")
}
