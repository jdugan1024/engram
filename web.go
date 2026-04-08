// ABOUTME: Web capture UI handlers for engram.
// ABOUTME: Serves the single-page capture UI and processes POST /capture requests.

package main

import (
	"encoding/json"
	_ "embed"
	"fmt"
	"log"
	"net/http"

	"open-brain-go/brain"
	"open-brain-go/brain/service"
)

//go:embed web/index.html
var webUI string

// RegisterWebHandlers adds the web UI and capture endpoint to the mux.
func RegisterWebHandlers(mux *http.ServeMux, a *brain.App, es *service.EntryService) {
	mux.HandleFunc("/", serveWebUI())
	mux.Handle("POST /capture", authMiddleware(a, http.HandlerFunc(webCaptureHandler(a, es))))
}

func serveWebUI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, webUI)
	}
}

type captureRequest struct {
	Text string `json:"text"`
}

type captureResponse struct {
	Tool    string `json:"tool"`
	Message string `json:"message"`
}

func webCaptureHandler(a *brain.App, es *service.EntryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req captureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
			http.Error(w, `{"error":"text is required"}`, http.StatusBadRequest)
			return
		}

		cr, err := es.Capture(r.Context(), req.Text, "web")
		if err != nil {
			log.Printf("entry capture error: %v", err)
			http.Error(w, `{"error":"failed to save"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(captureResponse{Tool: cr.RecordType, Message: cr.Message})
	}
}
