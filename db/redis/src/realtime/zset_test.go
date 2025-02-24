package realtime

import (
	"context"
	"fmt"
	"kyle-redis/client"
	"testing"
	"time"
)

// go test -v -run TestSimulateUserAction2
func TestSimulateUserAction2(t *testing.T) {
	// Initialize Redis client
	rdb := client.InitRedisClient("6379")
	defer rdb.Conn().Close()

	// context
	ctx := context.Background()

	// Initialize UserManager
	userManager := NewUserManager2(rdb, "active_users2", 10)

	// Add users
	if err := userManager.AddUser(ctx, "user1"); err != nil {
		t.Error(err)
	}
	if err := userManager.AddUser(ctx, "user2"); err != nil {
		t.Error(err)
	}

	// Simulate periodic cleanup of expired users
	go func() {
		for {
			userManager.CleanUpExpiredUsers(ctx)
			time.Sleep(5 * time.Second)
		}
	}()

	// Monitor active users
	for {
		users, _ := userManager.GetAllUsers(ctx)
		count, _ := userManager.GetUserCount(ctx)
		fmt.Printf("Active users: %v (count: %d)\n", users, count)
		fmt.Println(userManager.client.ZRange(ctx, "active_users2", 0, -1))

		if err := userManager.AddUser(ctx, "user1"); err != nil {
			t.Error(err)
		}
		time.Sleep(2 * time.Second)
	}
}
