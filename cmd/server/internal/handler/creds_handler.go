package handler

import (
	"context"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) StoreCredentials(ctx context.Context, r *proto.Credentials) (*proto.RecordID, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	creds := &domain.Creds{
		Login:       r.GetUsername(),
		Password:    r.GetPassword(),
		ServiceName: r.GetServiceName(),
		Metadata:    r.GetMetadata(),
	}
	creds.UserID = userID

	credsID, err := h.CredsService.Create(ctx, creds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create creds in storage: %v", err)
	}

	i32CredsID := int32(credsID)
	response := proto.RecordID_builder{
		Id: &i32CredsID,
	}

	return response.Build(), nil
}

func (h *Handler) GetCredentials(ctx context.Context, r *proto.RecordID) (*proto.Credentials, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	creds, err := h.CredsService.Get(ctx, int(r.GetId()), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get creds: %v", err)
	}

	response := proto.Credentials_builder{
		Username:    &creds.Login,
		Password:    &creds.Password,
		ServiceName: &creds.ServiceName,
		Metadata:    &creds.Metadata,
	}

	return response.Build(), nil
}

func (h *Handler) ListCredentials(ctx context.Context, _ *proto.Empty) (*proto.RecordList, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	creds, err := h.CredsService.ListIDs(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get list if creds IDs: %v", err)
	}

	response := proto.RecordList_builder{
		Records: make([]*proto.RecordID, 0, len(creds)),
	}

	for _, credsID := range creds {
		i32ID := int32(credsID)
		id := proto.RecordID_builder{
			Id: &i32ID,
		}
		response.Records = append(response.Records, id.Build())
	}

	return response.Build(), nil
}

func (h *Handler) DeleteCredentials(ctx context.Context, r *proto.RecordID) (*proto.Empty, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	err := h.CredsService.Delete(ctx, int(r.GetId()), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot delete cred: %v", err)
	}

	return nil, nil
}
