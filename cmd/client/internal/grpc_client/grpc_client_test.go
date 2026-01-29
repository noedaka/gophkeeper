package grpcclient

import (
	"context"
	"fmt"
	"sync"
	"testing"

	pb "gophkeeper/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAuthServiceClient struct {
	mock.Mock
}

func (m *mockAuthServiceClient) RegisterUser(ctx context.Context, req *pb.RegisterRequest, opts ...grpc.CallOption) (*pb.RegisterResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.RegisterResponse), args.Error(1)
}

func (m *mockAuthServiceClient) AuthUser(ctx context.Context, req *pb.AuthRequest, opts ...grpc.CallOption) (*pb.AuthResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.AuthResponse), args.Error(1)
}

type mockSecureStorageClient struct {
	mock.Mock
}

func (m *mockSecureStorageClient) StoreRecord(ctx context.Context, record *pb.EncryptedRecord, opts ...grpc.CallOption) (*pb.RecordID, error) {
	args := m.Called(ctx, record)
	return args.Get(0).(*pb.RecordID), args.Error(1)
}

func (m *mockSecureStorageClient) GetRecord(ctx context.Context, req *pb.RecordID, opts ...grpc.CallOption) (*pb.EncryptedRecord, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.EncryptedRecord), args.Error(1)
}

func (m *mockSecureStorageClient) ListRecords(ctx context.Context, req *pb.ListRequest, opts ...grpc.CallOption) (*pb.RecordList, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.RecordList), args.Error(1)
}

func (m *mockSecureStorageClient) DeleteRecord(ctx context.Context, req *pb.RecordID, opts ...grpc.CallOption) (*pb.Empty, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.Empty), args.Error(1)
}

type mockBinaryStorageClient struct {
	mock.Mock
}

func (m *mockBinaryStorageClient) UploadBinary(ctx context.Context, opts ...grpc.CallOption) (pb.BinaryStorage_UploadBinaryClient, error) {
	args := m.Called(ctx)
	return args.Get(0).(pb.BinaryStorage_UploadBinaryClient), args.Error(1)
}

func (m *mockBinaryStorageClient) DownloadBinary(ctx context.Context, req *pb.BinaryRecordID, opts ...grpc.CallOption) (pb.BinaryStorage_DownloadBinaryClient, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(pb.BinaryStorage_DownloadBinaryClient), args.Error(1)
}

func (m *mockBinaryStorageClient) ListBinaries(ctx context.Context, req *pb.BinaryEmpty, opts ...grpc.CallOption) (*pb.BinaryRecordList, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.BinaryRecordList), args.Error(1)
}

func (m *mockBinaryStorageClient) DeleteBinary(ctx context.Context, req *pb.BinaryRecordID, opts ...grpc.CallOption) (*pb.BinaryEmpty, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*pb.BinaryEmpty), args.Error(1)
}

type mockUploadStream struct {
	mock.Mock
}

func (m *mockUploadStream) Send(chunk *pb.BinaryChunk) error {
	args := m.Called(chunk)
	return args.Error(0)
}

func (m *mockUploadStream) CloseAndRecv() (*pb.BinaryRecordID, error) {
	args := m.Called()
	return args.Get(0).(*pb.BinaryRecordID), args.Error(1)
}

func (m *mockUploadStream) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (m *mockUploadStream) Trailer() metadata.MD {
	return metadata.MD{}
}

func (m *mockUploadStream) CloseSend() error {
	return nil
}

func (m *mockUploadStream) Context() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *mockUploadStream) SendMsg(any interface{}) error {
	return nil
}

func (m *mockUploadStream) RecvMsg(any interface{}) error {
	return nil
}

type mockDownloadStream struct {
	mock.Mock
}

func (m *mockDownloadStream) Recv() (*pb.BinaryChunk, error) {
	args := m.Called()
	return args.Get(0).(*pb.BinaryChunk), args.Error(1)
}

func (m *mockDownloadStream) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (m *mockDownloadStream) Trailer() metadata.MD {
	return metadata.MD{}
}

func (m *mockDownloadStream) CloseSend() error {
	return nil
}

func (m *mockDownloadStream) Context() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *mockDownloadStream) SendMsg(any interface{}) error {
	return nil
}

func (m *mockDownloadStream) RecvMsg(any interface{}) error {
	return nil
}

