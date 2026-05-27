package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

const (
	// Cache TTLs
	UserCacheTTL            = 5 * time.Minute
	SettingsCacheTTL        = 10 * time.Minute
	PublicSettingsCacheTTL  = 5 * time.Minute
	ProvidersCacheTTL       = 10 * time.Minute
)

type CacheService struct {
	cache port.Cache
}

func NewCacheService(cache port.Cache) *CacheService {
	return &CacheService{cache: cache}
}

// User caching
func (s *CacheService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	data, err := s.cache.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *CacheService) SetUser(ctx context.Context, user *domain.User) error {
	return s.cache.SetUser(ctx, user.ID, user, UserCacheTTL)
}

func (s *CacheService) InvalidateUser(ctx context.Context, userID uuid.UUID) error {
	return s.cache.DeleteUser(ctx, userID)
}

func (s *CacheService) GetUserByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	return s.cache.GetUserByEmail(ctx, email)
}

func (s *CacheService) SetUserByEmail(ctx context.Context, email string, userID uuid.UUID) error {
	return s.cache.SetUserByEmail(ctx, email, userID, UserCacheTTL)
}

func (s *CacheService) InvalidateUserByEmail(ctx context.Context, email string) error {
	return s.cache.DeleteUserByEmail(ctx, email)
}

// Public settings caching
func (s *CacheService) GetPublicSettings(ctx context.Context) (map[string]string, error) {
	data, err := s.cache.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	var settings map[string]string
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *CacheService) SetPublicSettings(ctx context.Context, settings map[string]string) error {
	return s.cache.SetPublicSettings(ctx, settings, PublicSettingsCacheTTL)
}

func (s *CacheService) InvalidatePublicSettings(ctx context.Context) error {
	return s.cache.DeletePublicSettings(ctx)
}

// Individual setting caching
func (s *CacheService) GetSetting(ctx context.Context, key string) (string, error) {
	return s.cache.GetSetting(ctx, key)
}

func (s *CacheService) SetSetting(ctx context.Context, key, value string) error {
	return s.cache.SetSetting(ctx, key, value, SettingsCacheTTL)
}

func (s *CacheService) InvalidateSetting(ctx context.Context, key string) error {
	return s.cache.DeleteSetting(ctx, key)
}

func (s *CacheService) InvalidateAllSettings(ctx context.Context) error {
	// Invalidate both individual settings and public settings cache
	if err := s.cache.InvalidateAllSettings(ctx); err != nil {
		return err
	}
	return s.cache.DeletePublicSettings(ctx)
}

// Provider caching
func (s *CacheService) GetProviders(ctx context.Context) ([]*domain.ProviderConfig, error) {
	data, err := s.cache.GetProviders(ctx)
	if err != nil {
		return nil, err
	}
	var providers []*domain.ProviderConfig
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *CacheService) SetProviders(ctx context.Context, providers []*domain.ProviderConfig) error {
	return s.cache.SetProviders(ctx, providers, ProvidersCacheTTL)
}

func (s *CacheService) InvalidateProviders(ctx context.Context) error {
	return s.cache.DeleteProviders(ctx)
}
