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

// UserRepo структура репозитория пользователя
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo создает новый UserRepo
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create создает пользователя в бд
func (repo *UserRepo) Create(ctx context.Context, user *domain.UserCredentials) (string, error) {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			if !errors.Is(err, sql.ErrTxDone) {
				log.Printf("failed to rollback the transaction: %v", err)
			}
		}
	}()

	err = repo.isLoginFree(ctx, user.Login)
	if err != nil {
		return "", err
	}

	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		return "", err
	}

	var userID string
	err = tx.QueryRowContext(ctx,
		"INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id",
		user.Login, hashedPassword).Scan(&userID)
	if err != nil {
		return "", err
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return userID, nil
}

// GetIDByCreds получает ID пользователя по данным пользователя
func (repo *UserRepo) GetIDByCreds(ctx context.Context, user *domain.UserCredentials) (string, error) {
	var userFromDB domain.UserCredentials
	var userID string
	err := repo.db.QueryRowContext(ctx,
		"SELECT id, password FROM users WHERE login = $1", user.Login,
	).Scan(&userID, &userFromDB.Password)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", model.ErrNoUser
		}

		return "", err
	}

	isPasswordCorrect := auth.CheckPasswordHash(user.Password, userFromDB.Password)
	if isPasswordCorrect {
		return userID, nil
	}

	return "", model.ErrIncorrectPass
}

// IsLoginFree проверяет свободен ли login
func (repo *UserRepo) isLoginFree(ctx context.Context, login string) error {
	var count int
	err := repo.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE login = $1", login,
	).Scan(&count)
	if err != nil {
		return err
	}

	if count == 1 {
		return model.ErrOccupiedLogin
	} else {
		return nil
	}
}
