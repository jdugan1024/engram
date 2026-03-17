// ABOUTME: Open Brain MCP server entry point.
// ABOUTME: Wires shared infrastructure, core tools, and extensions into a single HTTP server.

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"open-brain-go/brain"
	"open-brain-go/core"
	"open-brain-go/extensions/calendar"
	"open-brain-go/extensions/crm"
	"open-brain-go/extensions/household"
	"open-brain-go/extensions/jobhunt"
	"open-brain-go/extensions/maintenance"
	"open-brain-go/extensions/meals"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	openRouterKey := os.Getenv("OPENROUTER_API_KEY")
	if openRouterKey == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	app, err := brain.New(ctx, dbURL, openRouterKey)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	defer app.Pool.Close()
	log.Println("Connected to database")

	s := server.NewMCPServer("open-brain", "1.0.0")

	core.Register(s, app)
	household.Register(s, app)
	maintenance.Register(s, app)
	calendar.Register(s, app)
	meals.Register(s, app)
	crm.Register(s, app)
	jobhunt.Register(s, app)

	mcpHandler := server.NewStreamableHTTPServer(s)

	mux := http.NewServeMux()
	mux.Handle("/", authMiddleware(app, mcpHandler))

	log.Printf("Open Brain MCP server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// authMiddleware resolves the caller's user ID from the access key header or query param
// and injects it into the request context for RLS-scoped DB transactions.
func authMiddleware(a *brain.App, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-brain-key")
		if key == "" {
			key = r.URL.Query().Get("key")
		}
		if key == "" {
			http.Error(w, `{"error":"missing access key"}`, http.StatusUnauthorized)
			return
		}

		var userID string
		err := a.Pool.QueryRow(r.Context(),
			"SELECT id::text FROM mcp_users WHERE access_key = $1", key,
		).Scan(&userID)
		if err != nil {
			http.Error(w, `{"error":"invalid access key"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), brain.CtxUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
