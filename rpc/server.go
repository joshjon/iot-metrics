package rpc

import (
	"context"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/grpcreflect"
	"connectrpc.com/vanguard"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/joshjon/iot-metrics/proto/gen/health/v1/healthv1connect"
)

type Server struct {
	mux          *http.ServeMux
	httpSrv      *http.Server
	vanSvcs      []*vanguard.Service
	servicePaths []string
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	return &Server{
		mux: mux,
		httpSrv: &http.Server{
			Addr:    addr,
			Handler: h2c.NewHandler(mux, &http2.Server{}),
		},
	}
}

func (s *Server) Register(path string, handler http.Handler) {
	s.mux.Handle(path, handler)
	s.servicePaths = append(s.servicePaths, path)

	// Vanguard service for REST transcoding
	s.vanSvcs = append(s.vanSvcs, vanguard.NewService(path, handler))
}

func (s *Server) Serve() error {
	// Register health service
	s.Register(healthv1connect.NewHealthServiceHandler(&healthService{}))

	var svcNames []string
	for _, path := range s.servicePaths {
		// Static reflector wants the fully-qualified service name without
		// leading/trailing slashes.
		svcNames = append(svcNames, strings.Trim(path, "/"))
	}
	reflector := grpcreflect.NewStaticReflector(svcNames...)
	s.mux.Handle(grpcreflect.NewHandlerV1(reflector))
	s.mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	// REST to RPC transcoder
	transcoder, err := vanguard.NewTranscoder(s.vanSvcs)
	if err != nil {
		return err
	}
	s.mux.Handle("/", transcoder)

	return s.httpSrv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}
