package main

import (
	"database/sql"
	"fmt"
	server "gophkeeper/cmd/server/internal/grpc_server"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/cmd/server/internal/service"
	"gophkeeper/internal/config"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Init()
	DBAdress := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)

	db, err := sql.Open("pgx", DBAdress)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	cardRepo := repository.NewCardRepo(db)
	credRepo := repository.NewCredsRepo(db)

	userService := service.NewUserService(userRepo)
	cardService := service.NewCardService(cardRepo)
	credService := service.NewCredsService(credRepo)

	server := server.NewGRPCServer(userService, cardService, credService, *cfg)

	//TODO Better logs
	err = server.StartServer()
	if err != nil {
		log.Fatal(err)
	}
}
