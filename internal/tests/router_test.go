package tests

import (
	"context"
	"fmt"
	"github.com/chirino/fair-router/internal/api"
	"github.com/chirino/fair-router/internal/router"
	"github.com/chirino/fair-router/internal/worker"
	"github.com/stretchr/testify/require"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestRouter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	require := require.New(t)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	tlsConfig := api.TLSConfig{Insecure: true}

	type workerServer struct {
		name             string
		service          *worker.CapacityService
		listener         net.Listener
		address          string
		acceptedRequests int32
	}
	workerServersList := []*workerServer{}
	workerServers := map[string]*workerServer{}
	upstreamEndpoints := []string{}

	// start 10 workers..
	for i := 0; i < 10; i++ {

		listener, err := net.Listen("tcp", ":0")
		require.NoError(err)
		defer func() {
			listener.Close()
		}()
		address := DialAddress(require, listener)

		service := worker.New()
		service.Log = log

		// simulate some workers having more capacity than others.
		value := int32(10 + (i * 2))
		service.SetCapacity(value)
		service.SetRemaining(value)
		fmt.Printf("worker %s: has a capacity of %d\n", address, value)

		go func() {
			_ = worker.Serve(log, tlsConfig, listener, service)
		}()

		server := &workerServer{
			name:     fmt.Sprintf("%d", i),
			address:  address,
			service:  service,
			listener: listener,
		}
		workerServers[address] = server
		workerServersList = append(workerServersList, server)
		upstreamEndpoints = append(upstreamEndpoints, address)
	}

	rejectedRequests := int32(0)
	acceptedRequests := int32(0)

	routers := []*router.Router{}
	// start 3 routers
	for i := 0; i < 3; i++ {
		r, err := router.New(router.Options{
			Log:               log,
			Name:              fmt.Sprintf("%d", i),
			UpstreamEndpoints: upstreamEndpoints,
			TlsConfig:         tlsConfig,
		})
		require.NoError(err)
		go func() {
			err := r.Run(ctx)
			require.NoError(err)
		}()
		routers = append(routers, r)

		// Simulate clients that generating load on the router. 50 per router, so 150 in total.
		for client := 0; client < 50; client++ {
			go func(router *router.Router, client int) {
				for {

					time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

					address := router.GetNextBestUpstream()
					if address == "" {
						atomic.AddInt32(&rejectedRequests, 1)
						time.Sleep(1 * time.Second)
						continue
					}
					atomic.AddInt32(&acceptedRequests, 1)

					// Simulate a request to the worker.. this would be a http request in a real world scenario.
					server := workerServers[address]
					atomic.AddInt32(&server.acceptedRequests, 1)
					server.service.AddRemaining(-1) // reduce the remaining capacity
					workTime := (50 * time.Millisecond) + time.Duration(rand.Intn(5))*time.Second
					//fmt.Printf("worker %s: processing request for %v seconds\n", address, workTime.Seconds())
					time.Sleep(workTime)           // simulate work being done in the worker.
					server.service.AddRemaining(1) // increase the remaining capacity

				}
			}(r, client)
		}

	}

	// monitor until the test times out
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("test done\n")
			return
		default:
		}

		time.Sleep(1 * time.Second)
		fmt.Printf("--------------------------------------------------------\n")
		fmt.Printf("total rejected requests: %d\n", atomic.SwapInt32(&rejectedRequests, 0))
		fmt.Printf("total accepted requests: %d\n", atomic.SwapInt32(&acceptedRequests, 0))
		for _, server := range workerServersList {
			remaining := float32(server.service.GetRemaining())
			capacity := float32(server.service.GetCapacity())
			fmt.Printf("worker %s: idle%%: %f, accepted requests %d\n", server.name, (remaining/capacity)*100, atomic.SwapInt32(&server.acceptedRequests, 0))
		}

	}
}
func DialAddress(require *require.Assertions, listener net.Listener) string {
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(err)
	return "localhost:" + port
}
