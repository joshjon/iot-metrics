package device

import (
	"strings"

	iotv1 "github.com/joshjon/iot-metrics/proto/gen/iot/v1"
	"github.com/joshjon/iot-metrics/rpc"
)

func validateConfigureDeviceReq(msg *iotv1.ConfigureDeviceRequest) error {
	v := rpc.NewRequestValidator()
	v.Field("device_id").When(isBlank(msg.DeviceId)).Message("Must not be blank")
	v.Field("temperature_threshold").
		When(msg.TemperatureThreshold < minTemperature || msg.TemperatureThreshold > maxTemperature).
		Messagef("Must be between %.2f and %.2f", minTemperature, maxTemperature)
	v.Field("battery_threshold").
		When(msg.BatteryThreshold < minBattery || msg.BatteryThreshold > maxBattery).
		Messagef("Must be between %d and %d", minBattery, maxBattery)
	return v.Error()
}

func validateRecordMetricReq(msg *iotv1.RecordMetricRequest) error {
	v := rpc.NewRequestValidator()
	v.Field("device_id").When(isBlank(msg.DeviceId)).Message("Must not be blank")
	v.Field("timestamp").
		When(msg.Timestamp == nil || (msg.Timestamp.Seconds+int64(msg.Timestamp.Nanos) == 0)).
		Message("Must not be empty")
	v.Field("temperature").
		When(msg.Temperature < minTemperature || msg.Temperature > maxTemperature).
		Messagef("Must be between %.2f and %.2f", minTemperature, maxTemperature)
	v.Field("battery").
		When(msg.Battery < minBattery || msg.Battery > maxBattery).
		Messagef("Must be between %d and %d", minBattery, maxBattery)
	return v.Error()
}

func validateGetDeviceAlertsReq(msg *iotv1.GetDeviceAlertsRequest) error {
	v := rpc.NewRequestValidator()
	v.Field("device_id").When(isBlank(msg.DeviceId)).Message("Must not be blank")
	if msg.Timeframe != nil {
		v.Field("timeframe.start").
			When(msg.Timeframe.Start != nil && !msg.Timeframe.Start.IsValid()).
			Message("Invalid timestamp")
		v.Field("timeframe.end").
			When(msg.Timeframe.End != nil && !msg.Timeframe.End.IsValid()).
			Message("Invalid timestamp")
	}
	return v.Error()
}

func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}
