// Command server is the Terminus web application: configuration, the
// service layer over PostgreSQL, and the HTTP shell (httpapi).
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"stoptime/internal/app"
	"stoptime/internal/httpapi"
	"stoptime/internal/store"
)

// devSessionKey is the fixed development default (architecture.md:
// required in production, a fixed dev default otherwise).
const devSessionKey = "dev-insecure-session-key-change-me-32b"

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	sessionKey := []byte(os.Getenv("SESSION_KEY"))
	if len(sessionKey) < 32 {
		log.Print("WARNING: SESSION_KEY unset or under 32 bytes — using the fixed dev default; set a real key before serving beyond this machine")
		sessionKey = []byte(devSessionKey)
	}

	ctx := context.Background()
	st, err := store.Open(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	svc := app.New(st, sessionKey)
	server, err := httpapi.New(svc, sessionKey)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("terminus listening on http://%s", addr)
	if err := http.ListenAndServe(addr, server.Router()); err != nil {
		log.Fatal(err)
	}
}
