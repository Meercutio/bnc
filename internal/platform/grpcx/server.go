package grpcx

import (
	"net"
	"time"

	"google.golang.org/grpc"
)

type Server struct {
	grpc *grpc.Server
	lis  net.Listener
}

func New(addr string, opts ...grpc.ServerOption) (*Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer(append(opts, grpc.ConnectionTimeout(5*time.Second))...)
	return &Server{grpc: s, lis: lis}, nil
}

func (s *Server) GRPC() *grpc.Server { return s.grpc }

func (s *Server) Serve() error { return s.grpc.Serve(s.lis) }

func (s *Server) Stop() { s.grpc.GracefulStop() }
