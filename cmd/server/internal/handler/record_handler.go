package handler

import (
	"context"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StoreRecord сохраняет запись
func (h *Handler) StoreRecord(ctx context.Context, r *proto.EncryptedRecord) (*proto.RecordID, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	record := &domain.Record{
		Ciphertext: r.GetCiphertext(),
		Nonce:      r.GetNonce(),
		Metadata:   r.GetMetadata(),
		RecordType: r.GetRecordType(),
	}

	record.UserID = userID

	recordID, err := h.RecordService.Create(ctx, record)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create record in storage: %v", err)
	}

	i32RecordID := int32(recordID)
	response := proto.RecordID_builder{
		Id: &i32RecordID,
	}

	return response.Build(), nil
}

// GetRecord возвращает запись
func (h *Handler) GetRecord(ctx context.Context, r *proto.RecordID) (*proto.EncryptedRecord, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	record, err := h.RecordService.Get(ctx, int(r.GetId()), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get record: %v", err)
	}

	response := proto.EncryptedRecord_builder{
		Ciphertext: record.Ciphertext,
		Nonce:      record.Nonce,
		Metadata:   &record.Metadata,
		RecordType: &record.RecordType,
	}

	return response.Build(), nil
}

// ListRecords возвращает список ID записей
func (h *Handler) ListRecords(ctx context.Context, r *proto.ListRequest) (*proto.RecordList, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	infos, err := h.RecordService.List(ctx, userID, r.GetRecordType())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get list of records: %v", err)
	}

	response := proto.RecordList_builder{
		Records: make([]*proto.RecordInfo, 0, len(infos)),
	}

	for _, info := range infos {
		i32ID := int32(info.ID)
		recInfo := proto.RecordInfo_builder{
			Id:         &i32ID,
			Metadata:   &info.Metadata,
			RecordType: &info.RecordType,
		}
		response.Records = append(response.Records, recInfo.Build())
	}

	return response.Build(), nil
}

// DeleteRecord удаляет запись
func (h *Handler) DeleteRecord(ctx context.Context, r *proto.RecordID) (*proto.Empty, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	err := h.RecordService.Delete(ctx, int(r.GetId()), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot delete record: %v", err)
	}

	response := proto.Empty_builder{}

	return response.Build(), nil
}
