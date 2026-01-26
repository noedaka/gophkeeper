package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gophkeeper/cmd/server/internal/handler"
	"gophkeeper/cmd/server/internal/interceptor"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/config"
	"gophkeeper/internal/proto"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	minioClient, err := initMinIO(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO: %v", err)
	}

	if err := ensureMinIOBucket(minioClient, cfg); err != nil {
		log.Fatalf("Failed to ensure MinIO bucket: %v", err)
	}

	userRepo := repository.NewUserRepo(db)
	recordRepo := repository.NewRecordRepo(db)
	binaryRepo := repository.NewBinaryRepo(db, minioClient, *cfg)

	userService := service.NewUserService(userRepo)
	recordService := service.NewRecordServ(recordRepo)
	binaryService := service.NewBinaryServ(binaryRepo, minioClient, *cfg)

	interceptor := interceptor.NewInterceptor(*cfg)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.AuthUnaryInterceptor),
		grpc.StreamInterceptor(interceptor.AuthStreamInterceptor),
	)

	handler := handler.NewHandler(userService, recordService, binaryService, cfg)
	proto.RegisterAuthServiceServer(grpcServer, handler)
	proto.RegisterSecureStorageServer(grpcServer, handler)
	proto.RegisterBinaryStorageServer(grpcServer, handler)

	listener, err := net.Listen("tcp", cfg.ServerPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Starting gRPC server on %s", cfg.ServerPort)

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Received shutdown signal, initiating graceful shutdown...")
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	gracefulCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-gracefulCtx.Done():
		log.Println("Forced shutdown after timeout")
		grpcServer.Stop()
	case <-stopped:
		log.Println("Server stopped gracefully")
	}
}

// initDatabase инициализирует подключение к базе данных
func initDatabase(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// initMinIO инициализирует клиент MinIO
func initMinIO(cfg *config.Config) (*minio.Client, error) {
	minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOTLS,
	})
	if err != nil {
		return nil, err
	}

	log.Println("MinIO client initialized successfully")
	return minioClient, nil
}

// ensureMinIOBucket проверяет существование bucket и создает его при необходимости
func ensureMinIOBucket(minioClient *minio.Client, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, cfg.MinIOBucket)
	if err != nil {
		return err
	}

	if !exists {
		if err := minioClient.MakeBucket(ctx, cfg.MinIOBucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
		log.Printf("Bucket '%s' created successfully", cfg.MinIOBucket)
	} else {
		log.Printf("Bucket '%s' already exists", cfg.MinIOBucket)
	}

	return nil
}
