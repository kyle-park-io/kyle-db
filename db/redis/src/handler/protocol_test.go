package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"kyle-redis/client"
	"kyle-redis/logger"
	"kyle-redis/realtime"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// A conversation with the real TCP handler over a real socket, against a
// real Redis. This is the layer the proxy actually talks to, and none of it
// was covered: the previous tests reached the Redis helpers directly and the
// command parser not at all.
type conn struct {
	t      *testing.T
	socket net.Conn
	reader *bufio.Reader
}

func (c *conn) say(format string, args ...any) string {
	c.t.Helper()
	if err := c.socket.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		c.t.Fatal(err)
	}
	if _, err := fmt.Fprintf(c.socket, format+"\n", args...); err != nil {
		c.t.Fatalf("writing: %v", err)
	}
	line, err := c.reader.ReadString('\n')
	if err != nil {
		c.t.Fatalf("reading the reply: %v", err)
	}
	return line[:len(line)-1]
}

func (c *conn) snapshot() realtime.Snapshot {
	c.t.Helper()
	var s realtime.Snapshot
	reply := c.say("REAL-TIME")
	if err := json.Unmarshal([]byte(reply), &s); err != nil {
		c.t.Fatalf("REAL-TIME did not answer with JSON: %q (%v)", reply, err)
	}
	return s
}

// serve starts the handler on a loopback listener and dials it.
func serve(t *testing.T) *conn {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("set REDIS_ADDR to run the protocol tests")
	}

	if logger.Log == nil {
		viper.Set("LOG_PATH", t.TempDir())
		logger.InitLogger()
	}

	// Its own database; see the note in realtime/session_redis_test.go.
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 15})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("no Redis at %s: %v", addr, err)
	}
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	client.RedisClient = rdb

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			go RedisHandleConnection(c)
		}
	}()

	socket, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		socket.Close()
		listener.Close()
		rdb.Close()
	})

	return &conn{t: t, socket: socket, reader: bufio.NewReader(socket)}
}

func TestProtocolCountsAVisitorOnceHoweverOftenTheyRefresh(t *testing.T) {
	// One reader loading one page used to arrive as roughly twenty ADDUSERs
	// with twenty different ids, because the id was the Cloudflare edge
	// address of each individual request.
	c := serve(t)

	for i := 0; i < 20; i++ {
		if reply := c.say(`ADDUSER {"id":"aaa","country":"KR","browser":"Chrome","path":"/blog"}`); reply != "OK" {
			t.Fatalf("ADDUSER %d answered %q", i, reply)
		}
	}

	if got := c.snapshot(); got.Count != 1 {
		t.Errorf("count = %d after 20 refreshes from one visitor, want 1", got.Count)
	}
}

func TestProtocolReportsTheBreakdowns(t *testing.T) {
	c := serve(t)

	c.say(`ADDUSER {"id":"a","country":"KR","browser":"Chrome","os":"macOS","path":"/blog"}`)
	c.say(`ADDUSER {"id":"b","country":"KR","browser":"Safari","os":"iOS","path":"/blog"}`)
	c.say(`ADDUSER {"id":"c","country":"US","browser":"Firefox","os":"Linux","path":"/"}`)

	got := c.snapshot()
	if got.Count != 3 {
		t.Fatalf("count = %d, want 3", got.Count)
	}
	if len(got.Countries) != 2 || got.Countries[0].Key != "KR" || got.Countries[0].N != 2 {
		t.Errorf("countries = %+v", got.Countries)
	}
	if len(got.Pages) != 2 || got.Pages[0].Key != "/blog" || got.Pages[0].N != 2 {
		t.Errorf("pages = %+v", got.Pages)
	}
}

func TestProtocolDropsAVisitorOnRequest(t *testing.T) {
	c := serve(t)

	c.say(`ADDUSER {"id":"a","country":"KR"}`)
	c.say(`ADDUSER {"id":"b","country":"KR"}`)
	if reply := c.say("DROPUSER b"); reply != "OK" {
		t.Fatalf("DROPUSER answered %q", reply)
	}

	if got := c.snapshot(); got.Count != 1 {
		t.Errorf("count = %d after dropping one of two, want 1", got.Count)
	}
}

func TestProtocolStillAcceptsTheOldADDUSER(t *testing.T) {
	// The proxy and this server are separate images, so during a rollout one
	// of them is the old one and still sends a bare address.
	c := serve(t)

	if reply := c.say("ADDUSER 104.23.251.104:13965"); reply != "OK" {
		t.Fatalf("the old ADDUSER form answered %q", reply)
	}
	got := c.snapshot()
	if got.Count != 1 {
		t.Errorf("count = %d, want 1", got.Count)
	}
	if len(got.Countries) != 0 {
		t.Errorf("a legacy session has no country to report: %+v", got.Countries)
	}
}

func TestProtocolAnswersBadInputWithoutClosingTheConnection(t *testing.T) {
	// The counter server holds one connection per proxy call. A malformed
	// line must not cost the caller its connection, or a single bad frame
	// would stop the count until the next dial.
	c := serve(t)

	if reply := c.say(`ADDUSER {"country":"KR"}`); reply[:3] != "ERR" {
		t.Errorf("an id-less ADDUSER answered %q, want an ERR", reply)
	}
	if reply := c.say("NONSENSE"); reply[:7] != "Unknown" {
		t.Errorf("an unknown command answered %q", reply)
	}
	// Still usable.
	if reply := c.say(`ADDUSER {"id":"a","country":"KR"}`); reply != "OK" {
		t.Errorf("the connection did not survive: %q", reply)
	}
	if got := c.snapshot(); got.Count != 1 {
		t.Errorf("count = %d, want 1", got.Count)
	}
}

func TestProtocolReportsNobodyAsAnEmptySnapshot(t *testing.T) {
	c := serve(t)

	reply := c.say("REAL-TIME")
	if reply != `{"count":0,"countries":[],"pages":[]}` {
		t.Errorf("an empty site answered %q", reply)
	}
}

func TestViewIsCountedOncePerVisitor(t *testing.T) {
	c := serve(t)

	for i := 0; i < 2; i++ {
		if got := c.say(`VIEW {"slug":"rfq-deep-dive","id":"v1"}`); got != `{"slug":"rfq-deep-dive","views":1}` {
			t.Fatalf("VIEW %d from v1 answered %q", i, got)
		}
	}
	if got := c.say(`VIEW {"slug":"rfq-deep-dive","id":"v2"}`); got != `{"slug":"rfq-deep-dive","views":2}` {
		t.Errorf("VIEW from v2 answered %q", got)
	}
	if got := c.say("VIEWS rfq-deep-dive,chain-communicator-v2"); got != `{"chain-communicator-v2":0,"rfq-deep-dive":2}` {
		t.Errorf("VIEWS answered %q", got)
	}
}

func TestBadViewCommandsAnswerErr(t *testing.T) {
	c := serve(t)

	for _, line := range []string{
		"VIEW",
		"VIEW nonsense",
		`VIEW {"slug":"../x","id":"v1"}`,
		"VIEWS",
		"VIEWS a,,b",
	} {
		if got := c.say("%s", line); !strings.HasPrefix(got, "ERR ") {
			t.Errorf("%q answered %q, want an ERR line", line, got)
		}
	}
	if got := c.say("PING"); got != "PONG" {
		t.Errorf("after the bad commands PING answered %q", got)
	}
}
