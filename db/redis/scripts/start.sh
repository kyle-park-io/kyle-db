#!/bin/sh

# exec, so the server is PID 1 and gets SIGTERM itself. As a child of sh it
# never saw it: sh does not pass signals on, and a pod stop waited out the
# whole 30s grace period before the kill (2026-11-19, docker stop 10.2s).
exec /app/app
