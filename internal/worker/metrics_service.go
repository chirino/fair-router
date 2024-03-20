package worker

import (
	"github.com/chirino/fair-router/internal/api"
	"github.com/chirino/fair-router/internal/signalbus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"sync/atomic"
)

type CapacityService struct {
	Log       *slog.Logger
	remaining int32
	capacity  int32
	signal    signalbus.SignalBus
}

func New() *CapacityService {
	return &CapacityService{
		signal: signalbus.NewSignalBus(),
	}
}

var _ api.MetricsServiceServer = &CapacityService{}

func (s *CapacityService) SetCapacity(value int32) {
	atomic.StoreInt32(&s.capacity, value)
	s.signal.NotifyAll()
}

func (s *CapacityService) GetCapacity() int32 {
	return atomic.LoadInt32(&s.capacity)
}

func (s *CapacityService) SetRemaining(value int32) {
	atomic.StoreInt32(&s.remaining, value)
	s.signal.NotifyAll()
}

func (s *CapacityService) GetRemaining() int32 {
	return atomic.LoadInt32(&s.remaining)
}

func (s *CapacityService) AddRemaining(value int32) {
	atomic.AddInt32(&s.remaining, value)
	s.signal.NotifyAll()
}

func (s *CapacityService) Monitor(msg *api.MetricsRequest, connection api.MetricsService_MonitorServer) error {
	s.Log.Info("accepted router monitor connection", "router", msg.GetRouter())
	sub := s.signal.Subscribe("metrics")
	defer sub.Close()

	lastCapacity := atomic.LoadInt32(&s.capacity)
	lastRemaining := atomic.LoadInt32(&s.remaining)
	err := connection.Send(&api.MetricsResponse{
		Capacity:  &lastCapacity,
		Remaining: &lastRemaining,
	})
	if err != nil {
		if status.Code(err) != codes.Canceled {
			s.Log.Error("connection.Send failed", "err", err)
		}
		return nil
	}

	response := api.MetricsResponse{
		Capacity:  &s.capacity,
		Remaining: &s.remaining,
	}
	for {
		select {
		case <-connection.Context().Done():
			return nil
		case <-sub.Signal():

			capacity := atomic.LoadInt32(&s.capacity)
			remaining := atomic.LoadInt32(&s.remaining)
			if capacity != lastCapacity {
				response.Capacity = &capacity
			} else {
				response.Capacity = nil
			}
			lastCapacity = capacity
			if remaining != lastRemaining {
				response.Remaining = &remaining
			} else {
				response.Remaining = nil
			}
			lastRemaining = remaining

			if response.Capacity != nil || response.Remaining != nil {
				err := connection.Send(&response)
				if err != nil {
					if status.Code(err) != codes.Canceled {
						s.Log.Error("connection.Send failed", "err", err)
					}
					return nil
				}
			}
		}
	}
	return nil
}
