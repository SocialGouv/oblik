package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/SocialGouv/oblik/pkg/client"
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"
)

var (
	HTTPSPort = 9443
	HTTPPort  = 8081
	CertFile  = "/etc/webhook/certs/cert.pem"
	KeyFile   = "/etc/webhook/certs/key.pem"
)

const gracefulShutdownTimeout = 25 * time.Second

func init() {
	_ = admissionv1.AddToScheme(scheme)
	_ = cnpgv1.AddToScheme(scheme)
	operatorUsername = fmt.Sprintf("system:serviceaccount:%s:%s", operatorNamespace, operatorServiceAccount)
}

func createMux(kubeClients *client.KubeClients) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", HealthCheckHandler)
	mux.HandleFunc("/readyz", func(writer http.ResponseWriter, request *http.Request) {
		ReadyCheckHandler(context.Background(), writer, request)
	})
	mux.HandleFunc("/mutate", func(writer http.ResponseWriter, request *http.Request) {
		MutateHandler(writer, request, kubeClients)
	})

	return mux
}

func startHTTPServer(ctx context.Context, mux *http.ServeMux) *http.Server {
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", HTTPPort),
		Handler: mux,
	}

	go func() {
		klog.Infof("Starting HTTP server on port %d", HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Errorf("HTTP server error: %v", err)
		}
	}()

	return httpServer
}

func startHTTPSServer(ctx context.Context, mux *http.ServeMux) *http.Server {
	httpsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", HTTPSPort),
		Handler: mux,
	}

	go func() {
		klog.Infof("Starting HTTPS server on port %d", HTTPSPort)
		if err := httpsServer.ListenAndServeTLS(CertFile, KeyFile); err != nil && err != http.ErrServerClosed {
			klog.Errorf("HTTPS server error: %v", err)
		}
	}()

	return httpsServer
}

func Server(ctx context.Context, kubeClients *client.KubeClients) error {
	mux := createMux(kubeClients)

	httpServer := startHTTPServer(ctx, mux)
	httpsServer := startHTTPSServer(ctx, mux)

	metricsHandler := promhttp.Handler()
	go startMetricsServer(metricsHandler)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		klog.Errorf("HTTP server forced to shutdown: %v", err)
	}

	if err := httpsServer.Shutdown(shutdownCtx); err != nil {
		klog.Errorf("HTTPS server forced to shutdown: %v", err)
	}

	return nil
}
