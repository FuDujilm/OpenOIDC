package memcache

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/anthropic/oidc-platform/internal/port"
	"github.com/google/uuid"
)

const (
	prefixOAuthState     = "oauth_state:"
	prefixRateLimit      = "rate_limit:"
	prefixEmailVerify    = "email_verify:"
	prefixPasswordReset  = "pwd_reset:"
	prefixUser           = "user:"
	prefixUserByEmail    = "user_email:"
	prefixSettings       = "settings:"
	prefixPublicSettings = "public_settings"
	prefixProviders      = "providers"
)

type entry struct {
	value     []byte
	expiresAt time.Time
}

func (e entry) expired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

// MemCache is an in-memory implementation of port.Cache.
// It stores entries in a map protected by a RWMutex and runs a background
// goroutine that evicts expired entries every 30 seconds.
type MemCache struct {
	mu      sync.RWMutex
	entries map[string]entry
	stopCh  chan struct{}
}

// NewMemCache creates a new MemCache and starts the background cleanup goroutine.
func NewMemCache() *MemCache {
	c := &MemCache{
		entries: make(map[string]entry),
		stopCh:  make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// Close stops the background cleanup goroutine.
func (c *MemCache) Close() error {
	close(c.stopCh)
	return nil
}

func (c *MemCache) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.evictExpired()
		}
	}
}

func (c *MemCache) evictExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
}

// ---------------------------------------------------------------------------
// Generic
// ---------------------------------------------------------------------------

func (c *MemCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[key] = entry{value: value, expiresAt: exp}
	return nil
}

func (c *MemCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, port.ErrNotFound
	}
	return e.value, nil
}

func (c *MemCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
	return nil
}

// ---------------------------------------------------------------------------
// OAuth state
// ---------------------------------------------------------------------------

func (c *MemCache) SetOAuthState(_ context.Context, state string, data []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixOAuthState+state] = entry{value: data, expiresAt: exp}
	return nil
}

func (c *MemCache) GetOAuthState(_ context.Context, state string) ([]byte, error) {
	key := prefixOAuthState + state
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, port.ErrNotFound
	}
	return e.value, nil
}

func (c *MemCache) DeleteOAuthState(_ context.Context, state string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, prefixOAuthState+state)
	return nil
}

// ---------------------------------------------------------------------------
// Rate limit
// ---------------------------------------------------------------------------

func (c *MemCache) IncrementRateLimit(_ context.Context, key string, window time.Duration) (int64, error) {
	fullKey := prefixRateLimit + key
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[fullKey]
	var count int64

	if ok && !e.expired() {
		if len(e.value) >= 8 {
			count = int64(binary.LittleEndian.Uint64(e.value))
		}
	}

	count++
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(count))

	var exp time.Time
	if window > 0 {
		// Only set new expiry if this is a fresh counter (first increment in window).
		if ok && !e.expired() {
			exp = e.expiresAt
		} else {
			exp = time.Now().Add(window)
		}
	}

	c.entries[fullKey] = entry{value: buf, expiresAt: exp}
	return count, nil
}

// ---------------------------------------------------------------------------
// Email verify tokens
// ---------------------------------------------------------------------------

func (c *MemCache) SetEmailVerifyToken(_ context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixEmailVerify+token] = entry{value: []byte(userID.String()), expiresAt: exp}
	return nil
}

func (c *MemCache) GetEmailVerifyToken(_ context.Context, token string) (uuid.UUID, error) {
	key := prefixEmailVerify + token
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return uuid.Nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return uuid.Nil, port.ErrNotFound
	}
	return uuid.Parse(string(e.value))
}

// ---------------------------------------------------------------------------
// Password reset tokens
// ---------------------------------------------------------------------------

func (c *MemCache) SetPasswordResetToken(_ context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixPasswordReset+token] = entry{value: []byte(userID.String()), expiresAt: exp}
	return nil
}

