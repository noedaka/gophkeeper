package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gophkeeper/cmd/server/internal/handler"
	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) Register(ctx context.Context, creds *domain.UserCredentials) (int, error) {
	args := m.Called(ctx, creds)
	return args.Int(0), args.Error(1)
}

func (m *mockUserService) Login(ctx context.Context, creds *domain.UserCredentials) (int, error) {
	args := m.Called(ctx, creds)
	return args.Int(0), args.Error(1)
}

func TestHandler_RegisterUser(t *testing.T) {
	ctx := context.Background()
	secret := "testsecret"
	cfg := &config.Config{JWTSecret: secret}

	tests := []struct {
		name        string
		login       string
		password    string
		mockSetup   func(m *mockUserService)
		wantUserID  int
		wantErr     bool
		wantErrCode codes.Code
		checkToken  bool
	}{
		{
			name:     "success",
			login:    "testuser",
			password: "testpass",
			mockSetup: func(m *mockUserService) {
				m.On("Register", mock.Anything, mock.MatchedBy(func(creds *domain.UserCredentials) bool {
					return creds.Login == "testuser" && creds.Password == "testpass"
				})).Return(42, nil)
			},
			wantUserID: 42,
			wantErr:    false,
			checkToken: true,
		},
		{
			name:     "service error",
			login:    "testuser",
			password: "testpass",
			mockSetup: func(m *mockUserService) {
				m.On("Register", mock.Anything, mock.MatchedBy(func(creds *domain.UserCredentials) bool {
					return creds.Login == "testuser" && creds.Password == "testpass"
				})).Return(0, errors.New("db error"))
			},
			wantErr:     true,
			wantErrCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockUserService{}
			tt.mockSetup(mockSvc)

			h := handler.NewHandler(mockSvc, nil, nil, cfg)

			login := tt.login
			password := tt.password
			req := proto.RegisterRequest_builder{
				Login:    &login,
				Password: &password,
			}.Build()

			resp, err := h.RegisterUser(ctx, req)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				tokenStr := resp.GetToken()
				assert.NotEmpty(t, tokenStr)

				if tt.checkToken {
					token, parseErr := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
						return []byte(secret), nil
					})
					assert.NoError(t, parseErr)
					assert.True(t, token.Valid)

					claims, ok := token.Claims.(jwt.MapClaims)
					assert.True(t, ok)
					assert.Equal(t, "42", claims["user_id"])
					exp, ok := claims["exp"].(float64)
					assert.True(t, ok)
					expectedExp := float64(time.Now().Add(24 * time.Hour).Unix())
					assert.GreaterOrEqual(t, exp, expectedExp-10)
					assert.LessOrEqual(t, exp, expectedExp+10)
				}
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_AuthUser(t *testing.T) {
	ctx := context.Background()
	secret := "testsecret"
	cfg := &config.Config{JWTSecret: secret}

	tests := []struct {
		name        string
		login       string
		password    string
		mockSetup   func(m *mockUserService)
		wantUserID  int
		wantErr     bool
		wantErrCode codes.Code
		checkToken  bool
	}{
		{
			name:     "success",
			login:    "testuser",
			password: "testpass",
			mockSetup: func(m *mockUserService) {
				m.On("Login", mock.Anything, mock.MatchedBy(func(creds *domain.UserCredentials) bool {
					return creds.Login == "testuser" && creds.Password == "testpass"
				})).Return(42, nil)
			},
			wantUserID: 42,
			wantErr:    false,
			checkToken: true,
		},
		{
			name:     "service error",
			login:    "testuser",
			password: "wrongpass",
			mockSetup: func(m *mockUserService) {
				m.On("Login", mock.Anything, mock.MatchedBy(func(creds *domain.UserCredentials) bool {
					return creds.Login == "testuser" && creds.Password == "wrongpass"
				})).Return(0, errors.New("invalid credentials"))
			},
			wantErr:     true,
			wantErrCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockUserService{}
			tt.mockSetup(mockSvc)

			h := handler.NewHandler(mockSvc, nil, nil, cfg)

			login := tt.login
			password := tt.password
			req := proto.AuthRequest_builder{
				Login:    &login,
				Password: &password,
			}.Build()

			resp, err := h.AuthUser(ctx, req)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				tokenStr := resp.GetToken()
				assert.NotEmpty(t, tokenStr)

				if tt.checkToken {
					token, parseErr := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
						return []byte(secret), nil
					})
					assert.NoError(t, parseErr)
					assert.True(t, token.Valid)

					claims, ok := token.Claims.(jwt.MapClaims)
					assert.True(t, ok)
					assert.Equal(t, "42", claims["user_id"])
				}
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
