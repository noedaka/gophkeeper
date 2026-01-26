package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/model"
)

// RecordRepo структура репозитория записи
type RecordRepo struct {
	db *sql.DB
}

// NewRecordRepo создаёт новый репозиторий для унифицированных записей
func NewRecordRepo(db *sql.DB) *RecordRepo {
	return &RecordRepo{db: db}
}

// Create сохраняет новую зашифрованную запись и возвращает её ID
func (r *RecordRepo) Create(ctx context.Context, record *domain.Record) (int, error) {
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

	var id int
	err = tx.QueryRowContext(ctx,
		`INSERT INTO records 
        (user_id, ciphertext, nonce, metadata, record_type) 
        VALUES ($1, $2, $3, $4, $5) 
        RETURNING id`,
		record.UserID, record.Ciphertext, record.Nonce, record.Metadata, record.RecordType,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

// Get получает полную зашифрованную запись по ID с проверкой владения
func (r *RecordRepo) Get(ctx context.Context, recordID int, userID string) (*domain.Record, error) {
	var rec domain.Record
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, ciphertext, nonce, metadata, record_type
        FROM records 
        WHERE id = $1`,
		recordID,
	).Scan(&rec.ID, &rec.UserID, &rec.Ciphertext, &rec.Nonce, &rec.Metadata, &rec.RecordType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoContent
		}
		return nil, err
	}

	if rec.UserID != userID {
		return nil, model.ErrUnauthorizedAccess
	}

	return &rec, nil
}

// List возвращает список упрощённой информации о записях пользователя
func (r *RecordRepo) List(ctx context.Context, userID string, recordType string) ([]domain.RecordInfo, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, metadata, record_type 
        FROM records WHERE user_id = $1 
        AND record_type = $2 ORDER BY id ASC`, userID, recordType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infos []domain.RecordInfo
	for rows.Next() {
		var info domain.RecordInfo
		if err = rows.Scan(&info.ID, &info.Metadata, &info.RecordType); err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(infos) == 0 {
		return nil, nil
	}

	return infos, nil
}

// Delete удаляет запись по ID с проверкой владения
func (r *RecordRepo) Delete(ctx context.Context, recordID int, userID string) error {
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

	result, err := tx.ExecContext(ctx,
		`DELETE FROM records 
		WHERE id = $1 AND user_id = $2`,
		recordID, userID,
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
