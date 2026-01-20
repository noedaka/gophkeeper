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
		cfg: cfg,
	}
}

func getUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(interceptor.UserIDKey{}).(int)
	if !ok || userID == 0 {
		return 0, false
	}
	return userID, true
}

const UserIDKey ContextKey = "user_id"

type ContextKey string
