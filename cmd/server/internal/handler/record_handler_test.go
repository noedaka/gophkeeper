package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gophkeeper/cmd/server/internal/handler"
	"gophkeeper/cmd/server/internal/interceptor"
	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"
)

type mockRecordService struct {
	mock.Mock
}

func (m *mockRecordService) Create(ctx context.Context, record *domain.Record) (int, error) {
	args := m.Called(ctx, record)
	return args.Int(0), args.Error(1)
}

func (m *mockRecordService) Get(ctx context.Context, id int, userID string) (*domain.Record, error) {
	args := m.Called(ctx, id, userID)
	return args.Get(0).(*domain.Record), args.Error(1)
}

func (m *mockRecordService) List(ctx context.Context, userID string, recordType string) ([]domain.RecordInfo, error) {
	args := m.Called(ctx, userID, recordType)
	return args.Get(0).([]domain.RecordInfo), args.Error(1)
}

func (m *mockRecordService) Delete(ctx context.Context, id int, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func ctxWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey{}, userID)
}

func TestHandler_StoreRecord(t *testing.T) {
	mockRecordSvc := &mockRecordService{}
	h := handler.NewHandler(nil, mockRecordSvc, nil, &config.Config{})

	tests := []struct {
		name         string
		ctx          context.Context
		request      *proto.EncryptedRecord
		mockSetup    func()
		wantRecordID int32
		wantErr      bool
		wantCode     codes.Code
	}{
		{
			name: "success",
			ctx:  ctxWithUserID("42"),
			request: proto.EncryptedRecord_builder{
				Ciphertext: []byte("encrypted data"),
				Nonce:      []byte("nonce123"),
				Metadata:   ptrString("meta"),
				RecordType: ptrString("text"),
			}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("Create", mock.Anything, mock.MatchedBy(func(r *domain.Record) bool {
					return r.UserID == "42" &&
						string(r.Ciphertext) == "encrypted data" &&
						string(r.Nonce) == "nonce123" &&
						r.Metadata == "meta" &&
						r.RecordType == "text"
				})).Return(100, nil)
			},
			wantRecordID: 100,
			wantErr:      false,
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			request: proto.EncryptedRecord_builder{}.Build(),
			mockSetup: func() {
			},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name:    "service error",
			ctx:     ctxWithUserID("42"),
			request: proto.EncryptedRecord_builder{}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("Create", mock.Anything, mock.Anything).Return(0, errors.New("db error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRecordSvc.ExpectedCalls = nil
			tt.mockSetup()

			resp, err := h.StoreRecord(tt.ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.wantRecordID, resp.GetId())
			}

			mockRecordSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_GetRecord(t *testing.T) {
	mockRecordSvc := &mockRecordService{}
	h := handler.NewHandler(nil, mockRecordSvc, nil, &config.Config{})

	tests := []struct {
		name      string
		ctx       context.Context
		request   *proto.RecordID
		mockSetup func()
		wantResp  *proto.EncryptedRecord
		wantErr   bool
		wantCode  codes.Code
	}{
		{
			name:    "success",
			ctx:     ctxWithUserID("42"),
			request: proto.RecordID_builder{Id: ptrInt32(100)}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("Get", mock.Anything, 100, "42").Return(&domain.Record{
					Ciphertext: []byte("encrypted"),
					Nonce:      []byte("nonce"),
					Metadata:   "meta",
					RecordType: "text",
				}, nil)
			},
			wantResp: proto.EncryptedRecord_builder{
				Ciphertext: []byte("encrypted"),
				Nonce:      []byte("nonce"),
				Metadata:   ptrString("meta"),
				RecordType: ptrString("text"),
			}.Build(),
			wantErr: false,
		},
		{
			name:      "unauthenticated",
			ctx:       context.Background(),
			request:   proto.RecordID_builder{}.Build(),
			mockSetup: func() {},
			wantErr:   true,
			wantCode:  codes.Unauthenticated,
		},
		{
			name:    "service error",
			ctx:     ctxWithUserID("42"),
			request: proto.RecordID_builder{Id: ptrInt32(999)}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("Get", mock.Anything, 999, "42").Return((*domain.Record)(nil), errors.New("not found"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRecordSvc.ExpectedCalls = nil
			tt.mockSetup()

			resp, err := h.GetRecord(tt.ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResp.GetCiphertext(), resp.GetCiphertext())
				assert.Equal(t, tt.wantResp.GetNonce(), resp.GetNonce())
				assert.Equal(t, tt.wantResp.GetMetadata(), resp.GetMetadata())
				assert.Equal(t, tt.wantResp.GetRecordType(), resp.GetRecordType())
			}

			mockRecordSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_ListRecords(t *testing.T) {
	mockRecordSvc := &mockRecordService{}
	h := handler.NewHandler(nil, mockRecordSvc, nil, &config.Config{})

	tests := []struct {
		name      string
		ctx       context.Context
		request   *proto.ListRequest
		mockSetup func()
		wantLen   int
		wantErr   bool
		wantCode  codes.Code
	}{
		{
			name:    "success with records",
			ctx:     ctxWithUserID("42"),
			request: proto.ListRequest_builder{RecordType: ptrString("text")}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("List", mock.Anything, "42", "text").Return([]domain.RecordInfo{
					{ID: 1, Metadata: "meta1", RecordType: "text"},
					{ID: 2, Metadata: "meta2", RecordType: "text"},
				}, nil)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "success empty list",
			ctx:     ctxWithUserID("42"),
			request: proto.ListRequest_builder{}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("List", mock.Anything, "42", "").Return([]domain.RecordInfo{}, nil)
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:      "unauthenticated",
			ctx:       context.Background(),
			request:   proto.ListRequest_builder{}.Build(),
			mockSetup: func() {},
			wantErr:   true,
			wantCode:  codes.Unauthenticated,
		},
		{
			name:    "service error",
			ctx:     ctxWithUserID("42"),
			request: proto.ListRequest_builder{}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("List", mock.Anything, "42", "").Return([]domain.RecordInfo{}, errors.New("db error"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRecordSvc.ExpectedCalls = nil
			tt.mockSetup()

			resp, err := h.ListRecords(tt.ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.GetRecords(), tt.wantLen)
			}

			mockRecordSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_DeleteRecord(t *testing.T) {
	mockRecordSvc := &mockRecordService{}
	h := handler.NewHandler(nil, mockRecordSvc, nil, &config.Config{})

	tests := []struct {
		name      string
		ctx       context.Context
		request   *proto.RecordID
		mockSetup func()
		wantErr   bool
		wantCode  codes.Code
	}{
		{
			name:    "success",
			ctx:     ctxWithUserID("42"),
			request: proto.RecordID_builder{Id: ptrInt32(100)}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("Delete", mock.Anything, 100, "42").Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "unauthenticated",
			ctx:       context.Background(),
			request:   proto.RecordID_builder{}.Build(),
			mockSetup: func() {},
			wantErr:   true,
			wantCode:  codes.Unauthenticated,
		},
		{
			name:    "service error",
			ctx:     ctxWithUserID("42"),
			request: proto.RecordID_builder{Id: ptrInt32(999)}.Build(),
			mockSetup: func() {
				mockRecordSvc.On("Delete", mock.Anything, 999, "42").Return(errors.New("not found"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRecordSvc.ExpectedCalls = nil
			tt.mockSetup()

			resp, err := h.DeleteRecord(tt.ctx, tt.request)

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

			mockRecordSvc.AssertExpectations(t)
		})
	}
}

func ptrString(s string) *string { return &s }
func ptrInt32(i int32) *int32    { return &i }
