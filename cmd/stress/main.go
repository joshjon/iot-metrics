package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/joshjon/iot-metrics/log"
	iotv1 "github.com/joshjon/iot-metrics/proto/gen/iot/v1"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1/iotv1connect"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	cliApp := cli.NewApp()
	cliApp.Name = "stress"
	cliApp.Usage = "Stress test a gRPC server hosting the iot.v1.DeviceService API"

	cliApp.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "url",
			Aliases: []string{"u"},
			Value:   "http://localhost:8080",
			Usage:   "URL of gRPC server",
		},
		&cli.IntFlag{
			Name:    "devices",
			Aliases: []string{"d"},
			Value:   1000,
			Usage:   "Number of devices to simulate",
		},
		&cli.IntFlag{
			Name:    "interval",
			Aliases: []string{"i"},
			Value:   100,
			Usage:   "Specifies the interval in milliseconds per request",
		},
	}

	logger := log.NewLogger()

	cliApp.Commands = []*cli.Command{
		{
			Name:   "run",
			Usage:  "[default] runs the stress test",
			Action: run,
		},
	}

	cliApp.DefaultCommand = "run"

	if err := cliApp.RunContext(ctx, os.Args); err != nil {
		logger.Error("start stress test failed", "error", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	f := parseFlags(c)
	ctx := c.Context
	sessionID := strings.SplitN(uuid.New().String(), "-", 2)[0]
	logger := log.NewLogger(log.WithDevelopment())

	timeout := 5 * time.Second
	client := iotv1connect.NewDeviceServiceClient(&http.Client{Timeout: timeout}, f.serviceURL, connect.WithGRPC())

	tempThresh := 50.00
	batteryThresh := int32(50)
	getDeviceID := func(i int) string {
		return "session-" + sessionID + "-device-" + strconv.Itoa(i)
	}

	logger.Info(fmt.Sprintf("starting pool with %d workers", f.numDevices))
	pool := NewWorkerPool(f.numDevices, 5*time.Second)
	poolErrs := pool.Start(ctx)

	logger.Info(fmt.Sprintf("setting up configurations for %d devices", f.numDevices))
	for i := range f.numDevices {
		pool.QueueJob(func(ctx context.Context) error {
			deviceID := getDeviceID(i)
			_, err := client.ConfigureDevice(ctx, connect.NewRequest(&iotv1.ConfigureDeviceRequest{
				DeviceId:             deviceID,
				TemperatureThreshold: tempThresh,
				BatteryThreshold:     batteryThresh,
			}))
			if err != nil {
				return fmt.Errorf("configure device '%s': %v", deviceID, err)
			}
			logger.Info("configure device", "device_id", deviceID, "temperature_threshold", tempThresh, "battery_threshold", batteryThresh)
			return nil
		})
	}

	cfgsDoneCh := make(chan struct{})
	go func() {
		pool.Wait()
		close(cfgsDoneCh)
	}()

	select {
	case <-cfgsDoneCh:
	case err := <-poolErrs:
		if err != nil {
			return fmt.Errorf("setup devices: %w", err)
		}
	}

	logger.Info(fmt.Sprintf("simulating metrics for %d devices", f.numDevices))
	go func() {
		for {
			for i := range f.numDevices {
				if ctx.Err() != nil {
					return
				}
				pool.QueueJob(func(ctx context.Context) error {
					deviceID := getDeviceID(i)
					req := &iotv1.RecordMetricRequest{
						DeviceId:    deviceID,
						Timestamp:   timestamppb.New(time.Now().UTC()),
						Temperature: randFloat64(0, tempThresh*2),
						Battery:     randInt32(0, batteryThresh*2),
					}
					_, err := client.RecordMetric(ctx, connect.NewRequest(req))
					if err != nil {
						return fmt.Errorf("record device '%s' metric: %w", deviceID, err)
					}
					logger.Info("recorded device metric", "device_id", deviceID, "temperature", req.Temperature, "battery", req.Battery)
					return nil
				})
			}
			select {
			case <-time.After(time.Duration(f.intervalMS) * time.Millisecond):
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		<-ctx.Done()
		pool.Close()
	}()

	for err := range poolErrs {
		if !errors.Is(err, context.Canceled) {
			logger.Error("worker job failed", "error", err)
		}
	}

	return nil
}

type flags struct {
	serviceURL string
	numDevices int
	intervalMS int64
}

func parseFlags(c *cli.Context) flags {
	var parsed flags
	var flagErrs []string

	parsed.serviceURL = c.String("url")
	if parsed.serviceURL == "" {
		flagErrs = append(flagErrs, "url: cannot be empty")
	} else if _, err := url.ParseRequestURI(parsed.serviceURL); err != nil {
		flagErrs = append(flagErrs, "url: invalid format")
	}

	parsed.numDevices = c.Int("devices")
	if parsed.numDevices <= 0 {
		flagErrs = append(flagErrs, "devices: must be greater than zero")
	}

	parsed.intervalMS = c.Int64("interval")
	if parsed.intervalMS < 0 {
		flagErrs = append(flagErrs, "interval: must be non negative")
	}

	if len(flagErrs) > 0 {
		fmt.Fprintln(os.Stderr, "Flag errors:")
		for _, ferr := range flagErrs {
			fmt.Fprintln(os.Stderr, "  "+ferr)
		}
		fmt.Fprintln(os.Stdout)
		cli.ShowAppHelpAndExit(c, 1)
	}

	return parsed
}

func randFloat64(min, max float64) float64 {
	f := min + rand.Float64()*(max-min)
	scale := math.Pow(10, float64(2))
	return math.Trunc(f*scale) / scale
}

func randInt32(min, max int32) int32 {
	return min + rand.Int31n(max-min)
}
