package server

import (
	"gophkeeper/cmd/server/internal/handler"
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/proto"
	"net"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	service service.UserService
}

func NewGRPCServer(service service.UserService) *GRPCServer {
	return &GRPCServer{
		service: service,
	}
}

func (s *GRPCServer) StartServer() error {
	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
	//grpc.UnaryInterceptor(interceptor.AuthInterceptor),
	)

	handler := handler.NewHandler(s.service)

	proto.RegisterAuthServiceServer(grpcServer, handler)

	//TODO Graceful shutdown
	if err := grpcServer.Serve(listen); err != nil {
		return err
	}

	return nil
}
