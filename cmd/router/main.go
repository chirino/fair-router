package main

import (
	"context"
	"github.com/chirino/fair-router/internal/api"
	"github.com/chirino/fair-router/internal/router"
	"github.com/urfave/cli/v3"
	"log/slog"
	"os"
	"os/signal"
)

var version = ""
var commit = ""

func main() {
	tlsConfig := api.TLSConfig{
		CACerts:  "",
		Cert:     "",
		Key:      "",
		Insecure: false,
	}
	workerAddress := "localhost:8080"
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	app := &cli.Command{
		Name:  "router",
		Usage: "runs a router",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "worker-address",
				Usage:       "grpc address of the worker",
				Value:       workerAddress,
				Destination: &workerAddress,
				Sources:     cli.EnvVars("WORKER_LISTEN"),
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Usage:       "private key file",
				Value:       tlsConfig.Insecure,
				Destination: &tlsConfig.Insecure,
				Sources:     cli.EnvVars("WORKER_INSECURE"),
			},
			&cli.StringFlag{
				Name:        "ca",
				Usage:       "certificate authority certs",
				Value:       tlsConfig.CACerts,
				Destination: &tlsConfig.CACerts,
				Sources:     cli.EnvVars("WORKER_CA"),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			router, err := router.New(router.Options{
				UpstreamEndpoints: []string{workerAddress},
				Log:               log,
				TlsConfig:         tlsConfig,
			})
			if err != nil {
				log.Error("server error", "err", err)
				os.Exit(2)
			}
			err = router.Run(ctx)
			if err != nil {
				log.Error("server error", "err", err)
				os.Exit(2)
			}
			return nil
		},
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()
	if err := app.Run(ctx, os.Args); err != nil {
		log.Error("error", "err", err)
		os.Exit(1)
	}
}
