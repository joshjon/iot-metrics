package device

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joshjon/iot-metrics/log"
)

func TestHandler_ConfigureDevice(t *testing.T) {
	ctx := t.Context()

	req := ConfigureDeviceRequest{
		DeviceID:             "foo",
		TemperatureThreshold: 5.55,
		BatteryThreshold:     5,
	}

	r := &RepositoryMock{
		UpsertDeviceConfigFunc: func(ctx context.Context, deviceID string, cfg Config) error {
			assert.Equal(t, req.DeviceID, deviceID)
			assert.Equal(t, req.TemperatureThreshold, cfg.TemperatureThreshold)
			assert.Equal(t, req.BatteryThreshold, cfg.BatteryThreshold)
			return nil
		},
	}

	s := NewService(r, log.NewLogger())
	err := s.ConfigureDevice(ctx, req)
	assert.NoError(t, err)
}

func TestHandler_ConfigureDevice_requestValidation(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		override  func(req *ConfigureDeviceRequest)
	}{
		{
			name:      "empty device id",
			fieldName: "device_id",
			override: func(req *ConfigureDeviceRequest) {
				req.DeviceID = ""
			},
		},
		{
			name:      "temp threshold below minimum",
			fieldName: "temperature_threshold",
			override: func(req *ConfigureDeviceRequest) {
				req.TemperatureThreshold = minTemperature - 0.01
			},
		},
		{
			name:      "temp threshold above maximum",
			fieldName: "temperature_threshold",
			override: func(req *ConfigureDeviceRequest) {
				req.TemperatureThreshold = maxTemperature + 0.01
			},
		},
		{
			name:      "temp threshold below minimum",
			fieldName: "battery_threshold",
			override: func(req *ConfigureDeviceRequest) {
				req.BatteryThreshold = minBattery - 1
			},
		},
		{
			name:      "battery threshold above maximum",
			fieldName: "battery_threshold",
			override: func(req *ConfigureDeviceRequest) {
				req.BatteryThreshold = maxBattery + 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			req := ConfigureDeviceRequest{
				DeviceID:             "foo",
				TemperatureThreshold: 5.55,
				BatteryThreshold:     5,
			}
			require.NotNil(t, tt.override, "test config `override` field must not be nil")
			tt.override(&req)

			h := NewService(nil, log.NewLogger())
			err := h.ConfigureDevice(ctx, req)
			require.Error(t, err)
		})
	}
}

func TestHandler_RecordMetric(t *testing.T) {
	stubCfg := &Config{TemperatureThreshold: 5.55, BatteryThreshold: 5}

	tests := []struct {
		name             string
		deviceCfg        *Config
		reqTemp          float64
		reqBattery       int32
		wantTempAlert    bool
		wantBatteryAlert bool
	}{
		{
			name:      "without existing config",
			deviceCfg: nil,
		},
		{
			name:       "without breaching thresholds",
			deviceCfg:  stubCfg,
			reqTemp:    5.55,
			reqBattery: 5,
		},
		{
			name:          "breach temperature threshold",
			deviceCfg:     stubCfg,
			reqTemp:       stubCfg.TemperatureThreshold + 0.1,
			reqBattery:    stubCfg.BatteryThreshold,
			wantTempAlert: true,
		},
		{
			name:             "breach battery threshold",
			deviceCfg:        stubCfg,
			reqTemp:          stubCfg.TemperatureThreshold,
			reqBattery:       stubCfg.BatteryThreshold - 1,
			wantBatteryAlert: true,
		},
		{
			name:             "breach both thresholds",
			deviceCfg:        stubCfg,
			reqTemp:          stubCfg.TemperatureThreshold + 0.1,
			reqBattery:       stubCfg.BatteryThreshold - 1,
			wantTempAlert:    true,
			wantBatteryAlert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			req := RecordMetricRequest{
				DeviceID:    "foo",
				Temperature: tt.reqTemp,
				Battery:     tt.reqBattery,
				Timestamp:   time.Now().UTC(),
			}

			var gotAlerts []Alert

			r := &RepositoryMock{
				SaveDeviceMetricFunc: func(ctx context.Context, deviceID string, metric Metric) error {
					assert.Equal(t, req.DeviceID, deviceID)
					assert.Equal(t, req.Temperature, metric.Temperature)
					assert.Equal(t, req.Battery, metric.Battery)
					assert.Equal(t, req.Timestamp, metric.Time)
					return nil
				},
				GetDeviceConfigFunc: func(ctx context.Context, deviceID string) (Config, error) {
					assert.Equal(t, req.DeviceID, deviceID)
					if tt.deviceCfg == nil {
						return Config{}, ErrRepoItemNotFound
					}
					return *tt.deviceCfg, nil
				},
				SaveDeviceAlertFunc: func(ctx context.Context, deviceID string, alert Alert) error {
					assert.Equal(t, req.DeviceID, deviceID)
					gotAlerts = append(gotAlerts, alert)
					return nil
				},
			}

			h := NewService(r, log.NewLogger())

			err := h.RecordMetric(ctx, req)
			require.NoError(t, err)

			wantAlertsLen := 0
			if tt.wantTempAlert {
				wantAlertsLen++
				assert.Contains(t, gotAlerts, Alert{
					Reason: AlertReasonTemperatureHigh,
					Desc:   tempHighDesc(req.Temperature, tt.deviceCfg.TemperatureThreshold),
					Time:   req.Timestamp,
				})
			}
			if tt.wantBatteryAlert {
				wantAlertsLen++
				assert.Contains(t, gotAlerts, Alert{
					Reason: AlertReasonBatteryLow,
					Desc:   batteryLowDesc(req.Battery, tt.deviceCfg.BatteryThreshold),
					Time:   req.Timestamp,
				})
			}
			require.Len(t, gotAlerts, wantAlertsLen)
		})
	}
}

