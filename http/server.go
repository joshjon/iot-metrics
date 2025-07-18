package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	mux             *http.ServeMux
	echo            *echo.Echo
	httpSrv         *http.Server
	connectServices []string
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	e := echo.New()
	e.Use(middleware.Recover())
	return &Server{
		mux:  mux,
		echo: e,
		httpSrv: &http.Server{
			Addr:    addr,
			Handler: h2c.NewHandler(mux, &http2.Server{}),
		},
	}
}

type EchoHandler interface {
	Register(g *echo.Group, middleware ...echo.MiddlewareFunc)
}

func (s *Server) RegisterEcho(handler EchoHandler, middleware ...echo.MiddlewareFunc) {
	handler.Register(s.echo.Group(""), middleware...)
}

func (s *Server) RegisterConnect(path string, handler http.Handler) {
	s.mux.Handle(path, handler)
	s.connectServices = append(s.connectServices, path)
}

func (s *Server) Serve() error {
	var connectSvcNames []string
	for _, path := range s.connectServices {
		// Static reflector wants the fully-qualified service name without
		// leading/trailing slashes.
		connectSvcNames = append(connectSvcNames, strings.Trim(path, "/"))
	}

	// Register health handlers
	s.echo.GET("/healthz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	s.mux.Handle(grpchealth.NewHandler(grpchealth.NewStaticChecker(connectSvcNames...)))
	connectSvcNames = append(connectSvcNames, grpchealth.HealthV1ServiceName)

	// Register reflection
	reflector := grpcreflect.NewStaticReflector(connectSvcNames...)
	s.mux.Handle(grpcreflect.NewHandlerV1(reflector))
	s.mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
	s.mux.Handle("/", s.echo)

	return s.httpSrv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}
