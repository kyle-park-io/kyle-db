package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kyle-redis/client"
	"kyle-redis/logger"
	"kyle-redis/realtime"
	"net"
	"strings"
	"time"
)

// How long a session stays counted without a refresh. The proxy refreshes on
// a much shorter beat; see ingress-reverse-proxy/redis/wstotcp.go. These two
// numbers used to be the same 30 seconds, which meant a session expired in
// the gap between one refresh and the next and the count visibly dropped and
// recovered every half minute.
const sessionTTL = 30 * time.Second

// How long one visitor reading one post counts as a single view: one view
// per visitor per post per day (the blog view-count spec, 2026-11-18).
const viewWindow = 24 * time.Hour

func BasicHandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		message = strings.TrimSpace(message)
		var response string
		switch message {
		case "PING":
			response = "PONG\n"
		case "HELLO":
			response = "Hello, Client!\n"
		case "TIME":
			response = fmt.Sprintf("Current time: %s\n", time.Now().Format("15:04:05"))
		case "EXIT":
			conn.Write([]byte("Goodbye!\n"))
			return
		default:
			response = "Unknown command. Available commands: PING, HELLO, TIME, EXIT\n"
		}
		if _, err := conn.Write([]byte(response)); err != nil {
			return
		}
	}
}
func RedisHandleConnection(conn net.Conn) {
	defer conn.Close()
	peer := conn.RemoteAddr().String()
	store := realtime.NewSessionStore(client.RedisClient, sessionTTL)
	views := realtime.NewViewStore(client.RedisClient, viewWindow)
	ctx := context.Background()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// A client hanging up is how every one of these connections
			// ends. It was logged at Error, once per connection, which at
			// one connection per tick was most of the log.
			if !errors.Is(err, io.EOF) {
				logger.Log.Debugw("connection closed", "peer", peer, "err", err)
			}
			return
		}
		command, arg, err := parseMessage(line)
		if err != nil {
			logger.Log.Warnw("unparseable command", "peer", peer, "err", err)
			if !write(conn, "ERR "+err.Error()+"\n") {
				return
			}
			continue
		}
		response, keepGoing := dispatch(ctx, store, views, command, arg, peer)
		if !write(conn, response) {
			return
		}
		if !keepGoing {
			return
		}
	}
}

// dispatch runs one command and returns the line to write back, plus whether
// the connection should stay open.
func dispatch(
	ctx context.Context,
	store *realtime.SessionStore,
	views *realtime.ViewStore,
	command, arg, peer string,
) (response string, keepGoing bool) {
	switch command {
	case "PING":
		return "PONG\n", true
	case "HELLO":
		return "Hello, Client!\n", true
	case "TIME":
		return fmt.Sprintf("Current time: %s\n", time.Now().Format("15:04:05")), true
	case "EXIT":
		return "Goodbye!\n", false
	case "ADDUSER":
		session, err := realtime.ParseSession(arg)
		if err != nil {
			logger.Log.Warnw("bad ADDUSER", "peer", peer, "arg", arg, "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		if err := store.Touch(ctx, session); err != nil {
			logger.Log.Errorw("could not record session", "id", session.ID, "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		logger.Log.Debugw("session touched",
			"id", session.ID,
			"country", session.Country,
			"path", session.Path,
		)
		return "OK\n", true
	case "DROPUSER":
		// Called the moment a websocket closes, so someone who leaves stops
		// being counted at once rather than lingering for a whole TTL.
		if arg == "" {
			return "ERR DROPUSER needs a session id\n", true
		}
		if err := store.Drop(ctx, arg); err != nil {
			logger.Log.Errorw("could not drop session", "id", arg, "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		logger.Log.Debugw("session dropped", "id", arg)
		return "OK\n", true
	case "REAL-TIME":
		snapshot, err := store.Snapshot(ctx)
		if err != nil {
			logger.Log.Errorw("could not read snapshot", "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		blob, err := json.Marshal(snapshot)
		if err != nil {
			logger.Log.Errorw("could not encode snapshot", "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		logCountChange(snapshot)
		return string(blob) + "\n", true
	case "VIEW":
		// One read of one blog post. The proxy has already checked that the
		// post is listed and who the visitor is; see
		// ingress-reverse-proxy/redis/views.go. Good views are not logged:
		// there is one per reader per post.
		req, err := realtime.ParseView(arg)
		if err != nil {
			logger.Log.Warnw("bad VIEW", "peer", peer, "arg", arg, "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		n, err := views.View(ctx, req.Slug, req.ID)
		if err != nil {
			logger.Log.Errorw("could not record view", "slug", req.Slug, "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		return reply(struct {
			Slug  string `json:"slug"`
			Views int64  `json:"views"`
		}{req.Slug, n})
	case "VIEWS":
		slugs, err := realtime.ParseSlugs(arg)
		if err != nil {
			logger.Log.Warnw("bad VIEWS", "peer", peer, "arg", arg, "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		counts, err := views.Counts(ctx, slugs)
		if err != nil {
			logger.Log.Errorw("could not read views", "err", err)
			return "ERR " + err.Error() + "\n", true
		}
		return reply(counts)
	default:
		return "Unknown command. Available commands: ADDUSER, DROPUSER, REAL-TIME, VIEW, VIEWS, PING, HELLO, TIME, EXIT\n", true
	}
}

// reply encodes one JSON answer line.
func reply(v any) (string, bool) {
	blob, err := json.Marshal(v)
	if err != nil {
		return "ERR " + err.Error() + "\n", true
	}
	return string(blob) + "\n", true
}

// write returns false when the connection is gone.
func write(conn net.Conn, response string) bool {
	if _, err := conn.Write([]byte(response)); err != nil {
		logger.Log.Debugw("could not write response", "err", err)
		return false
	}
	return true
}

// lastCount is what the previous snapshot said. REAL-TIME runs on a timer,
// so logging every answer buried everything else under one line per tick,
// forever, and still did not say what had changed. Log the transitions.
var lastCount = -1

func logCountChange(s realtime.Snapshot) {
	if s.Count == lastCount {
		return
	}
	previous := lastCount
	lastCount = s.Count
	logger.Log.Infow("online count changed",
		"from", previous,
		"to", s.Count,
		"countries", s.Countries,
		"pages", s.Pages,
	)
}
