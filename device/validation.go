package device

import (
	"strings"

	"github.com/joshjon/iot-metrics/http"
)

func validateConfigureDeviceReq(req ConfigureDeviceRequest) error {
	v := http.NewRequestValidator()
	v.Field("device_id").When(isBlank(req.DeviceID)).Message("Must not be blank")
	v.Field("temperature_threshold").
		When(req.TemperatureThreshold < minTemperature || req.TemperatureThreshold > maxTemperature).
		Messagef("Must be between %.2f and %.2f", minTemperature, maxTemperature)
	v.Field("battery_threshold").
		When(req.BatteryThreshold < minBattery || req.BatteryThreshold > maxBattery).
		Messagef("Must be between %d and %d", minBattery, maxBattery)
	return v.Error()
}

func validateRecordMetricReq(req RecordMetricRequest) error {
	v := http.NewRequestValidator()
	v.Field("device_id").When(isBlank(req.DeviceID)).Message("Must not be blank")
	v.Field("timestamp").When(req.Timestamp.IsZero()).Message("Must not be empty")
	v.Field("temperature").
		When(req.Temperature < minTemperature || req.Temperature > maxTemperature).
		Messagef("Must be between %.2f and %.2f", minTemperature, maxTemperature)
	v.Field("battery").
		When(req.Battery < minBattery || req.Battery > maxBattery).
		Messagef("Must be between %d and %d", minBattery, maxBattery)
	return v.Error()
}

func validateGetDeviceAlertsReq(req GetDeviceAlertsRequest) error {
	v := http.NewRequestValidator()
	v.Field("device_id").When(isBlank(req.DeviceID)).Message("Must not be blank")
	if req.TimeframeStart != nil {
		v.Field("timeframe.start").When(req.TimeframeStart.IsZero()).Message("Must not be empty")
		if req.TimeframeEnd != nil {
			v.Field("timeframe.start").
				When(req.TimeframeStart.After(*req.TimeframeEnd)).
				Message("Must be before timeframe.end")
		}
	}
	if req.TimeframeEnd != nil {
		v.Field("timeframe.end").When(req.TimeframeEnd.IsZero()).Message("Must not be empty")
	}
	return v.Error()
}

func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}
