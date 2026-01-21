package grpcclient

import (
	"context"
	"fmt"
	"gophkeeper/internal/proto"
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
	binaryClient  pb.BinaryStorageClient
	token         string
	mu            sync.RWMutex
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
	binaryClient := pb.NewBinaryStorageClient(conn)

	return &GophKeeperClient{
		conn:          conn,
		authClient:    authClient,
		storageClient: cardClient,
		binaryClient:  binaryClient,
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

// StoreRecord сохраняет запись на сервер
func (c *GophKeeperClient) StoreRecord(ctx context.Context, record *pb.EncryptedRecord) (int32, error) {
	token := c.GetToken()
	if token == "" {
		return 0, fmt.Errorf("токен аутентификации не установлен")
	}
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))

	resp, err := c.storageClient.StoreRecord(ctx, record)
	if err != nil {
		return 0, err
	}
	return resp.GetId(), nil
}

// GetRecord получает запись с сервера по его ID
func (c *GophKeeperClient) GetRecord(ctx context.Context, recordID int32) (*pb.EncryptedRecord, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен аутентификации не установлен")
	}

	req := &pb.RecordID_builder{Id: &recordID}
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))

	return c.storageClient.GetRecord(ctx, req.Build())
}

// ListRecords получает список всех записей по ID, сделанные определенным пользователем
func (c *GophKeeperClient) ListRecords(ctx context.Context, recordType string) ([]*pb.RecordInfo, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен аутентификации не установлен")
	}
	req := &pb.ListRequest_builder{RecordType: &recordType}
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))

	resp, err := c.storageClient.ListRecords(ctx, req.Build())
	if err != nil {
		return nil, err
	}
	return resp.GetRecords(), nil
}

// DeleteRecord удаляет запись по ID
func (c *GophKeeperClient) DeleteRecord(ctx context.Context, recordID int32) error {
	token := c.GetToken()
	if token == "" {
		return fmt.Errorf("токен аутентификации не установлен")
	}

	req := &pb.RecordID_builder{Id: &recordID}

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))

	_, err := c.storageClient.DeleteRecord(ctx, req.Build())

	return err
}

// UploadBinary потоково загружает бинарный файл на сервер
func (c *GophKeeperClient) UploadBinary(ctx context.Context) (proto.BinaryStorage_UploadBinaryClient, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен не установлен")
	}
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	return c.binaryClient.UploadBinary(ctx)
}

// DownloadBinary потково выгружает файл с сервера
func (c *GophKeeperClient) DownloadBinary(ctx context.Context, recordID int32) (proto.BinaryStorage_DownloadBinaryClient, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен не установлен")
	}
	req := pb.BinaryRecordID_builder{Id: &recordID}.Build()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	return c.binaryClient.DownloadBinary(ctx, req)
}

// ListBinaries получает список бинарных файлов
func (c *GophKeeperClient) ListBinaries(ctx context.Context) ([]*pb.BinaryRecordInfo, error) {
	token := c.GetToken()
	if token == "" {
		return nil, fmt.Errorf("токен не установлен")
	}

	req := pb.BinaryEmpty_builder{}.Build()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))

	resp, err := c.binaryClient.ListBinaries(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetRecords(), nil
}

// DeleteBinary удаляет бинарный файл
func (c *GophKeeperClient) DeleteBinary(ctx context.Context, recordID int32) error {
	token := c.GetToken()
	if token == "" {
		return fmt.Errorf("токен не установлен")
	}

	req := pb.BinaryRecordID_builder{Id: &recordID}.Build()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	_, err := c.binaryClient.DeleteBinary(ctx, req)

	return err
}
