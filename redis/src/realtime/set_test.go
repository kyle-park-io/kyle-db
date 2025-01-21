package realtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"kyle-redis/client"
)

// go test -v -run TestTTL
func TestTTL(t *testing.T) {
	// Initialize Redis client
	rdb := client.InitRedisClient("6379")
	defer rdb.Conn().Close()

	// context
	ctx := context.Background()

	// Initialize UserManager
	userManager := NewUserManager(rdb, "active_users")
	userManager.SetTTL(ctx, 3)
	userManager.AddUser(ctx, "T")

	i := 0
	for {
		if i == 10 {
			return
		}

		ttl, err := userManager.client.TTL(ctx, "active_users").Result()
		if err != nil {
			t.Error(err)
		}
		t.Log(ttl)

		i++
		time.Sleep(1 * time.Second)
	}
}

// go test -v -run TestSimulateUserAction
func TestSimulateUserAction(t *testing.T) {
	// Initialize Redis client
	rdb := client.InitRedisClient("6379")
	defer rdb.Conn().Close()

	// context
	ctx := context.Background()

	// Initialize UserManager
	userManager := NewUserManager(rdb, "active_users")

	// Simulate user activity
	go simulateUserJoin(ctx, userManager)
	go simulateUserLeave(ctx, userManager)

	// Periodically fetch and display active users
	for {
		users, err := userManager.GetAllUsers(ctx)
		if err != nil {
			fmt.Println("Error fetching active users:", err)
		} else {
			fmt.Printf("Current active users: %v\n", users)
		}

		count, err := userManager.GetUserCount(ctx)
		if err != nil {
			fmt.Println("Error fetching user count:", err)
		} else {
			fmt.Printf("Total active users: %d\n", count)
		}

		time.Sleep(2 * time.Second)
	}
}

// Simulate users joining
func simulateUserJoin(ctx context.Context, userManager *UserManager) {
	userID := 1
	for {
		err := userManager.AddUser(ctx, fmt.Sprintf("user-%d", userID))
		if err != nil {
			fmt.Println("Error adding user:", err)
		} else {
			fmt.Printf("User joined: user-%d\n", userID)
		}
		userID++
		time.Sleep(1 * time.Second)
	}
}

// Simulate users leaving
func simulateUserLeave(ctx context.Context, userManager *UserManager) {
	for {
		userID := fmt.Sprintf("user-%d", randomUserID())
		err := userManager.RemoveUser(ctx, userID)
		if err != nil {
			fmt.Println("Error removing user:", err)
		} else {
			fmt.Printf("User left: %s\n", userID)
		}
		time.Sleep(3 * time.Second)
	}
}

// Generate a random user ID for simulation
func randomUserID() int {
	return time.Now().Second() % 10 // For simplicity, user IDs from 0 to 9
}
