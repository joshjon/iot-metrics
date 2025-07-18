package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v4"
	"github.com/urfave/cli/v2"
	"golang.org/x/time/rate"

	"github.com/joshjon/iot-metrics/config"
	"github.com/joshjon/iot-metrics/device"
	"github.com/joshjon/iot-metrics/http"
	"github.com/joshjon/iot-metrics/log"
	"github.com/joshjon/iot-metrics/proto/gen/iot/v1/iotv1connect"
	"github.com/joshjon/iot-metrics/rlimit"
	"github.com/joshjon/iot-metrics/sqlite"
	"github.com/joshjon/iot-metrics/sqlite/migrations"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	cliApp := cli.NewApp()
	cliApp.Name = "iot-metrics"
	cliApp.Usage = "Collect metrics from simulated IoT devices and trigger alerts when thresholds are breached"

	cliApp.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "config-file",
			Aliases: []string{"c"},
			Value:   "",
			Usage:   "path to yaml config file (required if not using environment variables)",
		},
	}

	cliApp.Commands = []*cli.Command{
		{
			Name:   "run",
			Usage:  "[default] runs the service",
			Action: run,
		},
	}

	cliApp.DefaultCommand = "run"

	if err := cliApp.RunContext(ctx, os.Args); err != nil {
		log.NewLogger().Error("failed to run service", "error", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	ctx := c.Context

	configFile := c.String("config-file")
	cfg, err := config.Load(configFile) // falls back to env var if config file is empty
	if err != nil {
		return err
	}

	var loggerOpts []log.LoggerOption
	if !cfg.Logger.Structured {
		loggerOpts = append(loggerOpts, log.WithDevelopment())
	}
	logger := log.NewLogger(loggerOpts...)

	db, err := sqlite.Open(ctx, sqlite.WithDir(cfg.SQLiteDir))
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	logger.Info("opened sqlite database connection")

	if err = sqlite.Migrate(db, migrations.FS()); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}
	logger.Info("migrated sqlite database")
	repo := sqlite.NewDeviceRepository(db)

	middleware := []echo.MiddlewareFunc{http.NewEchoErrorMiddleware(), http.NewEchoLogMiddleware(logger)}
	interceptors := []connect.Interceptor{http.NewConnectErrorInterceptor(), http.NewConnectLogInterceptor(logger)}

	if cfg.DeviceRateLimit != nil {
		rl := cfg.DeviceRateLimit
		limit := rate.Every(time.Duration(rl.Tokens / rl.Seconds))
		burst := min(rl.Tokens, 50)
		rateLimiter := rlimit.NewRateLimiter(limit, burst, 5*time.Minute, time.Minute)
		middleware = append(middleware, http.NewEchoRateLimiterMiddleware(rateLimiter, device.EchoRequestDeviceIDGetter))
		interceptors = append(interceptors, http.NewConnectRateLimitInterceptor(rateLimiter, device.ConnectRequestDeviceIDGetter))
		logger.Info("device rate limiter enabled")
	}

	svc := device.NewService(repo, logger)

	hostPort := ":" + strconv.Itoa(cfg.Port)
	srv := http.NewServer(hostPort)

	restHandler := device.NewEchoHandler(svc)
	srv.RegisterEcho(restHandler, middleware...)

	rpcHandler := device.NewConnectHandler(svc)
	srv.RegisterConnect(iotv1connect.NewDeviceServiceHandler(rpcHandler,
		http.WithConnectRecover(logger),
		connect.WithInterceptors(interceptors...),
	))

	errs := make(chan error)

	go func() {
		defer close(errs)
		if err := srv.Serve(); err != nil {
			errs <- fmt.Errorf("start server: %w", err)
		}
	}()

	logger.Info("server listening", "port", cfg.Port)

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			logger.Error("failed to stop server", "error", err)
			return
		}
		logger.Info("server stopped")
	}()

	select {
	case err = <-errs:
		return err
	case <-ctx.Done():
		return nil
	}
}
