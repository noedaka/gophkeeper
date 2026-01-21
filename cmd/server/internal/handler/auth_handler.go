package handler

import (
	"context"
	"fmt"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) RegisterUser(ctx context.Context, r *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	user := &domain.UserCredentials{
		Login:    r.GetLogin(),
		Password: r.GetPassword(),
	}

	//TODO Валидация ошибок
	userID, err := h.UserService.Register(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot register user: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": fmt.Sprintf("%v", userID),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot create JWT Token: %v", err)

	}

	response := proto.RegisterResponse_builder{
		Token: &tokenString,
	}

	return response.Build(), nil
}

func (h *Handler) AuthUser(ctx context.Context, r *proto.AuthRequest) (*proto.AuthResponse, error) {
	user := &domain.UserCredentials{
		Login:    r.GetLogin(),
		Password: r.GetPassword(),
	}

	//TODO Валидация ошибок
	userID, err := h.UserService.Login(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot login user: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": fmt.Sprintf("%v", userID),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot create JWT Token: %v", err)

	}

	response := proto.AuthResponse_builder{
		Token: &tokenString,
	}

	return response.Build(), nil
}
