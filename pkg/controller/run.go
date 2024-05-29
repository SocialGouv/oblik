package controller

import (
	"context"
	"sync"

	cron "github.com/robfig/cron/v3"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

var (
	cronScheduler = cron.New()
	cronJobs      = make(map[string]cron.EntryID)
	cronMutex     sync.Mutex
)

func Run() {
	klog.InitFlags(nil)

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating Kubernetes client: %s", err.Error())
	}

	vpaClientset, err := vpaclientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating VPA client: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cronScheduler.Start()

	go watchVPAs(ctx, clientset, vpaClientset)

	klog.Info("Starting VPA Operator...")
	<-ctx.Done()
	klog.Info("Shutting down VPA Operator...")
}