// Command server is the StopTime web application.
//
// T1 scope: configuration, routing, and the /healthz probe. The pgx pool,
// service wiring, and screens land with their cards (T4 onward); the seam
// they plug into is newRouter below.
package main

import (
	"log"
	"net/http"
	"os"
)

// listenAddr reads LISTEN_ADDR, defaulting to 127.0.0.1:8080
// (docs/spec/architecture.md, "Environment and tooling").
func listenAddr() string {
	if addr := os.Getenv("LISTEN_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:8080"
}

// newRouter wires paths to handlers. Handlers stay thin; business logic
// lives in internal/app services (docs/spec/architecture.md, layering rules).
func newRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func main() {
	addr := listenAddr()
	log.Printf("stoptime listening on http://%s", addr)
	if err := http.ListenAndServe(addr, newRouter()); err != nil {
		log.Fatal(err)
	}
}
