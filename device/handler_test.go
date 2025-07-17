package device

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/joshjon/iot-metrics/log"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1"
)

func TestHandler_ConfigureDevice(t *testing.T) {
	ctx := t.Context()

	req := &iotv1.ConfigureDeviceRequest{
		DeviceId:             "foo",
		TemperatureThreshold: 5.55,
		BatteryThreshold:     5,
	}

	r := &RepositoryMock{
		UpsertDeviceConfigFunc: func(ctx context.Context, deviceID string, cfg Config) error {
			assert.Equal(t, req.DeviceId, deviceID)
			assert.Equal(t, req.TemperatureThreshold, cfg.TemperatureThreshold)
			assert.Equal(t, req.BatteryThreshold, cfg.BatteryThreshold)
			return nil
		},
	}

	h := NewHandler(r, log.NewLogger())
	res, err := h.ConfigureDevice(ctx, connect.NewRequest(req))
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestHandler_ConfigureDevice_requestValidation(t *testing.T) {
	validReq := &iotv1.ConfigureDeviceRequest{
		DeviceId:             "foo",
		TemperatureThreshold: 5.55,
		BatteryThreshold:     5,
	}

	tests := []struct {
		name      string
		fieldName string
		override  func(req *iotv1.ConfigureDeviceRequest)
	}{
		{
			name:      "empty device id",
			fieldName: "device_id",
			override: func(req *iotv1.ConfigureDeviceRequest) {
				req.DeviceId = ""
			},
		},
		{
			name:      "temp threshold below minimum",
			fieldName: "temperature_threshold",
			override: func(req *iotv1.ConfigureDeviceRequest) {
				req.TemperatureThreshold = minTemperature - 0.01
			},
		},
		{
			name:      "temp threshold above maximum",
			fieldName: "temperature_threshold",
			override: func(req *iotv1.ConfigureDeviceRequest) {
				req.TemperatureThreshold = maxTemperature + 0.01
			},
		},
		{
			name:      "temp threshold below minimum",
			fieldName: "battery_threshold",
			override: func(req *iotv1.ConfigureDeviceRequest) {
				req.BatteryThreshold = minBattery - 1
			},
		},
		{
			name:      "battery threshold above maximum",
			fieldName: "battery_threshold",
			override: func(req *iotv1.ConfigureDeviceRequest) {
				req.BatteryThreshold = maxBattery + 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			req := proto.Clone(validReq).(*iotv1.ConfigureDeviceRequest)
			require.NotNil(t, tt.override, "test config `override` field must not be nil")
			tt.override(req)

			h := NewHandler(nil, log.NewLogger())

			res, err := h.ConfigureDevice(ctx, connect.NewRequest(req))
			require.Error(t, err)
			assert.Nil(t, res)
			assertFieldViolationErr(t, err, tt.fieldName, 1)
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

			req := &iotv1.RecordMetricRequest{
				DeviceId:    "foo",
				Temperature: tt.reqTemp,
				Battery:     tt.reqBattery,
				Timestamp:   timestamppb.Now(),
			}

			var gotAlerts []Alert

			r := &RepositoryMock{
				SaveDeviceMetricFunc: func(ctx context.Context, deviceID string, metric Metric) error {
					assert.Equal(t, req.DeviceId, deviceID)
					assert.Equal(t, req.Temperature, metric.Temperature)
					assert.Equal(t, req.Battery, metric.Battery)
					assert.Equal(t, req.Timestamp.AsTime(), metric.Time)
					return nil
				},
				GetDeviceConfigFunc: func(ctx context.Context, deviceID string) (Config, error) {
					assert.Equal(t, req.DeviceId, deviceID)
					if tt.deviceCfg == nil {
						return Config{}, ErrRepoItemNotFound
					}
					return *tt.deviceCfg, nil
				},
				SaveDeviceAlertFunc: func(ctx context.Context, deviceID string, alert Alert) error {
					assert.Equal(t, req.DeviceId, deviceID)
					gotAlerts = append(gotAlerts, alert)
					return nil
				},
			}

			h := NewHandler(r, log.NewLogger())

			res, err := h.RecordMetric(ctx, connect.NewRequest(req))
			require.NoError(t, err)
			assert.NotNil(t, res)

			wantAlertsLen := 0
			if tt.wantTempAlert {
				wantAlertsLen++
				assert.Contains(t, gotAlerts, Alert{
					Reason: AlertReasonTemperatureHigh,
					Desc:   tempHighDesc(req.Temperature, tt.deviceCfg.TemperatureThreshold),
					Time:   req.Timestamp.AsTime(),
				})
			}
			if tt.wantBatteryAlert {
				wantAlertsLen++
				assert.Contains(t, gotAlerts, Alert{
					Reason: AlertReasonBatteryLow,
					Desc:   batteryLowDesc(req.Battery, tt.deviceCfg.BatteryThreshold),
					Time:   req.Timestamp.AsTime(),
				})
			}
			require.Len(t, gotAlerts, wantAlertsLen)
		})
	}
}

func TestHandler_RecordMetric_requestValidation(t *testing.T) {
	validReq := &iotv1.RecordMetricRequest{
		DeviceId:    "foo",
		Temperature: 5.55,
		Battery:     5,
		Timestamp:   timestamppb.Now(),
	}

	tests := []struct {
		name      string
		fieldName string
		override  func(req *iotv1.RecordMetricRequest)
	}{
		{
			name:      "empty device id",
			fieldName: "device_id",
			override: func(req *iotv1.RecordMetricRequest) {
				req.DeviceId = ""
			},
		},
		{
			name:      "temp below minimum",
			fieldName: "temperature",
			override: func(req *iotv1.RecordMetricRequest) {
				req.Temperature = minTemperature - 0.01
			},
		},
		{
			name:      "temp above maximum",
			fieldName: "temperature",
			override: func(req *iotv1.RecordMetricRequest) {
				req.Temperature = maxTemperature + 0.01
			},
		},
		{
			name:      "temp below minimum",
			fieldName: "battery",
			override: func(req *iotv1.RecordMetricRequest) {
				req.Battery = minBattery - 1
			},
		},
		{
			name:      "battery above maximum",
			fieldName: "battery",
			override: func(req *iotv1.RecordMetricRequest) {
				req.Battery = maxBattery + 1
			},
		},
		{
			name:      "nil timestamp",
			fieldName: "timestamp",
			override: func(req *iotv1.RecordMetricRequest) {
				req.Timestamp = nil
			},
		},
		{
			name:      "empty timestamp",
			fieldName: "timestamp",
			override: func(req *iotv1.RecordMetricRequest) {
				req.Timestamp = &timestamppb.Timestamp{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			req := proto.Clone(validReq).(*iotv1.RecordMetricRequest)
			require.NotNil(t, tt.override, "test config `override` field must not be nil")
			tt.override(req)

			h := NewHandler(nil, log.NewLogger())

			res, err := h.RecordMetric(ctx, connect.NewRequest(req))
			require.Error(t, err)
			assert.Nil(t, res)
			assertFieldViolationErr(t, err, tt.fieldName, 1)
		})
	}
}

func TestHandler_GetDeviceAlerts(t *testing.T) {
	ctx := t.Context()

	wantTimeframe := Timeframe{
		Start: ptr(time.Now().Add(-time.Minute).UTC()),
		End:   ptr(time.Now().UTC()),
	}

	req := &iotv1.GetDeviceAlertsRequest{
		DeviceId: "foo",
		Timeframe: &iotv1.Timeframe{
			Start: timestamppb.New(*wantTimeframe.Start),
			End:   timestamppb.New(*wantTimeframe.End),
		},
		PageSize: 10,
	}
	ptkn := RepositoryPageToken{
		LastTime: ptr(time.Now().Add(-10 * time.Second).UTC()),
		LastID:   ptr[int64](1),
	}
	req.PageToken = newEncodedPageToken(t, req.DeviceId, req.Timeframe, ptkn)

	alerts := []Alert{
		{Reason: AlertReasonTemperatureHigh, Desc: tempHighDesc(5.56, 5.55), Time: time.Now()},
		{Reason: AlertReasonBatteryLow, Desc: batteryLowDesc(5, 6), Time: time.Now()},
	}
	wantAlerts := make([]*iotv1.Alert, len(alerts))
	for i, alert := range alerts {
		wantAlerts[i] = alert.Proto()
	}

	nextPageTkn := RepositoryPageToken{
		LastTime: ptr(time.Now().Add(-20 * time.Second)),
		LastID:   ptr[int64](2),
	}
	wantNextPageTkn := newEncodedPageToken(t, req.DeviceId, req.Timeframe, nextPageTkn)

	r := &RepositoryMock{
		GetDeviceAlertsFunc: func(ctx context.Context, deviceID string, timeframe Timeframe, pageOpts RepositoryPageOptions) (RepositoryPage[Alert], error) {
			assert.Equal(t, req.DeviceId, deviceID)
			assert.Equal(t, wantTimeframe, timeframe)
			assert.Equal(t, int(req.PageSize), pageOpts.Size)
			assert.Equal(t, ptkn, *pageOpts.Token)
			return RepositoryPage[Alert]{
				Items:         alerts,
				NextPageToken: &nextPageTkn,
			}, nil
		},
	}

	h := NewHandler(r, log.NewLogger())

	gotRes, err := h.GetDeviceAlerts(ctx, connect.NewRequest(req))
	require.NoError(t, err)
	require.NotNil(t, gotRes)

	wantRes := &iotv1.GetDeviceAlertsResponse{
		Alerts:        wantAlerts,
		NextPageToken: wantNextPageTkn,
	}
	assert.Equal(t, wantRes, gotRes.Msg)
}

func TestHandler_GetDeviceAlerts_requestValidation(t *testing.T) {
	validReq := &iotv1.GetDeviceAlertsRequest{
		DeviceId: "foo",
		Timeframe: &iotv1.Timeframe{
			Start: timestamppb.New(time.Now().Add(-time.Minute)),
			End:   timestamppb.New(time.Now()),
		},
		PageSize: 10,
	}
	validReq.PageToken = newEncodedPageToken(t, validReq.DeviceId, validReq.Timeframe, RepositoryPageToken{
		LastTime: ptr(time.Now().Add(-10 * time.Second).UTC()),
		LastID:   ptr[int64](1),
	})

	tests := []struct {
		name      string
		fieldName string
		override  func(req *iotv1.GetDeviceAlertsRequest)
	}{
		{
			name:      "empty device id",
			fieldName: "device_id",
			override: func(req *iotv1.GetDeviceAlertsRequest) {
				req.DeviceId = ""
			},
		},
		{
			name:      "Timeframe start is invalid",
			fieldName: "timeframe.start",
			override: func(req *iotv1.GetDeviceAlertsRequest) {
				req.Timeframe.Start = &timestamppb.Timestamp{
					Nanos: 1_000_000_000,
				}
			},
		},
		{
			name:      "Timeframe end is invalid",
			fieldName: "timeframe.end",
			override: func(req *iotv1.GetDeviceAlertsRequest) {
				req.Timeframe.End = &timestamppb.Timestamp{
					Nanos: 1_000_000_000,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			req := proto.Clone(validReq).(*iotv1.GetDeviceAlertsRequest)
			require.NotNil(t, tt.override, "test config `override` field must not be nil")
			tt.override(req)

			h := NewHandler(nil, log.NewLogger())

			res, err := h.GetDeviceAlerts(ctx, connect.NewRequest(req))
			require.Error(t, err)
			assert.Nil(t, res)
			assertFieldViolationErr(t, err, tt.fieldName, 1)
		})
	}
}

func TestHandler_GetDeviceAlerts_incompatiblePageToken(t *testing.T) {
	ctx := t.Context()

	req := &iotv1.GetDeviceAlertsRequest{
		DeviceId: "foo",
		Timeframe: &iotv1.Timeframe{
			Start: timestamppb.New(time.Now().Add(-time.Minute).UTC()),
			End:   timestamppb.New(time.Now().UTC()),
		},
		PageSize: 10,
	}
	ptkn := RepositoryPageToken{
		LastTime: ptr(time.Now().Add(-10 * time.Second).UTC()),
		LastID:   ptr[int64](1),
	}
	// create token with mismatched request values
	encPtkn, err := encodePageToken(ptkn, func(tkn *iotv1.PageToken) {
		tkn.DeviceId = "random-device-id"
		tkn.Timeframe = proto.CloneOf(req.Timeframe)
		tkn.Timeframe.Start = timestamppb.New(time.Now().Add(-time.Hour))
	})
	require.NoError(t, err)
	req.PageToken = encPtkn

	h := NewHandler(nil, log.NewLogger())

	res, err := h.GetDeviceAlerts(ctx, connect.NewRequest(req))
	require.Error(t, err)
	assert.Nil(t, res)

	var cerr *connect.Error
	require.ErrorAs(t, err, &cerr)

	assert.Equal(t, connect.CodeInvalidArgument, cerr.Code())
}

func assertFieldViolationErr(t *testing.T, err error, field string, numViolations int) {
	t.Helper()
	var cerr *connect.Error
	require.ErrorAs(t, err, &cerr)

	assert.Equal(t, connect.CodeInvalidArgument, cerr.Code())

	require.Len(t, cerr.Details(), 1)
	val, err := cerr.Details()[0].Value()
	require.NoError(t, err)

	badReq, ok := val.(*errdetails.BadRequest)
	require.True(t, ok, "proto message is not BadRequest")

	require.Len(t, badReq.FieldViolations, 1)

	gotCount := 0
	for _, fv := range badReq.FieldViolations {
		if fv.Field == field {
			gotCount++
		}
	}
	assert.Equal(t, numViolations, gotCount)
}

func newEncodedPageToken(t *testing.T, deviceID string, timeframe *iotv1.Timeframe, rTkn RepositoryPageToken) string {
	t.Helper()
	encPtkn, err := encodePageToken(rTkn, func(tkn *iotv1.PageToken) {
		tkn.DeviceId = deviceID
		tkn.Timeframe = timeframe
	})
	require.NoError(t, err)
	return encPtkn
}
