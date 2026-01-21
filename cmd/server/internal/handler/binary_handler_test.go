package handler_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"gophkeeper/cmd/server/internal/handler"
	"gophkeeper/cmd/server/internal/interceptor"
	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"
)

type mockBinaryService struct {
	mock.Mock
}

func (m *mockBinaryService) Create(ctx context.Context, userID int, metadata string) (int, string, error) {
	args := m.Called(ctx, userID, metadata)
	return args.Int(0), args.String(1), args.Error(2)
}

func (m *mockBinaryService) Upload(ctx context.Context, s3Key string, reader io.Reader, size int64) error {
	args := m.Called(ctx, s3Key, reader, size)
	return args.Error(0)
}

func (m *mockBinaryService) GetKey(ctx context.Context, recordID int, userID int) (string, error) {
	args := m.Called(ctx, recordID, userID)
	return args.String(0), args.Error(1)
}

func (m *mockBinaryService) Get(ctx context.Context, s3Key string) (io.ReadCloser, int64, error) {
	args := m.Called(ctx, s3Key)
	return args.Get(0).(io.ReadCloser), args.Get(1).(int64), args.Error(2)
}

func (m *mockBinaryService) List(ctx context.Context, userID int) ([]domain.BinaryRecord, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.BinaryRecord), args.Error(1)
}

func (m *mockBinaryService) Delete(ctx context.Context, recordID int, userID int) error {
	args := m.Called(ctx, recordID, userID)
	return args.Error(0)
}

type mockUploadStream struct {
	mock.Mock
}

func (m *mockUploadStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockUploadStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockUploadStream) SetTrailer(md metadata.MD)       {}
func (m *mockUploadStream) SendMsg(any) error               { return nil }
func (m *mockUploadStream) RecvMsg(any) error               { return nil }

func (m *mockUploadStream) Context() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *mockUploadStream) Recv() (*proto.BinaryChunk, error) {
	args := m.Called()
	return args.Get(0).(*proto.BinaryChunk), args.Error(1)
}

func (m *mockUploadStream) SendAndClose(resp *proto.BinaryRecordID) error {
	args := m.Called(resp)
	return args.Error(0)
}

type mockDownloadStream struct {
	mock.Mock
}

func (m *mockDownloadStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockDownloadStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockDownloadStream) SetTrailer(md metadata.MD)       {}
func (m *mockDownloadStream) SendMsg(any) error               { return nil }
func (m *mockDownloadStream) RecvMsg(any) error               { return nil }

func (m *mockDownloadStream) Context() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *mockDownloadStream) Send(chunk *proto.BinaryChunk) error {
	args := m.Called(chunk)
	return args.Error(0)
}

func binaryCtxWithUserID(userID int) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey{}, userID)
}

func binaryPtrString(s string) *string { return &s }
func binaryPtrBool(b bool) *bool       { return &b }
func binaryPtrInt32(i int32) *int32    { return &i }

