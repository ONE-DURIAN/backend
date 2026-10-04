package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"community-backend/internal/domain"
	"community-backend/pkg/database"
	"community-backend/pkg/redisclient"
	"community-backend/pkg/uid"
)

type Repository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	SaveRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error
	GetUserIDByRefreshToken(ctx context.Context, token string) (string, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}

type authRepo struct {
	db    *database.DB
	redis *redisclient.Client
}

func NewRepository(db *database.DB, redis *redisclient.Client) Repository {
	return &authRepo{
		db:    db,
		redis: redis,
	}
}

func (r *authRepo) CreateUser(ctx context.Context, u *domain.User) error {
	if u.ID == "" {
		u.ID = uid.NewV7()
	}

	query := `
		INSERT INTO users (id, phone_number, email, password_hash, full_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	u.IsActive = true

	err := r.db.QueryRowContext(ctx, query,
		u.ID,
		u.PhoneNumber,
		u.Email,
		u.PasswordHash,
		u.FullName,
		u.Role,
		u.IsActive,
		u.CreatedAt,
		u.UpdatedAt,
	).Scan(&u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *authRepo) GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	query := `
		SELECT id, phone_number, email, password_hash, full_name, avatar_url, role, national_id, is_active, created_at, updated_at
		FROM users
		WHERE (phone_number = $1 OR email = $1) AND deleted_at IS NULL
		LIMIT 1
	`
	var u domain.User
	var phone, email, avatar, nationalID sql.NullString

	err := r.db.QueryRowContext(ctx, query, identifier).Scan(
		&u.ID,
		&phone,
		&email,
		&u.PasswordHash,
		&u.FullName,
		&avatar,
		&u.Role,
		&nationalID,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}

	if phone.Valid {
		u.PhoneNumber = &phone.String
	}
	if email.Valid {
		u.Email = &email.String
	}
	if avatar.Valid {
		u.AvatarURL = &avatar.String
	}
	if nationalID.Valid {
		u.NationalID = &nationalID.String
	}

	return &u, nil
}

func (r *authRepo) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, phone_number, email, full_name, avatar_url, role, national_id, is_active, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	var u domain.User
	var phone, email, avatar, nationalID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&phone,
		&email,
		&u.FullName,
		&avatar,
		&u.Role,
		&nationalID,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}

	if phone.Valid {
		u.PhoneNumber = &phone.String
	}
	if email.Valid {
		u.Email = &email.String
	}
	if avatar.Valid {
		u.AvatarURL = &avatar.String
	}
	if nationalID.Valid {
		u.NationalID = &nationalID.String
	}

	return &u, nil
}

func (r *authRepo) SaveRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	return r.redis.Set(ctx, key, userID, ttl).Err()
}

func (r *authRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", token)
	userID, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		return "", errors.New("invalid or expired refresh token")
	}
	return userID, nil
}

func (r *authRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	return r.redis.Del(ctx, key).Err()
}
