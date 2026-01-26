package handler

import (
	"context"
	"errors"
	"time"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/model"
	"gophkeeper/internal/proto"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RegisterUser хэндлер для регистрации пользователя
func (h *Handler) RegisterUser(ctx context.Context, r *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	user := &domain.UserCredentials{
		Login:    r.GetLogin(),
		Password: r.GetPassword(),
	}

	userID, err := h.UserService.Register(ctx, user)
	if err != nil {
		if errors.Is(err, model.ErrOccupiedLogin) {
			return nil, status.Errorf(codes.AlreadyExists, "login is already occupied")
		}
		return nil, status.Errorf(codes.Internal, "cannot register user: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create JWT token: %v", err)
	}

	response := proto.RegisterResponse_builder{
		Token: &tokenString,
	}
	return response.Build(), nil
}

// AuthUser хэндлер для авторизации пользователя
func (h *Handler) AuthUser(ctx context.Context, r *proto.AuthRequest) (*proto.AuthResponse, error) {
	user := &domain.UserCredentials{
		Login:    r.GetLogin(),
		Password: r.GetPassword(),
	}

	userID, err := h.UserService.Login(ctx, user)
	if err != nil {
		if errors.Is(err, model.ErrNoUser) || errors.Is(err, model.ErrIncorrectPass) {
			return nil, status.Errorf(codes.Unauthenticated, "invalid login or password")
		}
		return nil, status.Errorf(codes.Internal, "cannot authenticate user: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create JWT token: %v", err)
	}

	response := proto.AuthResponse_builder{
		Token: &tokenString,
	}
	return response.Build(), nil
}