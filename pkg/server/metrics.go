package server

import (
	"fmt"
	"net/http"

	"k8s.io/klog/v2"
)

var (
	MetricsPort = 9090
)

func startMetricsServer(metricsHandler http.Handler) {
	klog.Infof("Starting metrics server on port %d\n", MetricsPort)
	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("/metrics", metricsHandler)

	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", MetricsPort),
		Handler: metricsRouter,
	}

	if err := metricsServer.ListenAndServe(); err != nil {
		klog.Fatal("Failed to start metrics server:", err)
	}
}
