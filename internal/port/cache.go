package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error

	SetOAuthState(ctx context.Context, state string, data []byte, ttl time.Duration) error
	GetOAuthState(ctx context.Context, state string) ([]byte, error)
	DeleteOAuthState(ctx context.Context, state string) error

	IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error)

	SetEmailVerifyToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	GetEmailVerifyToken(ctx context.Context, token string) (uuid.UUID, error)

	SetPasswordResetToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	GetPasswordResetToken(ctx context.Context, token string) (uuid.UUID, error)

	// User caching
	SetUser(ctx context.Context, userID uuid.UUID, data interface{}, ttl time.Duration) error
	GetUser(ctx context.Context, userID uuid.UUID) ([]byte, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	SetUserByEmail(ctx context.Context, email string, userID uuid.UUID, ttl time.Duration) error
	GetUserByEmail(ctx context.Context, email string) (uuid.UUID, error)
	DeleteUserByEmail(ctx context.Context, email string) error

	// Settings caching
	SetPublicSettings(ctx context.Context, data interface{}, ttl time.Duration) error
	GetPublicSettings(ctx context.Context) ([]byte, error)
	DeletePublicSettings(ctx context.Context) error
	SetSetting(ctx context.Context, key string, value string, ttl time.Duration) error
	GetSetting(ctx context.Context, key string) (string, error)
	DeleteSetting(ctx context.Context, key string) error
	InvalidateAllSettings(ctx context.Context) error

	// Provider caching
	SetProviders(ctx context.Context, data interface{}, ttl time.Duration) error
	GetProviders(ctx context.Context) ([]byte, error)
	DeleteProviders(ctx context.Context) error
}
