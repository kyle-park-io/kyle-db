package client

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"kyle-redis/logger"

	"github.com/spf13/viper"
)

func init() {
	viper.Set("LOG_PATH", os.TempDir())
	logger.InitLogger()
}

func TestWaitForRedisReturnsOnceRedisAnswers(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("set REDIS_ADDR to run this")
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}

	rdb := InitRedisClient(port)
	defer rdb.Close()

	attempts, err := waitForRedis(rdb, 10*time.Second)
	if err != nil {
		t.Fatalf("a reachable redis should be reported as up: %v", err)
	}
	if attempts != 1 {
		t.Errorf("a redis that is already up should take one attempt, took %d", attempts)
	}
}

func TestWaitForRedisGivesUpWithAnErrorRatherThanPanicking(t *testing.T) {
	// The whole point of this change: the process used to die on the spot
	// when Redis was not up yet, which on a cold start is most of the time,
	// and Kubernetes restarted it until the timing worked out. Nothing here
	// may panic, and it has to come back inside its budget.
	rdb := InitRedisClient("1") // nothing listens on port 1
	defer rdb.Close()

	started := time.Now()
	attempts, err := waitForRedis(rdb, 1500*time.Millisecond)
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("expected an error for an unreachable redis")
	}
	// It has to have kept trying rather than given up on the first refusal,
	// which is the whole point: the sidecar arrives a moment late.
	if attempts < 2 {
		t.Errorf("gave up after %d attempt(s); it should retry", attempts)
	}
	if !strings.Contains(err.Error(), "did not answer") {
		t.Errorf("the error should say what happened, got: %v", err)
	}
	// The budget plus one ping timeout is the most it can honestly take.
	if elapsed > 1500*time.Millisecond+redisPingTimeout+time.Second {
		t.Errorf("took %s, which overran the budget", elapsed)
	}
}

func TestInitDoesNotOpenTheUnusedRawSocket(t *testing.T) {
	// InitTCPClient panics when it cannot dial, and nothing in the serving
	// path reads from the socket it opens. Init must not call it.
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("set REDIS_ADDR to run this")
	}
	_, port, _ := net.SplitHostPort(addr)
	viper.Set("port.redis", port)

	TCPClient = nil
	if err := Init(); err != nil {
		t.Fatalf("Init against a live redis: %v", err)
	}
	if TCPClient != nil {
		t.Error("Init opened the raw socket again")
	}
	if RedisClient == nil {
		t.Error("Init did not build the client the server uses")
	}
}