func TestTokenMethods(t *testing.T) {
	c := &GophKeeperClient{}

	c.SetToken("test-token")
	assert.Equal(t, "test-token", c.GetToken())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				c.SetToken(fmt.Sprintf("token-%d", i))
			}
			_ = c.GetToken()
		}(i)
	}
	wg.Wait()

	c.ClearToken()
	assert.Equal(t, "", c.GetToken())
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name         string
		respToken    string
		respErr      error
		wantToken    string
		wantReturned string
		wantErr      bool
	}{
		{
			name:         "success with token",
			respToken:    "jwt-token",
			wantToken:    "jwt-token",
			wantReturned: "jwt-token",
		},
		{
			name:         "success empty token",
			respToken:    "",
			wantToken:    "",
			wantReturned: "",
		},
		{
			name:    "server error",
			respErr: status.Error(codes.Internal, "server error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := &mockAuthServiceClient{}
			c := &GophKeeperClient{
				authClient: mockAuth,
			}

			resp := &pb.RegisterResponse{}
			if tt.respToken != "" {
				resp.SetToken(tt.respToken)
			}

			mockAuth.On("RegisterUser", mock.Anything, mock.MatchedBy(func(r *pb.RegisterRequest) bool {
				return r.GetLogin() == "user" && r.GetPassword() == "pass"
			})).Return(resp, tt.respErr).Once()

			token, err := c.Register(context.Background(), "user", "pass")

			assert.Equal(t, tt.wantReturned, token)
			assert.Equal(t, tt.wantToken, c.GetToken())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name         string
		respToken    string
		respErr      error
		wantToken    string
		wantReturned string
		wantErr      bool
	}{
		{
			name:         "success with token",
			respToken:    "jwt-token",
			wantToken:    "jwt-token",
			wantReturned: "jwt-token",
		},
		{
			name:    "server error",
			respErr: status.Error(codes.Unauthenticated, "bad credentials"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := &mockAuthServiceClient{}
			c := &GophKeeperClient{
				authClient: mockAuth,
			}

			resp := &pb.AuthResponse{}
			if tt.respToken != "" {
				resp.SetToken(tt.respToken)
			}

			mockAuth.On("AuthUser", mock.Anything, mock.MatchedBy(func(r *pb.AuthRequest) bool {
				return r.GetLogin() == "user" && r.GetPassword() == "pass"
			})).Return(resp, tt.respErr).Once()

			token, err := c.Login(context.Background(), "user", "pass")

			assert.Equal(t, tt.wantReturned, token)
			assert.Equal(t, tt.wantToken, c.GetToken())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestStoreRecord(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		serverResp  *pb.RecordID
		serverErr   error
		wantID      int32
		wantErr     bool
		errContains string
	}{
		{
			name:        "no token",
			token:       "",
			wantErr:     true,
			errContains: "токен аутентификации не установлен",
		},
		{
			name:       "success",
			token:      "jwt",
			serverResp: &pb.RecordID{},
			wantID:     42,
		},
		{
			name:      "server error",
			token:     "jwt",
			serverErr: status.Error(codes.Internal, "db error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mockSecureStorageClient{}
			c := &GophKeeperClient{
				storageClient: mockStorage,
				token:         tt.token,
			}

			record := &pb.EncryptedRecord{}

			if tt.token != "" {
				if tt.serverResp != nil {
					tt.serverResp.SetId(tt.wantID)
				}

				mockStorage.On("StoreRecord", mock.MatchedBy(func(ctx context.Context) bool {
					md, ok := metadata.FromOutgoingContext(ctx)
					if !ok {
						return false
					}
					auth := md.Get("authorization")
					return len(auth) == 1 && auth[0] == "Bearer "+tt.token
				}), record).Return(tt.serverResp, tt.serverErr).Once()
			}

			id, err := c.StoreRecord(context.Background(), record)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestGetRecord(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no token",
			token:       "",
			wantErr:     true,
			errContains: "токен аутентификации не установлен",
		},
		{
			name:  "with token",
			token: "jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mockSecureStorageClient{}
			c := &GophKeeperClient{
				storageClient: mockStorage,
				token:         tt.token,
			}

			recordID := int32(42)
			respRecord := &pb.EncryptedRecord{}

			if tt.token != "" {
				req := pb.RecordID_builder{Id: &recordID}.Build()

				mockStorage.On("GetRecord", mock.MatchedBy(func(ctx context.Context) bool {
					md, ok := metadata.FromOutgoingContext(ctx)
					if !ok {
						return false
					}
					auth := md.Get("authorization")
					return len(auth) == 1 && auth[0] == "Bearer "+tt.token
				}), req).Return(respRecord, nil).Once()
			}

			got, err := c.GetRecord(context.Background(), recordID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, respRecord, got)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestUploadBinary(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no token",
			token:       "",
			wantErr:     true,
			errContains: "токен не установлен",
		},
		{
			name:  "with token",
			token: "jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBinary := &mockBinaryStorageClient{}
			mockStream := &mockUploadStream{}

			c := &GophKeeperClient{
				binaryClient: mockBinary,
				token:        tt.token,
			}

			if tt.token != "" {
				mockBinary.On("UploadBinary", mock.MatchedBy(func(ctx context.Context) bool {
					md, ok := metadata.FromOutgoingContext(ctx)
					if !ok {
						return false
					}
					auth := md.Get("authorization")
					return len(auth) == 1 && auth[0] == "Bearer "+tt.token
				})).Return(mockStream, nil).Once()

				mockStream.On("Context").Return(context.Background()).Maybe()
			}

			stream, err := c.UploadBinary(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, stream)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, mockStream, stream)
			}

			mockBinary.AssertExpectations(t)
		})
	}
}

func TestDownloadBinary(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no token",
			token:       "",
			wantErr:     true,
			errContains: "токен не установлен",
		},
		{
			name:  "with token",
			token: "jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBinary := &mockBinaryStorageClient{}
			mockStream := &mockDownloadStream{}

			c := &GophKeeperClient{
				binaryClient: mockBinary,
				token:        tt.token,
			}

			recordID := int32(123)

			if tt.token != "" {
				req := pb.BinaryRecordID_builder{Id: &recordID}.Build()

				mockBinary.On("DownloadBinary", mock.MatchedBy(func(ctx context.Context) bool {
					md, ok := metadata.FromOutgoingContext(ctx)
					if !ok {
						return false
					}
					auth := md.Get("authorization")
					return len(auth) == 1 && auth[0] == "Bearer "+tt.token
				}), req).Return(mockStream, nil).Once()

				mockStream.On("Context").Return(context.Background()).Maybe()
				mockStream.On("CloseSend").Return(nil).Maybe()
			}

			stream, err := c.DownloadBinary(context.Background(), recordID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, stream)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, mockStream, stream)
			}

			mockBinary.AssertExpectations(t)
			mockStream.AssertExpectations(t)
		})
	}
}

func TestClose(t *testing.T) {
	c := &GophKeeperClient{conn: nil}
	assert.NoError(t, c.Close())
}
