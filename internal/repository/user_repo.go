package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/sovatharaprom/craftform-backend/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Upsert(ctx context.Context, googleID, email, name, avatarURL string) (*model.User, error) {
	id := uuid.New().String()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, google_id, email, name, avatar_url)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			email      = VALUES(email),
			name       = VALUES(name),
			avatar_url = VALUES(avatar_url),
			updated_at = NOW(6)
	`, id, googleID, email, name, avatarURL)
	if err != nil {
		return nil, err
	}

	var u model.User
	err = r.db.QueryRowContext(ctx, `
		SELECT id, google_id, email, name, avatar_url, created_at, updated_at
		FROM users WHERE google_id = ?
	`, googleID).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, google_id, email, name, avatar_url, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
