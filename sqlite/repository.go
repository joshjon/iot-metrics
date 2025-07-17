package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/joshjon/iot-metrics/device"
	"github.com/joshjon/iot-metrics/sqlite/sqlc"
)

var _ device.Repository = (*DeviceRepository)(nil)

type DeviceRepository struct {
	querier sqlc.Querier
}

func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{
		querier: sqlc.New(db),
	}
}

func (d *DeviceRepository) UpsertDeviceConfig(ctx context.Context, deviceID string, config device.Config) error {
	return d.querier.UpsertDeviceConfig(ctx, sqlc.UpsertDeviceConfigParams{
		DeviceID:             deviceID,
		TemperatureThreshold: config.TemperatureThreshold,
		BatteryThreshold:     int64(config.BatteryThreshold),
	})
}

func (d *DeviceRepository) SaveDeviceMetric(ctx context.Context, deviceID string, metric device.Metric) error {
	return d.querier.SaveDeviceMetric(ctx, sqlc.SaveDeviceMetricParams{
		DeviceID:    deviceID,
		Temperature: metric.Temperature,
		Battery:     int64(metric.Battery),
		Timestamp:   metric.Time.Unix(),
	})
}

func (d *DeviceRepository) GetDeviceMetrics(
	ctx context.Context,
	deviceID string,
	timeframe device.Timeframe,
	pageOpts device.RepositoryPageOptions,
) (device.RepositoryPage[device.Metric], error) {
	params := sqlc.GetDeviceMetricsParams{
		DeviceID: deviceID,
		Limit:    int64(pageOpts.Size + 1),
	}
	if timeframe.Start != nil {
		params.StartTs = ptr(timeframe.Start.Unix())
	}
	if timeframe.End != nil {
		params.EndTs = ptr(timeframe.End.Unix())
	}
	if pageOpts.Token != nil {
		params.LastID = pageOpts.Token.LastID
		params.LastTs = ptr(pageOpts.Token.LastTime.Unix())
	}

	rows, err := d.querier.GetDeviceMetrics(ctx, params)
	if err != nil {
		return device.RepositoryPage[device.Metric]{}, err
	}

	var nextPageTkn *device.RepositoryPageToken
	// check if another page exists
	if len(rows) == int(params.Limit) {
		rows = rows[:len(rows)-1] // remove peeked row
		lastRow := rows[len(rows)-1]

		nextPageTkn = &device.RepositoryPageToken{
			LastID:   &lastRow.ID,
			LastTime: ptr(time.Unix(lastRow.Timestamp, 0).UTC()),
		}
	}

	metrics := make([]device.Metric, len(rows))
	for i, row := range rows {
		metrics[i] = device.Metric{
			Temperature: row.Temperature,
			Battery:     int32(row.Battery),
			Time:        time.Unix(row.Timestamp, 0).UTC(),
		}
	}

	return device.RepositoryPage[device.Metric]{
		Items:         metrics,
		NextPageToken: nextPageTkn,
	}, nil
}

func (d *DeviceRepository) GetDeviceConfig(ctx context.Context, deviceID string) (device.Config, error) {
	cfg, err := d.querier.GetDeviceConfig(ctx, deviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return device.Config{}, device.ErrRepoItemNotFound
		}
		return device.Config{}, err
	}
	return device.Config{
		TemperatureThreshold: cfg.TemperatureThreshold,
		BatteryThreshold:     int32(cfg.BatteryThreshold),
	}, nil
}

func (d *DeviceRepository) SaveDeviceAlert(ctx context.Context, deviceID string, alert device.Alert) error {
	return d.querier.SaveDeviceAlert(ctx, sqlc.SaveDeviceAlertParams{
		DeviceID:  deviceID,
		Reason:    string(alert.Reason),
		Desc:      alert.Desc,
		Timestamp: alert.Time.Unix(),
	})
}

func (d *DeviceRepository) GetDeviceAlerts(
	ctx context.Context,
	deviceID string,
	timeframe device.Timeframe,
	pageOpts device.RepositoryPageOptions,
) (device.RepositoryPage[device.Alert], error) {
	params := sqlc.GetDeviceAlertsParams{
		DeviceID: deviceID,
		Limit:    int64(pageOpts.Size + 1),
	}
	if timeframe.Start != nil {
		params.StartTs = ptr(timeframe.Start.Unix())
	}
	if timeframe.End != nil {
		params.EndTs = ptr(timeframe.End.Unix())
	}
	if pageOpts.Token != nil {
		params.LastID = pageOpts.Token.LastID
		params.LastTs = ptr(pageOpts.Token.LastTime.Unix())
	}

	rows, err := d.querier.GetDeviceAlerts(ctx, params)
	if err != nil {
		return device.RepositoryPage[device.Alert]{}, err
	}

	var nextPageTkn *device.RepositoryPageToken
	// check if another page exists
	if len(rows) == int(params.Limit) {
		rows = rows[:len(rows)-1] // remove peeked row
		lastRow := rows[len(rows)-1]

		nextPageTkn = &device.RepositoryPageToken{
			LastID:   &lastRow.ID,
			LastTime: ptr(time.Unix(lastRow.Timestamp, 0).UTC()),
		}
	}

	alerts := make([]device.Alert, len(rows))
	for i, row := range rows {
		alerts[i] = device.Alert{
			Reason: device.AlertReason(row.Reason),
			Desc:   row.Desc,
			Time:   time.Unix(row.Timestamp, 0).UTC(),
		}
	}

	return device.RepositoryPage[device.Alert]{
		Items:         alerts,
		NextPageToken: nextPageTkn,
	}, nil
}

func ptr[T any](v T) *T {
	return &v
}
