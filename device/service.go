package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/joshjon/iot-metrics/http"
	"github.com/joshjon/iot-metrics/log"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1"
)

const (
	defaultPageSize, maxPageSize   = 100, 250
	minTemperature, maxTemperature = -10000.00, 10000.00
	minBattery, maxBattery         = 0, 100
)

type Service struct {
	repo   Repository
	logger log.Logger
}

func NewService(repo Repository, logger log.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) ConfigureDevice(ctx context.Context, req ConfigureDeviceRequest) error {
	if err := validateConfigureDeviceReq(req); err != nil {
		return err
	}

	cfg := Config{
		TemperatureThreshold: req.TemperatureThreshold,
		BatteryThreshold:     req.BatteryThreshold,
	}
	if err := s.repo.UpsertDeviceConfig(ctx, req.DeviceID, cfg); err != nil {
		return fmt.Errorf("upsert device config: %w", err)
	}

	s.logger.Info("configured device",
		"device_id", req.DeviceID,
		"temperature_threshold", req.TemperatureThreshold,
		"battery_threshold", req.BatteryThreshold,
	)

	return nil
}

func (s *Service) RecordMetric(ctx context.Context, req RecordMetricRequest) error {
	if err := validateRecordMetricReq(req); err != nil {
		return err
	}

	timestamp := req.Timestamp.UTC()
	logger := s.logger.With("device_id", req.DeviceID, "timestamp", timestamp.Format(time.RFC3339))

	metric := Metric{
		Temperature: req.Temperature,
		Battery:     req.Battery,
		Time:        timestamp,
	}
	if err := s.repo.SaveDeviceMetric(ctx, req.DeviceID, metric); err != nil {
		return fmt.Errorf("save device metric: %w", err)
	}

	logger.Info("recorded metric", "temperature", req.Temperature, "battery", req.Battery)

	cfg, err := s.repo.GetDeviceConfig(ctx, req.DeviceID)
	if err != nil {
		if errors.Is(err, ErrRepoItemNotFound) {
			// no thresholds configured for the device
			return nil
		}
	}

	if req.Temperature > cfg.TemperatureThreshold {
		alert := Alert{
			Reason: AlertReasonTemperatureHigh,
			Desc:   tempHighDesc(req.Temperature, cfg.TemperatureThreshold),
			Time:   timestamp,
		}
		logger.Info("alert triggered",
			"reason", alert.Reason,
			"temperature", req.Temperature,
			"threshold", cfg.TemperatureThreshold,
			"difference", fmt.Sprintf("%.2f", req.Temperature-cfg.TemperatureThreshold),
		)
		err = s.repo.SaveDeviceAlert(ctx, req.DeviceID, alert)
		if err != nil {
			return fmt.Errorf("save temperature alert: %w", err)
		}
	}

	if req.Battery < cfg.BatteryThreshold {
		alert := Alert{
			Reason: AlertReasonBatteryLow,
			Desc:   batteryLowDesc(req.Battery, cfg.BatteryThreshold),
			Time:   timestamp,
		}
		logger.Info("alert triggered",
			"reason", alert.Reason,
			"battery", req.Battery,
			"threshold", cfg.BatteryThreshold,
			"difference", cfg.BatteryThreshold-req.Battery,
		)
		err = s.repo.SaveDeviceAlert(ctx, req.DeviceID, alert)
		if err != nil {
			return fmt.Errorf("save battery alert: %w", err)
		}
	}

	return nil
}

func (s *Service) GetDeviceAlerts(ctx context.Context, req GetDeviceAlertsRequest) (GetDeviceAlertsResponse, error) {
	if err := validateGetDeviceAlertsReq(req); err != nil {
		return GetDeviceAlertsResponse{}, err
	}

	if req.PageSize == 0 {
		req.PageSize = defaultPageSize
	} else if req.PageSize > maxPageSize {
		req.PageSize = maxPageSize
	}

	var pageTkn *RepositoryPageToken
	if req.PageToken != "" {
		dec, err := decodePageToken(req.PageToken)
		if err != nil {
			return GetDeviceAlertsResponse{}, &http.BadRequestError{}
		}
		pageTkn = &dec
	}

	page, err := s.repo.GetDeviceAlerts(ctx, req.DeviceID, req.Timeframe, RepositoryPageOptions{
		Size:  req.PageSize,
		Token: pageTkn,
	})
	if err != nil {
		return GetDeviceAlertsResponse{}, fmt.Errorf("get device alerts: %w", err)
	}

	alertspb := make([]*iotv1.Alert, len(page.Items))
	for i, a := range page.Items {
		alertspb[i] = a.Proto()
	}

	var nextPageTkn string
	if page.NextPageToken != nil {
		if nextPageTkn, err = encodePageToken(*page.NextPageToken); err != nil {
			return GetDeviceAlertsResponse{}, err
		}
	}

	return GetDeviceAlertsResponse{
		Alerts:        page.Items,
		NextPageToken: nextPageTkn,
	}, nil
}

func tempHighDesc(temp float64, threshold float64) string {
	return fmt.Sprintf("Temperature (%.2f) exceeded configured threshold (%.2f)", temp, threshold)
}

func batteryLowDesc(battery int32, threshold int32) string {
	return fmt.Sprintf("Battery (%d) dropped below configured threshold (%d)", battery, threshold)
}

func ptr[T any](v T) *T {
	return &v
}
