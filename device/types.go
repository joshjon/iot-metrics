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
	Reason string
	Time   time.Time
}

func (a Alert) Proto() *iotv1.Alert {
	return &iotv1.Alert{
		Reason:    a.Reason,
		Timestamp: timestamppb.New(a.Time),
	}
}

type Timeframe struct {
	Start *time.Time
	End   *time.Time
}