func TestHandler_UploadBinary(t *testing.T) {
	mockBinarySvc := &mockBinaryService{}
	h := handler.NewHandler(nil, nil, mockBinarySvc, &config.Config{})

	tests := []struct {
		name         string
		ctx          context.Context
		setupStream  func(*mockUploadStream)
		setupService func()
		wantRecordID int32
		wantErr      bool
		wantCode     codes.Code
		wantRollback bool
	}{
		{
			name: "success: multiple chunks",
			ctx:  binaryCtxWithUserID(42),
			setupStream: func(stream *mockUploadStream) {
				stream.On("Context").Return(binaryCtxWithUserID(42))

				firstChunk := proto.BinaryChunk_builder{
					Metadata: binaryPtrString("file.txt"),
					Data:     []byte("part1"),
				}.Build()
				stream.On("Recv").Return(firstChunk, nil).Once()

				secondChunk := proto.BinaryChunk_builder{
					Data: []byte("part2"),
				}.Build()
				stream.On("Recv").Return(secondChunk, nil).Once()

				lastChunk := proto.BinaryChunk_builder{
					Data:   []byte("part3"),
					IsLast: binaryPtrBool(true),
				}.Build()
				stream.On("Recv").Return(lastChunk, nil).Once()

				stream.On("Recv").Return((*proto.BinaryChunk)(nil), io.EOF).Once()

				stream.On("SendAndClose", mock.MatchedBy(func(resp *proto.BinaryRecordID) bool {
					return resp.GetId() == 100
				})).Return(nil).Once()
			},
			setupService: func() {
				mockBinarySvc.On("Create", mock.Anything, 42, "file.txt").Return(100, "s3key-100", nil)
				mockBinarySvc.On("Upload", mock.Anything, "s3key-100", mock.Anything, int64(-1)).Return(nil)
			},
			wantRecordID: 100,
			wantErr:      false,
		},
		{
			name: "no metadata in first chunk",
			ctx:  binaryCtxWithUserID(42),
			setupStream: func(stream *mockUploadStream) {
				stream.On("Context").Return(binaryCtxWithUserID(42))
				firstChunk := proto.BinaryChunk_builder{Data: []byte("data")}.Build()
				stream.On("Recv").Return(firstChunk, nil).Once()
			},
			setupService: func() {
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "no chunks at all",
			ctx:  binaryCtxWithUserID(42),
			setupStream: func(stream *mockUploadStream) {
				stream.On("Context").Return(binaryCtxWithUserID(42))
				stream.On("Recv").Return((*proto.BinaryChunk)(nil), io.EOF).Once()
			},
			setupService: func() {},
			wantErr:      true,
			wantCode:     codes.InvalidArgument,
		},
		{
			name: "upload error → rollback",
			ctx:  binaryCtxWithUserID(42),
			setupStream: func(stream *mockUploadStream) {
				stream.On("Context").Return(binaryCtxWithUserID(42))
				firstChunk := proto.BinaryChunk_builder{
					Metadata: binaryPtrString("file.txt"),
					Data:     []byte("data"),
					IsLast:   binaryPtrBool(true),
				}.Build()
				stream.On("Recv").Return(firstChunk, nil).Once()
				stream.On("Recv").Return((*proto.BinaryChunk)(nil), io.EOF).Once()
				stream.On("SendAndClose", mock.Anything).Return(nil).Maybe()
			},
			setupService: func() {
				mockBinarySvc.On("Create", mock.Anything, 42, "file.txt").Return(100, "s3key-100", nil)
				mockBinarySvc.On("Upload", mock.Anything, "s3key-100", mock.Anything, int64(-1)).Return(errors.New("minio error"))
				mockBinarySvc.On("Delete", mock.Anything, 100, 42).Return(nil)
			},
			wantErr:      true,
			wantCode:     codes.Internal,
			wantRollback: true,
		},
		{
			name: "unauthenticated",
			ctx:  context.Background(),
			setupStream: func(stream *mockUploadStream) {
				stream.On("Context").Return(context.Background())
			},
			setupService: func() {},
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &mockUploadStream{}
			tt.setupStream(stream)
			tt.setupService()

			err := h.UploadBinary(stream)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				assert.NoError(t, err)
			}

			mockBinarySvc.AssertExpectations(t)
			stream.AssertExpectations(t)
		})
	}
}

