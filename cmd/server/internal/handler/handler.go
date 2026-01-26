package handler

import (
	"context"
	"gophkeeper/cmd/server/internal/interceptor"
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/config"
	"gophkeeper/internal/proto"
)

type Handler struct {
	proto.UnimplementedAuthServiceServer
	proto.UnimplementedSecureStorageServer
	proto.UnimplementedBinaryStorageServer

	UserService   service.UserService
	RecordService service.RecordService
	BinaryService service.BinaryService

	cfg *config.Config
}

func NewHandler(userService service.UserService, recordService service.RecordService, binaryService service.BinaryService, cfg *config.Config) *Handler {
	return &Handler{
		UserService:   userService,
		RecordService: recordService,
		BinaryService: binaryService,
		cfg:           cfg,
	}
}

func getUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(interceptor.UserIDKey).(string)
	if !ok || userID == "" {
		return "", false
	}
	return userID, true
}
