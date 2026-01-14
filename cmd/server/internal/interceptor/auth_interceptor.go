package interceptor

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var JWTSecret = []byte("my-super-secret-key-for-testing")

// AuthInterceptor перехватчик для проверки JWT-токенов
func AuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	excludedMethods := map[string]bool{
        "/auth.gophkeeper.AuthService/RegisterUser": true,
        "/auth.gophkeeper.AuthService/AuthUser":     true,
    }

	if excludedMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	tokenString, err := extractTokenFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	claims, err := validateJWT(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return nil, status.Errorf(codes.Unauthenticated, "token missing user_id claim")
	}

	ctx = context.WithValue(ctx, "userID", userID)

	md, _ := metadata.FromIncomingContext(ctx)
	md = md.Copy()
	md.Set("user_id", userID)
	ctx = metadata.NewIncomingContext(ctx, md)

	return handler(ctx, req)
}

// extractTokenFromContext извлекает токен из заголовка Authorization
func extractTokenFromContext(ctx context.Context) (string, error) {
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
func validateJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, status.Errorf(codes.Unauthenticated, "unexpected signing method: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, status.Error(codes.Unauthenticated, "invalid token")
}