func TestHandler_DownloadBinary(t *testing.T) {
	mockBinarySvc := &mockBinaryService{}
	h := handler.NewHandler(nil, nil, mockBinarySvc, &config.Config{})

	tests := []struct {
		name         string
		ctx          context.Context
		req          *proto.BinaryRecordID
		setupStream  func(*mockDownloadStream)
		setupService func()
		wantChunks   int
		wantErr      bool
		wantCode     codes.Code
	}{
		{
			name: "success: data > chunkSize",
			ctx:  binaryCtxWithUserID(42),
			req:  proto.BinaryRecordID_builder{Id: binaryPtrInt32(100)}.Build(),
			setupStream: func(stream *mockDownloadStream) {
				stream.On("Context").Return(binaryCtxWithUserID(42))
				stream.On("Send", mock.Anything).Return(nil).Times(2)
			},
			setupService: func() {
				data := make([]byte, 5*1024*1024)
				for i := range data {
					data[i] = byte(i % 256)
				}
				mockBinarySvc.On("GetKey", mock.Anything, 100, 42).Return("s3key-100", nil)
				mockBinarySvc.On("Get", mock.Anything, "s3key-100").Return(io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil)
			},
			wantChunks: 2,
			wantErr:    false,
		},
		{
			name: "empty file",
			ctx:  binaryCtxWithUserID(42),
			req:  proto.BinaryRecordID_builder{Id: binaryPtrInt32(100)}.Build(),
			setupStream: func(stream *mockDownloadStream) {
				stream.On("Context").Return(binaryCtxWithUserID(42))
				stream.On("Send", mock.Anything).Return(nil).Times(0)
			},
			setupService: func() {
				mockBinarySvc.On("GetKey", mock.Anything, 100, 42).Return("s3key-100", nil)
				mockBinarySvc.On("Get", mock.Anything, "s3key-100").Return(io.NopCloser(bytes.NewReader([]byte{})), int64(0), nil)
			},
			wantChunks: 0,
			wantErr:    false,
		},
		{
			name: "not found",
			ctx:  binaryCtxWithUserID(42),
			req:  proto.BinaryRecordID_builder{Id: binaryPtrInt32(999)}.Build(),
			setupStream: func(stream *mockDownloadStream) {
				stream.On("Context").Return(binaryCtxWithUserID(42))
			},
			setupService: func() {
				mockBinarySvc.On("GetKey", mock.Anything, 999, 42).Return("", errors.New("not found"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
		{
			name: "unauthenticated",
			ctx:  context.Background(),
			req:  proto.BinaryRecordID_builder{}.Build(),
			setupStream: func(stream *mockDownloadStream) {
				stream.On("Context").Return(context.Background())
			},
			setupService: func() {},
			wantErr:      true,
			wantCode:     codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &mockDownloadStream{}
			tt.setupStream(stream)
			tt.setupService()

			err := h.DownloadBinary(tt.req, stream)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				assert.NoError(t, err)
			}

			mockBinarySvc.AssertExpectations(t)
			stream.AssertExpectations(t)
		})
	}
}

func TestHandler_ListBinaries(t *testing.T) {
	mockBinarySvc := &mockBinaryService{}
	h := handler.NewHandler(nil, nil, mockBinarySvc, &config.Config{})

	tests := []struct {
		name     string
		ctx      context.Context
		setup    func()
		wantLen  int
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name: "success",
			ctx:  binaryCtxWithUserID(42),
			setup: func() {
				mockBinarySvc.On("List", mock.Anything, 42).Return([]struct {
					ID       int
					Metadata string
				}{
					{ID: 1, Metadata: "file1.txt"},
					{ID: 2, Metadata: "file2.bin"},
				}, nil)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:     "unauthenticated",
			ctx:      context.Background(),
			setup:    func() {},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			resp, err := h.ListBinaries(tt.ctx, &proto.BinaryEmpty{})

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Len(t, resp.GetRecords(), tt.wantLen)
			}

			mockBinarySvc.AssertExpectations(t)
		})
	}
}

func TestHandler_DeleteBinary(t *testing.T) {
	mockBinarySvc := &mockBinaryService{}
	h := handler.NewHandler(nil, nil, mockBinarySvc, &config.Config{})

	tests := []struct {
		name     string
		ctx      context.Context
		req      *proto.BinaryRecordID
		setup    func()
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name: "success",
			ctx:  binaryCtxWithUserID(42),
			req:  proto.BinaryRecordID_builder{Id: binaryPtrInt32(100)}.Build(),
			setup: func() {
				mockBinarySvc.On("Delete", mock.Anything, 100, 42).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "unauthenticated",
			ctx:      context.Background(),
			req:      proto.BinaryRecordID_builder{}.Build(),
			setup:    func() {},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			resp, err := h.DeleteBinary(tt.ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockBinarySvc.AssertExpectations(t)
		})
	}
}