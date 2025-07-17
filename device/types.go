package device

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	iotv1 "github.com/joshjon/iot-metrics/proto/gen/iot/v1"
)

type Config struct {
	TemperatureThreshold float64
	BatteryThreshold     int32
}

type Metric struct {
	Temperature float64
	Battery     int32
	Time        time.Time
}

type Alert struct {
	Reason AlertReason
	Desc   string
	Time   time.Time
}

func (a Alert) Proto() *iotv1.Alert {
	return &iotv1.Alert{
		Reason:      a.Reason.Proto(),
		Description: a.Desc,
		Timestamp:   timestamppb.New(a.Time),
	}
}

const (
	AlertReasonTemperatureHigh AlertReason = "TEMPERATURE_HIGH"
	AlertReasonBatteryLow      AlertReason = "BATTERY_LOW"
)

type AlertReason string

func (r AlertReason) Proto() iotv1.Alert_Reason {
	switch r {
	case AlertReasonTemperatureHigh:
		return iotv1.Alert_REASON_TEMPERATURE_HIGH
	case AlertReasonBatteryLow:
		return iotv1.Alert_REASON_BATTERY_LOW
	}
	return iotv1.Alert_REASON_UNSPECIFIED
}

type Timeframe struct {
	Start *time.Time
	End   *time.Time
}
