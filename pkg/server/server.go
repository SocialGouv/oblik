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
	Port = 9443
)

const gracefulShutdownTimeout = 25 * time.Second

func init() {
	_ = admissionv1.AddToScheme(scheme)
	_ = cnpgv1.AddToScheme(scheme)
	operatorUsername = fmt.Sprintf("system:serviceaccount:%s:%s", operatorNamespace, operatorServiceAccount)
}

func Server(ctx context.Context, kubeClients *client.KubeClients) error {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", Port),
	}

	mux := http.NewServeMux()
	server.Handler = mux

	mux.HandleFunc("/healthz", HealthCheckHandler)
	mux.HandleFunc("/readyz", func(writer http.ResponseWriter, request *http.Request) {
		ReadyCheckHandler(ctx, writer, request)
	})
	mux.HandleFunc("/mutate", func(writer http.ResponseWriter, request *http.Request) {
		MutateHandler(writer, request, kubeClients)
	})

	metricsHandler := promhttp.Handler()
	go startMetricsServer(metricsHandler)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			klog.Errorf("Server forced to shutdown: %v", err)
		}

	}()

	return server.ListenAndServeTLS(CertFile, KeyFile)
}
