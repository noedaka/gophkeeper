package repository

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/model"
	"log"
)

type CardRepo struct {
	db *sql.DB
}

func NewCardRepo(db *sql.DB) *CardRepo {
	return &CardRepo{db: db}
}

func (r *CardRepo) Create(ctx context.Context, card *domain.Card) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				log.Printf("failed to rollback the transaction: %v", err)
			}
		}
	}()

	var ID int
	//TODO Card data encryption
	err = tx.QueryRowContext(ctx,
		"INSERT INTO cards (card_number, card_holder_name, expiry_date, cvv, metadata, user_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		card.CardNumber, card.CardHolderName, card.ExpiryDate, card.CVV, card.Metadata, card.UserID,
	).Scan(&ID)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return ID, nil
}

func (r *CardRepo) Get(ctx context.Context, cardID int, userID int) (*domain.Card, error) {
	var card domain.Card
	err := r.db.QueryRowContext(ctx,
		`SELECT card_number, card_holder_name, expiry_date, cvv, metadata, user_id
        FROM cards WHERE id = $1`, cardID,
	).Scan(&card.CardNumber, &card.CardHolderName, &card.ExpiryDate, &card.CVV, &card.Metadata, &card.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoContent
		}

		return nil, err
	}

	if userID != card.UserID {
		return nil, model.ErrUnauthorizedAccess
	}

	return &card, nil
}

func (r *CardRepo) ListIDs(ctx context.Context, userID int) ([]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM cards WHERE user_id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var IDs []int
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		IDs = append(IDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(IDs) == 0 {
		return nil, model.ErrNoContent
	}

	return IDs, nil
}

func (r *CardRepo) Delete(ctx context.Context, cardID int, userID int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				log.Printf("failed to rollback the transaction: %v", err)
			}
		}
	}()

	result, err := r.db.ExecContext(ctx,
		`DELETE FROM cards WHERE id = $1 AND user_id = $2`,
		cardID, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return model.ErrNoContent
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
