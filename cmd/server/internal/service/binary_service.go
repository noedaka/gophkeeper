package service

import (
	"context"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"io"

	"github.com/minio/minio-go/v7"
)

type BinaryServ struct {
	repo        repository.BinaryRepository
	minioClient *minio.Client
	cfg         config.Config
}

func NewBinaryServ(repo repository.BinaryRepository, minioClient *minio.Client, cfg config.Config) *BinaryServ {
	return &BinaryServ{
		repo:        repo,
		minioClient: minioClient,
		cfg:         cfg,
	}
}

func (s *BinaryServ) Create(ctx context.Context, userID string, metadata string) (int, string, error) {
	return s.repo.Create(ctx, userID, metadata)
}

// Get возвращает ReadCloser для streaming download
func (s *BinaryServ) Get(ctx context.Context, s3Key string) (io.ReadCloser, int64, error) {
	obj, err := s.minioClient.GetObject(ctx, s.cfg.MinIOBucket, s3Key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}

	stat, err := obj.Stat()
	if err != nil {
		return nil, 0, err
	}

	return obj, stat.Size, nil
}

func (s *BinaryServ) List(ctx context.Context, userID string) ([]domain.BinaryRecord, error) {
	return s.repo.List(ctx, userID)
}

func (s *BinaryServ) Delete(ctx context.Context, recordID int, userID string) error {
	return s.repo.Delete(ctx, recordID, userID)
}

// Upload сохраняет объект в MinIO
func (s *BinaryServ) Upload(ctx context.Context, s3Key string, reader io.Reader, size int64) error {
	_, err := s.minioClient.PutObject(ctx, s.cfg.MinIOBucket, s3Key, reader, size,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

func (r *BinaryServ) GetKey(ctx context.Context, ID int, userID string) (string, error) {
	return r.repo.GetKey(ctx, ID, userID)
}
