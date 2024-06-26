package controller

import (
	"context"

	"github.com/SocialGouv/oblik/pkg/watcher"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func Run() {
	klog.InitFlags(nil)

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating Kubernetes client: %s", err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Errorf("Error creating dynamic client: %s", err.Error())
		return
	}

	vpaClientset, err := vpaclientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating VPA client: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.CronScheduler.Start()

	go watcher.WatchVPAs(ctx, clientset, dynamicClient, vpaClientset)

	klog.Info("Starting VPA Operator...")
	<-ctx.Done()
	klog.Info("Shutting down VPA Operator...")
}
