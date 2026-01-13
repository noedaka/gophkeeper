package handler

import (
	"context"
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/proto"
)

type Handler struct {
	proto.UnimplementedAuthServiceServer

	UserService  service.UserService
	CardService  service.CardService
	CredsService service.CredsService
}

func NewHandler(userService service.UserService) *Handler {
	return &Handler{UserService: userService}
}

func getUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}
