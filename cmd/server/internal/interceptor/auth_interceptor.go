package interceptor

import (
	"context"
	"gophkeeper/internal/config"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Interceptor структура перехватчика
type Interceptor struct {
	cfg config.Config
}

// NewInterceptor создает новый Interceptor
func NewInterceptor(cfg config.Config) *Interceptor {
	return &Interceptor{
		cfg: cfg,
	}
}

// typed key для безопасного хранения userID в context
type UserIDKey struct{}

// AuthUnaryInterceptor unary interceptor для авторизации при обычных запросах
func (i *Interceptor) AuthUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	excludedMethods := map[string]bool{
		"/auth.gophkeeper.AuthService/RegisterUser": true,
		"/auth.gophkeeper.AuthService/AuthUser":     true,
	}

	if excludedMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	userID, err := i.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	newCtx := context.WithValue(ctx, UserIDKey{}, userID)
	return handler(newCtx, req)
}

// AuthStreamInterceptor tream interceptor для авторизации при потоковых запросах
func (i *Interceptor) AuthStreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	userID, err := i.authenticate(stream.Context())
	if err != nil {
		return err
	}

	newCtx := context.WithValue(stream.Context(), UserIDKey{}, userID)

	wrappedStream := &authServerStream{
		ServerStream: stream,
		ctx:          newCtx,
	}

	return handler(srv, wrappedStream)
}

type authServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authServerStream) Context() context.Context {
	return s.ctx
}

func (i *Interceptor) authenticate(ctx context.Context) (int, error) {
	tokenString, err := i.extractTokenFromContext(ctx)
	if err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	claims, err := i.validateJWT(tokenString)
	if err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok || userIDStr == "" {
		return 0, status.Errorf(codes.Unauthenticated, "token missing user_id claim")
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID == 0 {
		return 0, status.Errorf(codes.Unauthenticated, "invalid user_id in token")
	}

	return userID, nil
}

// extractTokenFromContext извлекает токен из заголовка Authorization
func (i *Interceptor) extractTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	authHeader := authHeaders[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format. Expected 'Bearer <token>'")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty token")
	}

	return token, nil
}

// validateJWT проверяет JWT-токен и возвращает claims
func (i *Interceptor) validateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, status.Errorf(codes.Unauthenticated, "unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(i.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, status.Error(codes.Unauthenticated, "invalid token")
}
