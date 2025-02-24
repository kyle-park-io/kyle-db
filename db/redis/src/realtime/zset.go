package realtime

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// UserManager2 manages active users using Redis ZSet
type UserManager2 struct {
	client *redis.Client
	key    string // Redis key for the active user ZSet
	ttl    int    // Default TTL for users in seconds
}

// NewUserManager2 creates a new UserManager2
func NewUserManager2(redisClient *redis.Client, key string, ttl int) *UserManager2 {
	return &UserManager2{
		client: redisClient,
		key:    key,
		ttl:    ttl,
	}
}

// AddUser adds a user to the active user ZSet with a TTL
func (m *UserManager2) AddUser(ctx context.Context, userID string) error {
	expiration := time.Now().Add(time.Duration(m.ttl) * time.Second).Unix()
	return m.client.ZAdd(ctx, m.key, redis.Z{
		Score:  float64(expiration),
		Member: userID,
	}).Err()
}

// RemoveUser removes a user from the active user ZSet
func (m *UserManager2) RemoveUser(ctx context.Context, userID string) error {
	return m.client.ZRem(ctx, m.key, userID).Err()
}

// GetAllUsers retrieves all active users from the ZSet
func (m *UserManager2) GetAllUsers(ctx context.Context) ([]string, error) {
	now := float64(time.Now().Unix())
	return m.client.ZRangeByScore(ctx, m.key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", now), // Only get non-expired users
		Max: "+inf",
	}).Result()
}

// GetUserCount returns the number of active users in the ZSet
func (m *UserManager2) GetUserCount(ctx context.Context) (int64, error) {
	now := float64(time.Now().Unix())
	return m.client.ZCount(ctx, m.key, fmt.Sprintf("%f", now), "+inf").Result()
}

// CleanUpExpiredUsers removes expired users from the ZSet
func (m *UserManager2) CleanUpExpiredUsers(ctx context.Context) error {
	now := float64(time.Now().Unix())
	_, err := m.client.ZRemRangeByScore(ctx, m.key, "-inf", fmt.Sprintf("%f", now)).Result()
	return err
}
