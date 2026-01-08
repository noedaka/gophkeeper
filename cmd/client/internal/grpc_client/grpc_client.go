package grpcclient

import (
	"context"

	pb "gophkeeper/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GophKeeperClient обёртка над gRPC клиентом
type GophKeeperClient struct {
	conn   *grpc.ClientConn
	client pb.AuthServiceClient
}

// NewGophKeeperClient создаёт нового клиента
func NewGophKeeperClient(serverAddr string) (*GophKeeperClient, error) {
	// Безопасное соединение только для тестирования
	// В production нужно использовать TLS
	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewAuthServiceClient(conn)

	return &GophKeeperClient{
		conn:   conn,
		client: client,
	}, nil
}

// Register регистрирует нового пользователя
func (c *GophKeeperClient) Register(ctx context.Context, login, password string) (string, error) {
	var req pb.RegisterRequest

	req.SetLogin(login)
	req.SetPassword(password)
	resp, err := c.client.RegisterUser(ctx, &req)

	if err != nil {
		return "", err
	}

	return resp.GetToken(), nil
}

// Login аутентифицирует пользователя
func (c *GophKeeperClient) Login(ctx context.Context, login, password string) (string, error) {
	var req pb.AuthRequest

	req.SetLogin(login)
	req.SetPassword(password)
	resp, err := c.client.AuthUser(ctx, &req)

	if err != nil {
		return "", err
	}

	return resp.GetToken(), nil
}

// Close закрывает соединение
func (c *GophKeeperClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
