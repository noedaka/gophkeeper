package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
)

type BinaryRepo struct {
	db          *sql.DB
	minioClient *minio.Client
	cfg         config.Config
}

// NewBinaryRepo создаёт репозиторий для бинарных файлов
func NewBinaryRepo(db *sql.DB, minioClient *minio.Client, cfg config.Config) *BinaryRepo {
	return &BinaryRepo{
		db:          db,
		minioClient: minioClient,
		cfg:         cfg,
	}
}

// generateS3Key генерирует уникальный ключ для объекта в MinIO
func (r *BinaryRepo) generateS3Key(userID string, recordID int) string {
	return fmt.Sprintf("binary/%s/%d.bin.enc", userID, recordID)
}

// CreateRecord создаёт запись в БД и возвращает ID + финальный S3 ключ для upload
func (r *BinaryRepo) Create(ctx context.Context, userID string, metadata string) (int, string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", err
	}
	 defer func() {
		if err := tx.Rollback(); err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				log.Printf("failed to rollback the transaction: %v", err)
			}
		}
	}()

	// Генерируем временный уникальный s3_key (чтобы удовлетворить NOT NULL + UNIQUE при INSERT)
	tempKey := fmt.Sprintf("pending/%s/%d", userID, time.Now().UnixNano())

	var id int
	err = tx.QueryRowContext(ctx,
		`INSERT INTO binary_records (user_id, metadata, s3_key) 
		VALUES ($1, $2, $3) 
		RETURNING id`,
		userID, metadata, tempKey,
	).Scan(&id)
	if err != nil {
		return 0, "", err
	}

	s3Key := r.generateS3Key(userID, id)

	_, err = tx.ExecContext(ctx,
		`UPDATE binary_records SET s3_key = $1 WHERE id = $2`,
		s3Key, id,
	)
	if err != nil {
		return 0, "", err
	}

	if err = tx.Commit(); err != nil {
		return 0, "", err
	}

	return id, s3Key, nil
}

func (r *BinaryRepo) GetKey(ctx context.Context, ID int, userID string) (string, error) {
	var s3Key string
	err := r.db.QueryRowContext(ctx,
		`SELECT s3_key
        FROM binary_records 
        WHERE id = $1 AND user_id = $2`,
		ID, userID).Scan(&s3Key)

	if err != nil {
		return "", err
	}

	return s3Key, nil
}

// List возвращает список бинарных записей пользователя
func (r *BinaryRepo) List(ctx context.Context, userID string) ([]domain.BinaryRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, metadata, s3_key FROM binary_records WHERE user_id = $1 ORDER BY id ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.BinaryRecord
	for rows.Next() {
		var rec domain.BinaryRecord
		if err = rows.Scan(&rec.ID, &rec.Metadata, &rec.S3Key); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// Delete удаляет объект из БД и из хранилища MinIO
func (r *BinaryRepo) Delete(ctx context.Context, recordID int, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var s3Key string
	err = tx.QueryRowContext(ctx,
		`DELETE FROM binary_records 
        WHERE id = $1 AND user_id = $2 
        RETURNING s3_key`,
		recordID, userID,
	).Scan(&s3Key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("record not found or unauthorized")
		}
		return err
	}

	removeOpts := minio.RemoveObjectOptions{}

	if err := r.minioClient.RemoveObject(ctx, r.cfg.MinIOBucket, s3Key, removeOpts); err != nil {
		return fmt.Errorf("failed to remove object from MinIO: %w", err)
	}

	return tx.Commit()
}
