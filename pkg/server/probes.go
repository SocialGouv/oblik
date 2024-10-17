package server

import (
	"context"
	"net/http"
)

func HealthCheckHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
}
func ReadyCheckHandler(ctx context.Context, writer http.ResponseWriter, _ *http.Request) {
	// if ctx is done it means the server is shutting down
	if ctx.Err() != nil {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	writer.WriteHeader(http.StatusOK)
}
