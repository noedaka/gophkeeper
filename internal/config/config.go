package config

import (
	"flag"
	"fmt"
	"net"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerPort        string `env:"SERVER_PORT"`
	GRPCServerAddress string `env:"GRPC_SERVER_ADDRESS"`

	DBHost     string `env:"DB_HOST"`
	DBPort     string `env:"DB_PORT"`
	DBUser     string `env:"DB_USER"`
	DBPassword string `env:"DB_PASSWORD"`
	DBName     string `env:"DB_NAME"`
	DBSSLMode  string `env:"DB_SSLMODE"`

	JWTSecret string `env:"JWT_SECRET"`
	CACrtFile string `env:"CA_CRT_FILE"`

	MinIOEndpoint  string `env:"MINIO_ENDPOINT"`
	MinIOAccessKey string `env:"MINIO_ROOT_USER"`
	MinIOSecretKey string `env:"MINIO_ROOT_PASSWORD"`
	MinIOBucket    string `env:"MINIO_BUCKET"`
	MinIOTLS       bool `env:"MINIO_TLS"`
}

func Init() (*Config, error) {
	cfg := &Config{}

	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	flag.Parse()

	return cfg, nil
}

func (cfg *Config) ValidateConfig() error {
	_, _, err := net.SplitHostPort(cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("invalid server address format: %w", err)
	}

	return nil
}
