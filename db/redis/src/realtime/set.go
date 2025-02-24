package realtime

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// UserManager manages active users using Redis
type UserManager struct {
	client *redis.Client
	key    string // Redis key for the active user set
}

// NewUserManager creates a new UserManager
func NewUserManager(redisClient *redis.Client, key string) *UserManager {
	return &UserManager{
		client: redisClient,
		key:    key,
	}
}

func (m *UserManager) SetTTL(ctx context.Context, ttl int) error {
	return m.client.Expire(ctx, m.key, time.Duration(ttl)*time.Second).Err()
}

// AddUser adds a user to the active user set
func (m *UserManager) AddUser(ctx context.Context, userID string) error {
	return m.client.SAdd(ctx, m.key, userID).Err()
}

// RemoveUser removes a user from the active user set
func (m *UserManager) RemoveUser(ctx context.Context, userID string) error {
	return m.client.SRem(ctx, m.key, userID).Err()
}

// GetAllUsers retrieves all active users
func (m *UserManager) GetAllUsers(ctx context.Context) ([]string, error) {
	return m.client.SMembers(ctx, m.key).Result()
}

// GetUserCount returns the number of active users
func (m *UserManager) GetUserCount(ctx context.Context) (int64, error) {
	return m.client.SCard(ctx, m.key).Result()
}
