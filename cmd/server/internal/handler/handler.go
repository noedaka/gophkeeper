package handler

import (
	"context"
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/proto"
	"strconv"

	"google.golang.org/grpc/metadata"
)

type Handler struct {
	proto.UnimplementedAuthServiceServer
	proto.UnimplementedSecureStorageServer

	UserService   service.UserService
	RecordService service.RecordService
}

func NewHandler(userService service.UserService, recordService service.RecordService) *Handler {
	return &Handler{
		UserService:   userService,
		RecordService: recordService,
	}
}

func getUserIDFromContext(ctx context.Context) (int, bool) {
	if val := ctx.Value("user_id"); val != nil {
		if userIDStr, ok := val.(string); ok && userIDStr != "" {
			if userID, err := strconv.Atoi(userIDStr); err == nil {
				return userID, true
			}
		}
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, false
	}

	values := md.Get("user_id")
	if len(values) > 0 && values[0] != "" {
		if userID, err := strconv.Atoi(values[0]); err == nil {
			return userID, true
		}
	}

	return 0, false
}

const UserIDKey ContextKey = "user_id"

type ContextKey string
