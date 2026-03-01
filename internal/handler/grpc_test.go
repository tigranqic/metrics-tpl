package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tigranqic/metrics-tpl/internal/proto"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestMetricsServer_UpdateMetrics(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	server := NewMetricsServer(store, logger)

	// Test data
	gaugeVal := 123.45
	counterDelta := int64(10)
	req := &proto.UpdateMetricsRequest{
		Metrics: []*proto.Metric{
			{Id: "GaugeMetric", Type: proto.Metric_GAUGE, Value: gaugeVal},
			{Id: "CounterMetric", Type: proto.Metric_COUNTER, Delta: counterDelta},
		},
	}

	resp, err := server.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify storage
	val, err := store.GetGauge("GaugeMetric")
	assert.NoError(t, err)
	assert.Equal(t, gaugeVal, val)

	count, err := store.GetCounter("CounterMetric")
	assert.NoError(t, err)
	assert.Equal(t, counterDelta, count)
}

func TestSubnetInterceptor(t *testing.T) {
	logger := zap.NewNop()
	trustedSubnet := "192.168.1.0/24"

	// Mock handler
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	interceptor := SubnetInterceptor(trustedSubnet, logger)

	tests := []struct {
		name    string
		ip      string
		wantErr codes.Code
	}{
		{
			name:    "Allowed IP",
			ip:      "192.168.1.10",
			wantErr: codes.OK,
		},
		{
			name:    "Forbidden IP",
			ip:      "10.0.0.1",
			wantErr: codes.PermissionDenied,
		},
		{
			name:    "Missing metadata",
			ip:      "",
			wantErr: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.ip != "" {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-real-ip", tt.ip))
			}

			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, mockHandler)

			if tt.wantErr == codes.OK {
				assert.NoError(t, err)
			} else {
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantErr, st.Code())
			}
		})
	}
}

func TestSubnetInterceptor_EmptySubnet(t *testing.T) {
	logger := zap.NewNop()
	interceptor := SubnetInterceptor("", logger)
	mockHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, mockHandler)
	assert.NoError(t, err)
}
