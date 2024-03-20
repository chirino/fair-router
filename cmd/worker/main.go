package main

import (
	"context"
	"github.com/chirino/fair-router/internal/api"
	"github.com/chirino/fair-router/internal/worker"
	"github.com/urfave/cli/v3"
	"log/slog"
	"os"
	"os/signal"
)

var version = ""
var commit = ""

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	tlsConfig := api.TLSConfig{
		CACerts:  "",
		Cert:     "",
		Key:      "",
		Insecure: false,
	}
	listen := ":8080"
	app := &cli.Command{
		Name:  "worker",
		Usage: "runs a worker",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "listen",
				Usage:       "grpc address to bind",
				Value:       listen,
				Destination: &listen,
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
				Name:        "cert",
				Usage:       "public certificate",
				Value:       tlsConfig.Cert,
				Destination: &tlsConfig.Cert,
				Sources:     cli.EnvVars("WORKER_CERT"),
			},
			&cli.StringFlag{
				Name:        "key",
				Usage:       "private key",
				Value:       tlsConfig.Key,
				Destination: &tlsConfig.Key,
				Sources:     cli.EnvVars("WORKER_KEY"),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			service := worker.New()
			service.Log = log
			err := worker.ListenAndServe(log, listen, tlsConfig, service)
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
		log.Error("usage error", "err", err)
		os.Exit(1)
	}
}
