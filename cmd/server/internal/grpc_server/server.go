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
	userService   service.UserService
	recordService service.RecordService
	cfg           config.Config
}

func NewGRPCServer(userService service.UserService, recordService service.RecordService, cfg config.Config) *GRPCServer {
	return &GRPCServer{
		userService:   userService,
		recordService: recordService,
		cfg:           cfg,
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

	handler := handler.NewHandler(s.userService, s.recordService)

	proto.RegisterAuthServiceServer(grpcServer, handler)
	proto.RegisterSecureStorageServer(grpcServer, handler)

	//TODO Graceful shutdown
	if err := grpcServer.Serve(listen); err != nil {
		return err
	}

	return nil
}
