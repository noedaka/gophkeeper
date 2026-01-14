package handler

import (
	"context"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) StoreCard(ctx context.Context, r *proto.Card) (*proto.RecordID, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	card := &domain.Card{
		CardNumber:     r.GetCardNumber(),
		CardHolderName: r.GetCardHolder(),
		ExpiryDate:     r.GetExpiryDate(),
		CVV:            r.GetCvv(),
		Metadata:       r.GetMetadata(),
		
	}
	card.UserID = userID

	cardID, err := h.CardService.Create(ctx, card)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create card in storage: %v", err)
	}

	i32CardID := int32(cardID)
	response := proto.RecordID_builder{
		Id: &i32CardID,
	}

	return response.Build(), nil
}

func (h *Handler) GetCard(ctx context.Context, r *proto.RecordID) (*proto.Card, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	card, err := h.CardService.Get(ctx, int(r.GetId()), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get card: %v", err)
	}

	response := proto.Card_builder{
		CardNumber: &card.CardNumber,
		CardHolder: &card.CardHolderName,
		Cvv:        &card.CVV,
		ExpiryDate: &card.ExpiryDate,
		Metadata:   &card.Metadata,
	}

	return response.Build(), nil
}

func (h *Handler) ListCards(ctx context.Context, _ *proto.Empty) (*proto.RecordList, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	cards, err := h.CardService.ListIDs(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get list if card IDs: %v", err)
	}

	response := proto.RecordList_builder{
		Records: make([]*proto.RecordID, 0, len(cards)),
	}

	for _, cardID := range cards {
		i32ID := int32(cardID)
		id := proto.RecordID_builder{
			Id: &i32ID,
		}
		response.Records = append(response.Records, id.Build())
	}

	return response.Build(), nil
}

func (h *Handler) DeleteCard(ctx context.Context, r *proto.RecordID) (*proto.Empty, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	err := h.CardService.Delete(ctx, int(r.GetId()), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot delete card: %v", err)
	}

	return nil, nil
}

const UserIDKey ContextKey = "user_id"

type ContextKey string
