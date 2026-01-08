package handler

import (
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/proto"
)

type Handler struct {
	proto.UnimplementedAuthServiceServer
	UserService service.UserService
}

func NewHandler(userService service.UserService) *Handler {
	return &Handler{UserService: userService}
}