func (c *MemCache) GetPasswordResetToken(_ context.Context, token string) (uuid.UUID, error) {
	key := prefixPasswordReset + token
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return uuid.Nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return uuid.Nil, port.ErrNotFound
	}
	return uuid.Parse(string(e.value))
}

// ---------------------------------------------------------------------------
// Pattern deletion
// ---------------------------------------------------------------------------

func (c *MemCache) DeletePattern(_ context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Simple pattern matching: support wildcards at the end
	prefix := strings.TrimSuffix(pattern, "*")
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// User caching
// ---------------------------------------------------------------------------

func (c *MemCache) SetUser(_ context.Context, userID uuid.UUID, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixUser+userID.String()] = entry{value: bytes, expiresAt: exp}
	return nil
}

func (c *MemCache) GetUser(_ context.Context, userID uuid.UUID) ([]byte, error) {
	key := prefixUser + userID.String()
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, port.ErrNotFound
	}
	return e.value, nil
}

func (c *MemCache) DeleteUser(_ context.Context, userID uuid.UUID) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, prefixUser+userID.String())
	return nil
}

func (c *MemCache) SetUserByEmail(_ context.Context, email string, userID uuid.UUID, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixUserByEmail+email] = entry{value: []byte(userID.String()), expiresAt: exp}
	return nil
}

func (c *MemCache) GetUserByEmail(_ context.Context, email string) (uuid.UUID, error) {
	key := prefixUserByEmail + email
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return uuid.Nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return uuid.Nil, port.ErrNotFound
	}
	return uuid.Parse(string(e.value))
}

func (c *MemCache) DeleteUserByEmail(_ context.Context, email string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, prefixUserByEmail+email)
	return nil
}

// ---------------------------------------------------------------------------
// Settings caching
// ---------------------------------------------------------------------------

func (c *MemCache) SetPublicSettings(_ context.Context, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixPublicSettings] = entry{value: bytes, expiresAt: exp}
	return nil
}

func (c *MemCache) GetPublicSettings(_ context.Context) ([]byte, error) {
	c.mu.RLock()
	e, ok := c.entries[prefixPublicSettings]
	c.mu.RUnlock()
	if !ok {
		return nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, prefixPublicSettings)
		c.mu.Unlock()
		return nil, port.ErrNotFound
	}
	return e.value, nil
}

func (c *MemCache) DeletePublicSettings(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, prefixPublicSettings)
	return nil
}

func (c *MemCache) SetSetting(_ context.Context, key string, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixSettings+key] = entry{value: []byte(value), expiresAt: exp}
	return nil
}

func (c *MemCache) GetSetting(_ context.Context, key string) (string, error) {
	fullKey := prefixSettings + key
	c.mu.RLock()
	e, ok := c.entries[fullKey]
	c.mu.RUnlock()
	if !ok {
		return "", port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, fullKey)
		c.mu.Unlock()
		return "", port.ErrNotFound
	}
	return string(e.value), nil
}

func (c *MemCache) DeleteSetting(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, prefixSettings+key)
	return nil
}

func (c *MemCache) InvalidateAllSettings(ctx context.Context) error {
	return c.DeletePattern(ctx, prefixSettings+"*")
}

// ---------------------------------------------------------------------------
// Provider caching
// ---------------------------------------------------------------------------

func (c *MemCache) SetProviders(_ context.Context, data interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.entries[prefixProviders] = entry{value: bytes, expiresAt: exp}
	return nil
}

func (c *MemCache) GetProviders(_ context.Context) ([]byte, error) {
	c.mu.RLock()
	e, ok := c.entries[prefixProviders]
	c.mu.RUnlock()
	if !ok {
		return nil, port.ErrNotFound
	}
	if e.expired() {
		c.mu.Lock()
		delete(c.entries, prefixProviders)
		c.mu.Unlock()
		return nil, port.ErrNotFound
	}
	return e.value, nil
}

func (c *MemCache) DeleteProviders(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, prefixProviders)
	return nil
}
