package api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type TLSConfig struct {
	Insecure bool
	CACerts  string
	Cert     string
	Key      string
}

func NewDialOptions(config TLSConfig) ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{}
	if config.Insecure {
		opts = append(opts, grpc.WithInsecure())
	} else {
		tlsConfig, err := newTLSConfig(config)
		if err != nil {
			return opts, err
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}
	return opts, nil
}

func NewServerOptions(config TLSConfig) ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	if !config.Insecure {
		tlsConfig, err := newTLSConfig(config)
		if err != nil {
			return opts, err
		}
		opts = []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsConfig))}
	}

	var streamInterceptors []grpc.StreamServerInterceptor
	var unaryInterceptors []grpc.UnaryServerInterceptor
	streamInterceptors = append(streamInterceptors, grpc_recovery.StreamServerInterceptor())
	unaryInterceptors = append(unaryInterceptors, grpc_recovery.UnaryServerInterceptor())
	opts = append(opts, grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)))
	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)))

	return opts, nil
}

func newTLSConfig(config TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{},
	}
	if config.Cert != "" && config.Key != "" {
		certificate, err := tls.X509KeyPair(
			[]byte(config.Cert),
			[]byte(config.Key),
		)
		if err != nil {
			return nil, fmt.Errorf("could not load certificate key pair (%s, %s): %v", config.Cert, config.Key, err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}

	if config.CACerts != "" {
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM([]byte(config.CACerts))
		if !ok {
			return nil, fmt.Errorf("failed to append ca certs")
		}
		tlsConfig.RootCAs = certPool
		tlsConfig.ClientCAs = certPool
	}
	return tlsConfig, nil
}
