package proto

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestMetricProto(t *testing.T) {
	m := &Metric{
		Id:    "test",
		Type:  Metric_GAUGE,
		Value: 1.23,
	}
	assert.Equal(t, "test", m.GetId())
	assert.Equal(t, Metric_GAUGE, m.GetType())
	assert.Equal(t, 1.23, m.GetValue())
	assert.NotEmpty(t, m.String())

	// Call descriptor and other generated methods for coverage
	_, _ = m.Descriptor()
	_ = m.ProtoReflect()

	m2 := &Metric{}
	m2.Reset()
	assert.Empty(t, m2.GetId())

	m3 := &Metric{
		Id:    "test2",
		Type:  Metric_COUNTER,
		Delta: 10,
	}
	assert.Equal(t, int64(10), m3.GetDelta())
}

func TestUpdateMetricsRequest(t *testing.T) {
	req := &UpdateMetricsRequest{
		Metrics: []*Metric{
			{Id: "m1", Type: Metric_GAUGE, Value: 1.0},
		},
	}
	assert.Len(t, req.GetMetrics(), 1)
	assert.NotEmpty(t, req.String())

	_, _ = req.Descriptor()
	_ = req.ProtoReflect()

	req.Reset()
	assert.Nil(t, req.GetMetrics())
}

func TestUpdateMetricsResponse(t *testing.T) {
	resp := &UpdateMetricsResponse{}
	_ = resp.String()
	_, _ = resp.Descriptor()
	_ = resp.ProtoReflect()

	resp.Reset()
	assert.NotNil(t, resp)
}

type mockServer struct {
	UnimplementedMetricsServer
}

func (s *mockServer) UpdateMetrics(ctx context.Context, req *UpdateMetricsRequest) (*UpdateMetricsResponse, error) {
	return &UpdateMetricsResponse{}, nil
}

func TestGRPCServiceRegistration(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	RegisterMetricsServer(s, &mockServer{})

	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := NewMetricsClient(conn)
	assert.NotNil(t, client)
}

func TestEnums(t *testing.T) {
	assert.Equal(t, "GAUGE", Metric_GAUGE.String())
	assert.Equal(t, "COUNTER", Metric_COUNTER.String())

	val, ok := Metric_MType_value["GAUGE"]
	assert.True(t, ok)
	assert.Equal(t, int32(Metric_GAUGE), val)
}
