package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	prefixOAuthState    = "oauth_state:"
	prefixRateLimit     = "rate_limit:"
	prefixEmailVerify   = "email_verify:"
	prefixPasswordReset = "pwd_reset:"
	prefixUser          = "user:"
	prefixUserByEmail   = "user_email:"
	prefixSettings      = "settings:"
	prefixPublicSettings = "public_settings"
	prefixProviders     = "providers"
)

type Cache struct {
	client *redis.Client
}

func NewCache(ctx context.Context, cfg config.RedisConfig) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Cache{client: client}, nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Client() *redis.Client {
	return c.client
}

// ---------------------------------------------------------------------------
// Generic
// ---------------------------------------------------------------------------

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// ---------------------------------------------------------------------------
// User cache
// ---------------------------------------------------------------------------

func (c *Cache) SetUser(ctx context.Context, userID uuid.UUID, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, prefixUser+userID.String(), bytes, ttl).Err()
}

func (c *Cache) GetUser(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	data, err := c.client.Get(ctx, prefixUser+userID.String()).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (c *Cache) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return c.client.Del(ctx, prefixUser+userID.String()).Err()
}

func (c *Cache) SetUserByEmail(ctx context.Context, email string, userID uuid.UUID, ttl time.Duration) error {
	return c.client.Set(ctx, prefixUserByEmail+email, userID.String(), ttl).Err()
}

func (c *Cache) GetUserByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	val, err := c.client.Get(ctx, prefixUserByEmail+email).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, port.ErrNotFound
		}
		return uuid.Nil, err
	}
	return uuid.Parse(val)
}

func (c *Cache) DeleteUserByEmail(ctx context.Context, email string) error {
	return c.client.Del(ctx, prefixUserByEmail+email).Err()
}

// ---------------------------------------------------------------------------
// Settings cache
// ---------------------------------------------------------------------------

func (c *Cache) SetPublicSettings(ctx context.Context, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, prefixPublicSettings, bytes, ttl).Err()
}

func (c *Cache) GetPublicSettings(ctx context.Context) ([]byte, error) {
	data, err := c.client.Get(ctx, prefixPublicSettings).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (c *Cache) DeletePublicSettings(ctx context.Context) error {
	return c.client.Del(ctx, prefixPublicSettings).Err()
}

func (c *Cache) SetSetting(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, prefixSettings+key, value, ttl).Err()
}

func (c *Cache) GetSetting(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, prefixSettings+key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", port.ErrNotFound
		}
		return "", err
	}
	return val, nil
}

func (c *Cache) DeleteSetting(ctx context.Context, key string) error {
	return c.client.Del(ctx, prefixSettings+key).Err()
}

func (c *Cache) InvalidateAllSettings(ctx context.Context) error {
	return c.DeletePattern(ctx, prefixSettings+"*")
}

// ---------------------------------------------------------------------------
// Provider cache
// ---------------------------------------------------------------------------

func (c *Cache) SetProviders(ctx context.Context, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, prefixProviders, bytes, ttl).Err()
}

func (c *Cache) GetProviders(ctx context.Context) ([]byte, error) {
	data, err := c.client.Get(ctx, prefixProviders).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (c *Cache) DeleteProviders(ctx context.Context) error {
	return c.client.Del(ctx, prefixProviders).Err()
}

// ---------------------------------------------------------------------------
// OAuth state
// ---------------------------------------------------------------------------

func (c *Cache) SetOAuthState(ctx context.Context, state string, data []byte, ttl time.Duration) error {
	return c.client.Set(ctx, prefixOAuthState+state, data, ttl).Err()
}

func (c *Cache) GetOAuthState(ctx context.Context, state string) ([]byte, error) {
	data, err := c.client.Get(ctx, prefixOAuthState+state).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, port.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (c *Cache) DeleteOAuthState(ctx context.Context, state string) error {
	return c.client.Del(ctx, prefixOAuthState+state).Err()
}

// ---------------------------------------------------------------------------
// Rate limit
// ---------------------------------------------------------------------------

func (c *Cache) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	fullKey := prefixRateLimit + key
	pipe := c.client.TxPipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// ---------------------------------------------------------------------------
// Email verify tokens
// ---------------------------------------------------------------------------

func (c *Cache) SetEmailVerifyToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	return c.client.Set(ctx, prefixEmailVerify+token, userID.String(), ttl).Err()
}

func (c *Cache) GetEmailVerifyToken(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := c.client.Get(ctx, prefixEmailVerify+token).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, port.ErrNotFound
		}
		return uuid.Nil, err
	}
	return uuid.Parse(val)
}

// ---------------------------------------------------------------------------
// Password reset tokens
// ---------------------------------------------------------------------------

func (c *Cache) SetPasswordResetToken(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	return c.client.Set(ctx, prefixPasswordReset+token, userID.String(), ttl).Err()
}

func (c *Cache) GetPasswordResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := c.client.Get(ctx, prefixPasswordReset+token).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, port.ErrNotFound
		}
		return uuid.Nil, err
	}
	return uuid.Parse(val)
}
