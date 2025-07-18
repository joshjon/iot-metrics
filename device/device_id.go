package device

import (
	"connectrpc.com/connect"
	"github.com/labstack/echo/v4"
)

// EchoRequestDeviceIDGetter gets the device ID from a REST based request for
// device rate limiting middleware.
func EchoRequestDeviceIDGetter(c echo.Context) (string, bool) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		return "", false
	}
	return deviceID, true
}

// ConnectRequestDeviceIDGetter gets the device ID from a REST based request for
// device rate limiting middleware.
func ConnectRequestDeviceIDGetter(req connect.AnyRequest) (string, bool) {
	if dr, ok := req.Any().(interface{ GetDeviceId() string }); ok {
		return dr.GetDeviceId(), true
	}
	return "", false
}
