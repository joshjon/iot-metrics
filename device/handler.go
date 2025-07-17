package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/joshjon/iot-metrics/log"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1/iotv1connect"
)

const (
	defaultPageSize, maxPageSize   = 100, 250
	minTemperature, maxTemperature = -10000.00, 10000.00
	minBattery, maxBattery         = 0, 100
)

var _ iotv1connect.DeviceServiceHandler = (*Service)(nil)

type Service struct {
	repo   Repository
	logger log.Logger
}

func NewHandler(repo Repository, logger log.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) ConfigureDevice(
	ctx context.Context,
	req *connect.Request[iotv1.ConfigureDeviceRequest],
) (*connect.Response[iotv1.ConfigureDeviceResponse], error) {
	msg := req.Msg
	if err := validateConfigureDeviceReq(msg); err != nil {
		return nil, err
	}

	cfg := Config{
		TemperatureThreshold: msg.TemperatureThreshold,
		BatteryThreshold:     msg.BatteryThreshold,
	}
	if err := s.repo.UpsertDeviceConfig(ctx, msg.DeviceId, cfg); err != nil {
		return nil, fmt.Errorf("upsert device config: %w", err)
	}

	s.logger.Info("configured device",
		"device_id", msg.DeviceId,
		"temperature_threshold", msg.TemperatureThreshold,
		"battery_threshold", msg.BatteryThreshold,
	)

	return &connect.Response[iotv1.ConfigureDeviceResponse]{}, nil
}

func (s *Service) RecordMetric(
	ctx context.Context,
	req *connect.Request[iotv1.RecordMetricRequest],
) (*connect.Response[iotv1.RecordMetricResponse], error) {
	msg := req.Msg
	if err := validateRecordMetricReq(msg); err != nil {
		return nil, err
	}

	timestamp := msg.Timestamp.AsTime().UTC()
	logger := s.logger.With("device_id", msg.DeviceId, "timestamp", timestamp.Format(time.RFC3339))

	metric := Metric{
		Temperature: msg.Temperature,
		Battery:     msg.Battery,
		Time:        timestamp,
	}
	if err := s.repo.SaveDeviceMetric(ctx, msg.DeviceId, metric); err != nil {
		return nil, fmt.Errorf("save device metric: %w", err)
	}

	logger.Info("recorded metric", "temperature", msg.Temperature, "battery", msg.Battery)

	cfg, err := s.repo.GetDeviceConfig(ctx, msg.DeviceId)
	if err != nil {
		if errors.Is(err, ErrRepoItemNotFound) {
			// no thresholds configured for the device
			return &connect.Response[iotv1.RecordMetricResponse]{}, nil
		}
	}

	if msg.Temperature > cfg.TemperatureThreshold {
		alert := Alert{
			Reason: AlertReasonTemperatureHigh,
			Desc:   tempHighDesc(msg.Temperature, cfg.TemperatureThreshold),
			Time:   timestamp,
		}
		logger.Info("alert triggered",
			"reason", alert.Reason,
			"temperature", msg.Temperature,
			"threshold", cfg.TemperatureThreshold,
			"difference", fmt.Sprintf("%.2f", msg.Temperature-cfg.TemperatureThreshold),
		)
		err = s.repo.SaveDeviceAlert(ctx, msg.DeviceId, alert)
		if err != nil {
			return nil, fmt.Errorf("save temperature alert: %w", err)
		}
	}

	if msg.Battery < cfg.BatteryThreshold {
		alert := Alert{
			Reason: AlertReasonBatteryLow,
			Desc:   batteryLowDesc(msg.Battery, cfg.BatteryThreshold),
			Time:   timestamp,
		}
		logger.Info("alert triggered",
			"reason", alert.Reason,
			"battery", msg.Battery,
			"threshold", cfg.BatteryThreshold,
			"difference", cfg.BatteryThreshold-msg.Battery,
		)
		err = s.repo.SaveDeviceAlert(ctx, msg.DeviceId, alert)
		if err != nil {
			return nil, fmt.Errorf("save battery alert: %w", err)
		}
	}

	return &connect.Response[iotv1.RecordMetricResponse]{}, nil
}

func (s *Service) GetDeviceAlerts(
	ctx context.Context,
	req *connect.Request[iotv1.GetDeviceAlertsRequest],
) (*connect.Response[iotv1.GetDeviceAlertsResponse], error) {
	msg := req.Msg
	if err := validateGetDeviceAlertsReq(msg); err != nil {
		return nil, err
	}

	if msg.PageSize == 0 {
		msg.PageSize = defaultPageSize
	} else if msg.PageSize > maxPageSize {
		msg.PageSize = maxPageSize
	}

	var pageTkn *RepositoryPageToken
	if req.Msg.PageToken != "" {
		dec, err := decodePageToken(req.Msg.PageToken, func(tkn *iotv1.PageToken) bool {
			return tkn.DeviceId == req.Msg.DeviceId && proto.Equal(tkn.Timeframe, req.Msg.Timeframe)
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid page token"))
		}
		pageTkn = &dec
	}

	timeframe := unmarshalTimeframe(req.Msg.Timeframe)
	page, err := s.repo.GetDeviceAlerts(ctx, msg.DeviceId, timeframe, RepositoryPageOptions{
		Size:  int(req.Msg.PageSize),
		Token: pageTkn,
	})
	if err != nil {
		return nil, fmt.Errorf("get device alerts: %w", err)
	}

	alertspb := make([]*iotv1.Alert, len(page.Items))
	for i, a := range page.Items {
		alertspb[i] = a.Proto()
	}

	var nextPageTkn string
	if page.NextPageToken != nil {
		if nextPageTkn, err = encodePageToken(*page.NextPageToken, func(tkn *iotv1.PageToken) {
			tkn.DeviceId = req.Msg.DeviceId
			tkn.Timeframe = req.Msg.Timeframe
		}); err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(&iotv1.GetDeviceAlertsResponse{
		Alerts:        alertspb,
		NextPageToken: nextPageTkn,
	}), nil
}

func tempHighDesc(temp float64, threshold float64) string {
	return fmt.Sprintf("Temperature (%.2f) exceeded configured threshold (%.2f)", temp, threshold)
}

func batteryLowDesc(battery int32, threshold int32) string {
	return fmt.Sprintf("Battery (%d) dropped below configured threshold (%d)", battery, threshold)
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

func ptr[T any](v T) *T {
	return &v
}