func TestHandler_RecordMetric_requestValidation(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		override  func(req *RecordMetricRequest)
	}{
		{
			name:      "empty device id",
			fieldName: "device_id",
			override: func(req *RecordMetricRequest) {
				req.DeviceID = ""
			},
		},
		{
			name:      "temp below minimum",
			fieldName: "temperature",
			override: func(req *RecordMetricRequest) {
				req.Temperature = minTemperature - 0.01
			},
		},
		{
			name:      "temp above maximum",
			fieldName: "temperature",
			override: func(req *RecordMetricRequest) {
				req.Temperature = maxTemperature + 0.01
			},
		},
		{
			name:      "temp below minimum",
			fieldName: "battery",
			override: func(req *RecordMetricRequest) {
				req.Battery = minBattery - 1
			},
		},
		{
			name:      "battery above maximum",
			fieldName: "battery",
			override: func(req *RecordMetricRequest) {
				req.Battery = maxBattery + 1
			},
		},
		{
			name:      "empty timestamp",
			fieldName: "timestamp",
			override: func(req *RecordMetricRequest) {
				req.Timestamp = time.Time{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			req := RecordMetricRequest{
				DeviceID:    "foo",
				Temperature: 5.55,
				Battery:     5,
				Timestamp:   time.Now().UTC(),
			}
			require.NotNil(t, tt.override, "test config `override` field must not be nil")
			tt.override(&req)

			h := NewService(nil, log.NewLogger())

			err := h.RecordMetric(ctx, req)
			require.Error(t, err)
		})
	}
}

func TestHandler_GetDeviceAlerts(t *testing.T) {
	ctx := t.Context()

	wantTimeframe := Timeframe{
		Start: ptr(time.Now().Add(-time.Minute).UTC()),
		End:   ptr(time.Now().UTC()),
	}

	req := GetDeviceAlertsRequest{
		DeviceID: "foo",
		Timeframe: Timeframe{
			Start: wantTimeframe.Start,
			End:   wantTimeframe.End,
		},
		PageSize: 10,
	}
	ptkn := RepositoryPageToken{
		LastTime: ptr(time.Now().Add(-10 * time.Second).UTC()),
		LastID:   ptr[int64](1),
	}
	reqTkn, err := encodePageToken(ptkn)
	require.NoError(t, err)
	req.PageToken = reqTkn

	alerts := []Alert{
		{Reason: AlertReasonTemperatureHigh, Desc: tempHighDesc(5.56, 5.55), Time: time.Now()},
		{Reason: AlertReasonBatteryLow, Desc: batteryLowDesc(5, 6), Time: time.Now()},
	}
	nextPageTkn := RepositoryPageToken{
		LastTime: ptr(time.Now().Add(-20 * time.Second)),
		LastID:   ptr[int64](2),
	}
	wantNextPageTkn, err := encodePageToken(nextPageTkn)
	require.NoError(t, err)

	r := &RepositoryMock{
		GetDeviceAlertsFunc: func(ctx context.Context, deviceID string, timeframe Timeframe, pageOpts RepositoryPageOptions) (RepositoryPage[Alert], error) {
			assert.Equal(t, req.DeviceID, deviceID)
			assert.Equal(t, wantTimeframe, timeframe)
			assert.Equal(t, int(req.PageSize), pageOpts.Size)
			assert.Equal(t, ptkn, *pageOpts.Token)
			return RepositoryPage[Alert]{
				Items:         alerts,
				NextPageToken: &nextPageTkn,
			}, nil
		},
	}

	h := NewService(r, log.NewLogger())

	gotRes, err := h.GetDeviceAlerts(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, gotRes)

	wantRes := GetDeviceAlertsResponse{
		Alerts:        alerts,
		NextPageToken: wantNextPageTkn,
	}
	assert.Equal(t, wantRes, gotRes)
}

func TestHandler_GetDeviceAlerts_requestValidation(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		override  func(req *GetDeviceAlertsRequest)
	}{
		{
			name:      "empty device id",
			fieldName: "device_id",
			override: func(req *GetDeviceAlertsRequest) {
				req.DeviceID = ""
			},
		},
		{
			name:      "Timeframe start is empty",
			fieldName: "timeframe.start",
			override: func(req *GetDeviceAlertsRequest) {
				req.Timeframe.Start = &time.Time{}
			},
		},
		{
			name:      "Timeframe end is empty",
			fieldName: "timeframe.end",
			override: func(req *GetDeviceAlertsRequest) {
				req.Timeframe.End = &time.Time{}
			},
		},
		{
			name:      "Timeframe start is after end",
			fieldName: "timeframe.start",
			override: func(req *GetDeviceAlertsRequest) {
				req.Timeframe.Start = ptr(time.Now().UTC())
				req.Timeframe.End = ptr(time.Now().Add(-time.Minute).UTC())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			req := GetDeviceAlertsRequest{
				DeviceID: "foo",
				Timeframe: Timeframe{
					Start: ptr(time.Now().Add(-time.Minute).UTC()),
					End:   ptr(time.Now().UTC()),
				},
				PageSize: 10,
			}
			reqTkn, err := encodePageToken(RepositoryPageToken{
				LastTime: ptr(time.Now().Add(-10 * time.Second).UTC()),
				LastID:   ptr[int64](1),
			})
			require.NoError(t, err)
			req.PageToken = reqTkn

			require.NotNil(t, tt.override, "test config `override` field must not be nil")
			tt.override(&req)

			h := NewService(nil, log.NewLogger())

			_, err = h.GetDeviceAlerts(ctx, req)
			require.Error(t, err)
		})
	}
}
