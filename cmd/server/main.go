package main

import (
	"database/sql"
	server "gophkeeper/cmd/server/internal/grpc_server"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/cmd/server/internal/service"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, err := sql.Open("pgx", "postgres://noedaka:admin@localhost:5432/gophkeeper?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	service := service.NewUserService(userRepo)
	server := server.NewGRPCServer(service)

	//TODO Better logs
	err = server.StartServer()
	if err != nil {
		log.Fatal(err)
	}
}
