package handler

import (
	"context"
	"net"

	models "github.com/tigranqic/metrics-tpl/internal/model"
	"github.com/tigranqic/metrics-tpl/internal/proto"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MetricsServer implements the gRPC Metrics service.
type MetricsServer struct {
	proto.UnimplementedMetricsServer
	store repository.Storage
	log   *zap.Logger
}

// NewMetricsServer creates a new MetricsServer instance.
func NewMetricsServer(store repository.Storage, log *zap.Logger) *MetricsServer {
	return &MetricsServer{
		store: store,
		log:   log,
	}
}

// UpdateMetrics receives a batch of metrics via gRPC and saves them to storage.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	metrics := make([]models.Metrics, 0, len(req.Metrics))

	for _, m := range req.Metrics {
		item := models.Metrics{
			ID: m.Id,
		}

		switch m.Type {
		case proto.Metric_GAUGE:
			item.MType = "gauge"
			val := m.Value
			item.Value = &val
		case proto.Metric_COUNTER:
			item.MType = "counter"
			val := m.Delta
			item.Delta = &val
		}
		metrics = append(metrics, item)
	}

	if err := s.store.UpdateBatch(metrics); err != nil {
		s.log.Error("failed to update metrics batch via gRPC", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update metrics batch: %v", err)
	}

	return &proto.UpdateMetricsResponse{}, nil
}

// SubnetInterceptor checks if the request's X-Real-IP metadata belongs to the trusted subnet.
func SubnetInterceptor(trustedSubnet string, log *zap.Logger) grpc.UnaryServerInterceptor {
	var subnet *net.IPNet
	if trustedSubnet != "" {
		_, ipNet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			log.Error("failed to parse trusted subnet for gRPC", zap.String("subnet", trustedSubnet), zap.Error(err))
		} else {
			subnet = ipNet
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if subnet == nil {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is missing")
		}

		ips := md.Get("x-real-ip")
		if len(ips) == 0 {
			log.Debug("gRPC access denied: x-real-ip header is missing")
			return nil, status.Error(codes.PermissionDenied, "x-real-ip is missing")
		}

		ip := net.ParseIP(ips[0])
		if ip == nil || !subnet.Contains(ip) {
			log.Debug("gRPC access denied: IP not in trusted subnet", zap.String("ip", ips[0]))
			return nil, status.Error(codes.PermissionDenied, "ip is not trusted")
		}

		return handler(ctx, req)
	}
}
