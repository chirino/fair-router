package worker

import (
	"fmt"
	"github.com/chirino/fair-router/internal/api"
	"google.golang.org/grpc"
	"log/slog"
	"net"
)

func ListenAndServe(log *slog.Logger, bindAddress string, tlsConfig api.TLSConfig, service *CapacityService) error {
	listener, err := net.Listen("tcp", bindAddress)
	if err != nil {
		return err
	}
	defer listener.Close()
	return Serve(log, tlsConfig, listener, service)
}

func Serve(log *slog.Logger, tlsConfig api.TLSConfig, listener net.Listener, service *CapacityService) error {
	opts, err := api.NewServerOptions(tlsConfig)
	if err != nil {
		return fmt.Errorf("invalid GRPC server configuration: %w", err)
	}

	server := grpc.NewServer(opts...)
	defer server.Stop()

	api.RegisterMetricsServiceServer(server, service)

	log.Info("listening for GRPC connections", "addr", listener.Addr())
	return server.Serve(listener)
}
