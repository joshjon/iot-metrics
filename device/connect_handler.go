package device

import (
	"context"

	"connectrpc.com/connect"

	"github.com/joshjon/iot-metrics/proto/gen/iot/v1"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1/iotv1connect"
)

var _ iotv1connect.DeviceServiceHandler = (*ConnectHandler)(nil)

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
	res, err := s.svc.GetDeviceAlerts(ctx, GetDeviceAlertsRequest{
		DeviceID:  req.Msg.DeviceId,
		Timeframe: unmarshalTimeframe(req.Msg.Timeframe),
		PageSize:  int(req.Msg.PageSize),
		PageToken: req.Msg.PageToken,
	})
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

func unmarshalTimeframe(tfpb *iotv1.Timeframe) Timeframe {
	timeframe := Timeframe{}
	if tfpb != nil {
		if tfpb.Start != nil {
			timeframe.Start = ptr(tfpb.Start.AsTime().UTC())
		}
		if tfpb.End != nil {
			timeframe.End = ptr(tfpb.End.AsTime().UTC())
		}
	}
	return timeframe
}
