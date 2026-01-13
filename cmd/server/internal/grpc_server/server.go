package server

import (
	"gophkeeper/cmd/server/internal/handler"
	"gophkeeper/cmd/server/internal/interceptor"
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/config"
	"gophkeeper/internal/proto"
	"net"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	service service.UserService
	cfg     config.Config
}

func NewGRPCServer(service service.UserService, cfg config.Config) *GRPCServer {
	return &GRPCServer{
		service: service,
		cfg:     cfg,
	}
}

func (s *GRPCServer) StartServer() error {
	listen, err := net.Listen("tcp", s.cfg.ServerPort)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.AuthInterceptor),
	)

	handler := handler.NewHandler(s.service)

	proto.RegisterAuthServiceServer(grpcServer, handler)

	//TODO Graceful shutdown
	if err := grpcServer.Serve(listen); err != nil {
		return err
	}

	return nil
}
