package repository

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/model"
	"log"
)

type CredsRepo struct {
	db *sql.DB
}

func NewCredsRepo(db *sql.DB) *CredsRepo {
	return &CredsRepo{db: db}
}

func (r *CredsRepo) Create(ctx context.Context, creds *domain.Creds) (int, error) {
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

	//TODO Card data encryption
	var ID int
	err = tx.QueryRowContext(ctx,
		"INSERT INTO creds (login, password, service_name, metadata, user_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		creds.Login, creds.Password, creds.ServiceName, creds.Metadata, creds.UserID,
	).Scan(&ID)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return ID, nil
}

func (r *CredsRepo) Get(ctx context.Context, credsID int, userID int) (*domain.Creds, error) {
	var creds domain.Creds
	err := r.db.QueryRowContext(ctx,
		`SELECT login, password, service_name, metadata, user_id
        FROM creds WHERE id = $1`, credsID,
	).Scan(&creds.Login, &creds.Password, &creds.ServiceName, &creds.Metadata, &creds.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoContent
		}

		return nil, err
	}

	if userID != creds.UserID {
		return nil, model.ErrUnauthorizedAccess
	}

	return &creds, nil
}

func (r *CredsRepo) ListIDs(ctx context.Context, userID int) ([]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM creds WHERE user_id = $1`, userID,
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

func (r *CredsRepo) Delete(ctx context.Context, credsID int, userID int) error {
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
		`DELETE FROM creds WHERE id = $1 AND user_id = $2`,
		credsID, userID,
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
