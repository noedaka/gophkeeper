package repository

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/auth"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/model"
	"log"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create создает пользователя в бд
func (repo *UserRepo) Create(ctx context.Context, user *domain.UserCredentials) (int, error) {
	tx, err := repo.db.BeginTx(ctx, nil)
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

	isFree, err := repo.isLoginFree(ctx, user.Login)
	if err != nil {
		return 0, err
	}

	if !isFree {
		return 0, model.ErrOccupiedLogin
	}

	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		return 0, err
	}

	var userID int
	err = tx.QueryRowContext(ctx,
		"INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id",
		user.Login, hashedPassword).Scan(&userID)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return userID, nil
}

// GetIDByCreds получает ID пользователя по данным пользователя
func (repo *UserRepo) GetIDByCreds(ctx context.Context, user *domain.UserCredentials) (int, error) {
	var userFromDB domain.UserCredentials
	var userID int
	err := repo.db.QueryRowContext(ctx,
		"SELECT id, password FROM users WHERE login = $1", user.Login,
	).Scan(&userID, &userFromDB.Password)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, model.ErrNoUser
		}

		return 0, err
	}

	isPasswordCorrect := auth.CheckPasswordHash(user.Password, userFromDB.Password)
	if isPasswordCorrect {
		return userID, nil
	}

	return 0, model.ErrIncorrectPass
}

func (repo *UserRepo) isLoginFree(ctx context.Context, login string) (bool, error) {
	var count int
	err := repo.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE login = $1", login,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	if count == 1 {
		return false, nil
	} else {
		return true, nil
	}
}
