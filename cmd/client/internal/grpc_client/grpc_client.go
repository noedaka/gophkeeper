package grpcclient

import (
	"context"
	"fmt"
	pb "gophkeeper/internal/proto"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GophKeeperClient обёртка над gRPC клиентом
type GophKeeperClient struct {
	conn          *grpc.ClientConn
	authClient    pb.AuthServiceClient
	storageClient pb.SecureStorageClient
	token         string
	mu            sync.RWMutex // для безопасного доступа к токену
}

// NewGophKeeperClient создаёт нового клиента
func NewGophKeeperClient(serverAddr string) (*GophKeeperClient, error) {
	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	authClient := pb.NewAuthServiceClient(conn)
	cardClient := pb.NewSecureStorageClient(conn)

	return &GophKeeperClient{
		conn:          conn,
		authClient:    authClient,
		storageClient: cardClient,
		token:         "",
	}, nil
}

// SetToken устанавливает токен аутентификации
func (c *GophKeeperClient) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// GetToken возвращает текущий токен
func (c *GophKeeperClient) GetToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// ClearToken очищает токен аутентификации
func (c *GophKeeperClient) ClearToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
}

// Register регистрирует нового пользователя
func (c *GophKeeperClient) Register(ctx context.Context, login, password string) (string, error) {
	var req pb.RegisterRequest
	req.SetLogin(login)
	req.SetPassword(password)

	resp, err := c.authClient.RegisterUser(ctx, &req)
	if err != nil {
		return "", err
	}

	token := resp.GetToken()
	if token != "" {
		c.SetToken(token)
	}

	return token, nil
}

// Login аутентифицирует пользователя
func (c *GophKeeperClient) Login(ctx context.Context, login, password string) (string, error) {
	var req pb.AuthRequest
	req.SetLogin(login)
	req.SetPassword(password)

	resp, err := c.authClient.AuthUser(ctx, &req)
	if err != nil {
		return "", err
	}

	token := resp.GetToken()
	if token != "" {
		c.SetToken(token)
	}

	return token, nil
}

// StoreCard сохраняет новую карту
func (c *GophKeeperClient) StoreCard(ctx context.Context, card *pb.Card) (int32, error) {
	// Используем токен из клиента
	token := c.GetToken()
	if token == "" {
		return 0, fmt.Errorf("токен аутентификации не установлен")
	}

	// Создаем контекст с токеном
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	resp, err := c.storageClient.StoreCard(ctx, card)
	if err != nil {
		return 0, err
	}
	return resp.GetId(), nil
}

// GetCard получает карту по ID
func (c *GophKeeperClient) GetCard(ctx context.Context, cardID int32) (*pb.Card, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен аутентификации не установлен")
	}

	var req pb.RecordID
	req.SetId(cardID)

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	return c.storageClient.GetCard(ctx, &req)
}

// ListCards получает список ID карт пользователя
func (c *GophKeeperClient) ListCards(ctx context.Context) ([]int32, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен аутентификации не установлен")
	}

	var empty pb.Empty
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	resp, err := c.storageClient.ListCards(ctx, &empty)
	if err != nil {
		return nil, err
	}

	var ids []int32
	for _, record := range resp.GetRecords() {
		ids = append(ids, record.GetId())
	}
	return ids, nil
}

// DeleteCard удаляет карту
func (c *GophKeeperClient) DeleteCard(ctx context.Context, cardID int32) error {
	token := c.GetToken()
	if token == "" {
		return fmt.Errorf("токен аутентификации не установлен")
	}

	var req pb.RecordID
	req.SetId(cardID)

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	_, err := c.storageClient.DeleteCard(ctx, &req)
	return err
}

// Close закрывает соединение
func (c *GophKeeperClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *GophKeeperClient) StoreCredentials(ctx context.Context, creds *pb.Credentials) (int32, error) {
	token := c.GetToken()
	if token == "" {
		return 0, fmt.Errorf("токен аутентификации не установлен")
	}
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	resp, err := c.storageClient.StoreCredentials(ctx, creds)
	if err != nil {
		return 0, err
	}
	return resp.GetId(), nil
}

func (c *GophKeeperClient) GetCredentials(ctx context.Context, credsID int32) (*pb.Credentials, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен аутентификации не установлен")
	}
	var req pb.RecordID
	req.SetId(credsID)
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	return c.storageClient.GetCredentials(ctx, &req)
}

func (c *GophKeeperClient) ListCredentials(ctx context.Context) ([]int32, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен аутентификации не установлен")
	}
	var empty pb.Empty
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	resp, err := c.storageClient.ListCredentials(ctx, &empty)
	if err != nil {
		return nil, err
	}
	var ids []int32
	for _, record := range resp.GetRecords() {
		ids = append(ids, record.GetId())
	}
	return ids, nil
}

func (c *GophKeeperClient) DeleteCredentials(ctx context.Context, credsID int32) error {
	token := c.GetToken()
	if token == "" {
		return fmt.Errorf("токен аутентификации не установлен")
	}
	var req pb.RecordID
	req.SetId(credsID)
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	_, err := c.storageClient.DeleteCredentials(ctx, &req)
	return err
}
