package router

import (
	"context"
	"github.com/chirino/fair-router/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
	"sync/atomic"
)

type Options struct {
	Log               *slog.Logger
	Name              string
	UpstreamEndpoints []string
	TlsConfig         api.TLSConfig
}
type stats struct {
	remaining int32
	capacity  int32
}
type Router struct {
	log         *slog.Logger
	Name        string
	endpoints   []string
	stats       []stats
	signal      chan struct{}
	dialOptions []grpc.DialOption
}

func New(opts Options) (*Router, error) {

	dialOptions, err := api.NewDialOptions(opts.TlsConfig)
	if err != nil {
		return nil, err
	}

	return &Router{
		log:         opts.Log,
		Name:        opts.Name,
		endpoints:   opts.UpstreamEndpoints,
		dialOptions: dialOptions,
		signal:      make(chan struct{}, len(opts.UpstreamEndpoints)),
		stats:       make([]stats, len(opts.UpstreamEndpoints)),
	}, nil
}

func (r *Router) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}
	r.stats = make([]stats, len(r.endpoints))
	for capacityIndex, address := range r.endpoints {
		err := r.StartUpstreamMonitor(ctx, capacityIndex, address, wg)
		if err != nil {
			return err
		}
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

func (r *Router) StartUpstreamMonitor(ctx context.Context, capacityIndex int, address string, wg *sync.WaitGroup) error {
	log := r.log

	name := r.Name
	var err error
	if name == "" {
		name, err = os.Hostname()
		if err != nil {
			name = "unknown"
		}
	}

	// dial each service separately to allow the server to load balance them..
	log.Info("connecting to GRPC server", "address", r.endpoints, "router", name)
	c, err := grpc.Dial(address, r.dialOptions...)
	if err != nil {
		return err
	}

	client := api.NewMetricsServiceClient(c)

	connection, err := client.Monitor(ctx, &api.MetricsRequest{
		Router: name,
	})
	if err != nil {
		if status.Code(err) != codes.Canceled {
			log.Error("connection.Recv failed", "err", err)
		}
		return err
	}
	defer connection.CloseSend()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			monitorResponse, err := connection.Recv()
			if err != nil {
				if status.Code(err) != codes.Canceled {
					log.Error("connection.Recv failed", "err", err)
				}
				return
			}
			if monitorResponse.Capacity != nil {
				atomic.StoreInt32(&r.stats[capacityIndex].capacity, monitorResponse.GetCapacity())
			}
			if monitorResponse.Remaining != nil {
				atomic.StoreInt32(&r.stats[capacityIndex].remaining, monitorResponse.GetRemaining())
			}
		}
	}()
	return nil
}

func (r *Router) GetNextBestUpstream() string {
	type upstream struct {
		address string
		bucket  uint8
		index   int
	}
	upstreams := make([]upstream, len(r.stats))
	for i := range r.stats {
		upstreams[i].index = i
		upstreams[i].address = r.endpoints[i]
		capacity := atomic.LoadInt32(&r.stats[i].capacity)
		if capacity <= 0 {
			capacity = math.MaxInt32
		}
		remaining := atomic.LoadInt32(&r.stats[i].remaining)
		if remaining <= 0 {
			remaining = 0
		}

		idle := float32(remaining) / float32(capacity) // between 0 and 1.0
		idle = idle * 10                               // scale to buckets of 0 - 10
		upstreams[i].bucket = uint8(idle)
	}

	sort.Slice(upstreams, func(i, j int) bool {
		return upstreams[i].bucket > upstreams[j].bucket
	})
	if upstreams[0].bucket == 0 {
		// no capacity available
		return ""
	}

	// we used buckets to group the upstreams.  how many upstreams are in the best bucket?
	chooseFrom := 0
	for _, u := range upstreams {
		if u.bucket == upstreams[0].bucket {
			chooseFrom += 1
		} else {
			break
		}
	}

	// to avoid the thundering herd problem, we randomly pick one of the best upstreams.
	picked := upstreams[rand.Intn(chooseFrom)]
	atomic.AddInt32(&r.stats[picked.index].remaining, -1)

	return picked.address

}
