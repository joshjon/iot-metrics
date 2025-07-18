package device

import (
	"context"

	"connectrpc.com/connect"

	"github.com/joshjon/iot-metrics/proto/gen/iot/v1"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1/iotv1connect"
)

var _ iotv1connect.DeviceServiceHandler = (*ConnectHandler)(nil)

// ConnectHandler is a Connect/gRPC based handler for the IoT Device Metrics API.
type ConnectHandler struct {
	svc *Service
}

func NewConnectHandler(service *Service) *ConnectHandler {
	return &ConnectHandler{
		svc: service,
	}
}

func (s *ConnectHandler) ConfigureDevice(
	ctx context.Context,
	req *connect.Request[iotv1.ConfigureDeviceRequest],
) (*connect.Response[iotv1.ConfigureDeviceResponse], error) {
	if err := s.svc.ConfigureDevice(ctx, ConfigureDeviceRequest{
		DeviceID:             req.Msg.DeviceId,
		TemperatureThreshold: req.Msg.TemperatureThreshold,
		BatteryThreshold:     req.Msg.BatteryThreshold,
	}); err != nil {
		return nil, err
	}
	return &connect.Response[iotv1.ConfigureDeviceResponse]{}, nil
}

func (s *ConnectHandler) RecordMetric(
	ctx context.Context,
	req *connect.Request[iotv1.RecordMetricRequest],
) (*connect.Response[iotv1.RecordMetricResponse], error) {
	svcReq := RecordMetricRequest{
		DeviceID:    req.Msg.DeviceId,
		Temperature: req.Msg.Temperature,
		Battery:     req.Msg.Battery,
	}
	if req.Msg.Timestamp != nil {
		svcReq.Timestamp = req.Msg.Timestamp.AsTime()
	}
	if err := s.svc.RecordMetric(ctx, svcReq); err != nil {
		return nil, err
	}
	return &connect.Response[iotv1.RecordMetricResponse]{}, nil
}

func (s *ConnectHandler) GetDeviceAlerts(
	ctx context.Context,
	req *connect.Request[iotv1.GetDeviceAlertsRequest],
) (*connect.Response[iotv1.GetDeviceAlertsResponse], error) {
	svcReq := GetDeviceAlertsRequest{
		DeviceID:  req.Msg.DeviceId,
		PageSize:  int(req.Msg.PageSize),
		PageToken: req.Msg.PageToken,
	}
	if req.Msg.Timeframe != nil {
		if req.Msg.Timeframe.Start != nil {
			svcReq.TimeframeStart = ptr(req.Msg.Timeframe.Start.AsTime().UTC())
		}
		if req.Msg.Timeframe.End != nil {
			svcReq.TimeframeEnd = ptr(req.Msg.Timeframe.End.AsTime().UTC())
		}
	}
	res, err := s.svc.GetDeviceAlerts(ctx, svcReq)
	if err != nil {
		return nil, err
	}

	alertspb := make([]*iotv1.Alert, len(res.Alerts))
	for i, a := range res.Alerts {
		alertspb[i] = a.Proto()
	}
	return connect.NewResponse(&iotv1.GetDeviceAlertsResponse{
		Alerts:        alertspb,
		NextPageToken: res.NextPageToken,
	}), nil
}
