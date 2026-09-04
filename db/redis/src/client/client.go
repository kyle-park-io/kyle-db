package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"kyle-redis/logger"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

var (
	RedisClient *redis.Client
	TCPClient   net.Conn
)

// How long to wait for Redis at startup, and the ceiling on the backoff
// between attempts. Redis runs as a sidecar in the same pod, so on a cold
// start this server is usually up first; a minute is far longer than the
// sidecar has ever taken and still short enough that a genuinely broken
// Redis restarts the pod rather than leaving it up and useless.
const (
	redisStartupBudget = 60 * time.Second
	redisRetryCeiling  = 2 * time.Second
	redisPingTimeout   = 2 * time.Second
)

func InitRedisClient(port string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     ":" + port,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// InitTCPClient opens a raw socket to Redis, speaking RESP by hand. Only the
// experiments under protocol/tcp use it; the server itself talks through
// RedisClient. It is not opened at startup - see Init.
func InitTCPClient(port string) net.Conn {
	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		panic(err)
	}
	return conn
}

// Init prepares the Redis client and waits for Redis to answer.
//
// It used to also open a raw TCP socket and hold it for the life of the
// process. Nothing read from it, but it was dialled eagerly and panicked
// when it failed - so when this container started before its Redis sidecar,
// which is most cold starts, the process died on the spot and Kubernetes
// restarted it until the timing happened to work out. Observed: two restarts
// on a routine rollout.
//
// go-redis connects lazily and pools, so it cannot fail here. That leaves
// the opposite problem: without a check the server would accept connections
// and fail every command until Redis appeared. So wait for a PING, and
// report progress while waiting.
func Init() error {
	port := viper.GetString("port.redis")
	if port == "" {
		return errors.New("port.redis is not configured")
	}

	RedisClient = InitRedisClient(port)
	_, err := waitForRedis(RedisClient, redisStartupBudget)
	return err
}

// waitForRedis pings until Redis answers or the budget runs out, and returns
// how many attempts it took. The count is returned rather than only logged
// so a test can prove the loop retries: against a real Redis a retry never
// happens, and a hand-written stub cannot stand in for one, because go-redis
// opens with a RESP3 HELLO and reads a map reply.
func waitForRedis(rdb *redis.Client, budget time.Duration) (int, error) {
	deadline := time.Now().Add(budget)
	delay := 250 * time.Millisecond

	for attempt := 1; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
		err := rdb.Ping(ctx).Err()
		cancel()

		if err == nil {
			if attempt > 1 {
				logger.Log.Infow("redis is up", "attempts", attempt)
			}
			return attempt, nil
		}

		if time.Now().After(deadline) {
			return attempt, fmt.Errorf("redis did not answer within %s: %w", budget, err)
		}

		logger.Log.Infow("waiting for redis", "attempt", attempt, "retry_in", delay.String(), "err", err)
		time.Sleep(delay)
		if delay < redisRetryCeiling {
			delay *= 2
		}
	}
}
