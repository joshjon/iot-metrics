package device

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// EchoHandler is a REST based handler for the IoT Device Metrics API.
type EchoHandler struct {
	svc *Service
}

func NewEchoHandler(service *Service) *EchoHandler {
	return &EchoHandler{
		svc: service,
	}
}

func (h *EchoHandler) Register(g *echo.Group, middleware ...echo.MiddlewareFunc) {
	g.POST("/devices/:device_id/config", h.ConfigureDevice, middleware...)
	g.POST("/devices/:device_id/metrics", h.RecordMetric, middleware...)
	g.GET("/devices/:device_id/alerts", h.GetDeviceAlerts, middleware...)
}

type ConfigureDeviceRequest struct {
	DeviceID             string  `param:"device_id" json:"-"`
	TemperatureThreshold float64 `json:"temperature_threshold"`
	BatteryThreshold     int32   `json:"battery_threshold"`
}

func (h *EchoHandler) ConfigureDevice(c echo.Context) error {
	var req ConfigureDeviceRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.svc.ConfigureDevice(c.Request().Context(), req); err != nil {
		return err
	}
	return c.NoContent(http.StatusCreated)
}

type RecordMetricRequest struct {
	DeviceID    string    `param:"device_id" json:"-"`
	Temperature float64   `json:"temperature"`
	Battery     int32     `json:"battery"`
	Timestamp   time.Time `json:"timestamp"`
}

func (h *EchoHandler) RecordMetric(c echo.Context) error {
	var req RecordMetricRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.svc.RecordMetric(c.Request().Context(), req); err != nil {
		return err
	}
	return c.NoContent(http.StatusCreated)
}

type GetDeviceAlertsRequest struct {
	DeviceID       string     `param:"device_id" json:"-"`
	TimeframeStart *time.Time `query:"timeframe.start" json:"-"`
	TimeframeEnd   *time.Time `query:"timeframe.end" json:"-"`
	PageSize       int        `query:"page.size" json:"-"`
	PageToken      string     `query:"page.token" json:"-"`
}

type GetDeviceAlertsResponse struct {
	Alerts        []Alert `json:"alerts"`
	NextPageToken string  `json:"next_page_token,omitempty"`
}

func (h *EchoHandler) GetDeviceAlerts(c echo.Context) error {
	var req GetDeviceAlertsRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.GetDeviceAlerts(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}
