package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/joshjon/iot-metrics/proto/gen/health/v1"
	v1 "github.com/joshjon/iot-metrics/proto/gen/health/v1"
)

type healthService struct{}

func (h *healthService) GetHealth(
	_ context.Context,
	_ *connect.Request[v1.GetHealthRequest],
) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&healthv1.GetHealthResponse{}), nil
}
