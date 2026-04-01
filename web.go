// ABOUTME: Web capture UI handlers for engram.
// ABOUTME: Serves the single-page capture UI and processes POST /capture requests.

package main

import (
	"context"
	"encoding/json"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"open-brain-go/brain"
)

//go:embed web/index.html
var webUI string

// RegisterWebHandlers adds the web UI and capture endpoint to the mux.
func RegisterWebHandlers(mux *http.ServeMux, a *brain.App) {
	mux.HandleFunc("/", serveWebUI())
	mux.Handle("POST /capture", authMiddleware(a, http.HandlerFunc(webCaptureHandler(a))))
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

func webCaptureHandler(a *brain.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req captureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
			http.Error(w, `{"error":"text is required"}`, http.StatusBadRequest)
			return
		}

		result, err := a.DispatchCapture(r.Context(), req.Text)
		if err != nil {
			log.Printf("dispatch error: %v", err)
			http.Error(w, `{"error":"dispatch failed"}`, http.StatusInternalServerError)
			return
		}

		toolUsed, message, err := executeDispatch(r.Context(), a, result)
		if err != nil {
			log.Printf("execute error (tool=%s): %v", result.Tool, err)
			http.Error(w, `{"error":"failed to save"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(captureResponse{Tool: toolUsed, Message: message})
	}
}

func executeDispatch(ctx context.Context, a *brain.App, result *brain.DispatchResult) (toolUsed, message string, err error) {
	switch result.Tool {

	case brain.ToolCaptureThought:
		var p brain.CaptureThoughtParams
		if err := json.Unmarshal(result.Params, &p); err != nil {
			return "", "", err
		}
		msg, err := a.SaveThought(ctx, p.Content, "web")
		return string(brain.ToolCaptureThought), msg, err

	case brain.ToolLogInteraction:
		var p brain.LogInteractionParams
		if err := json.Unmarshal(result.Params, &p); err != nil {
			return "", "", err
		}
		return executeLogInteraction(ctx, a, p)

	case brain.ToolAddContact:
		var p brain.AddContactParams
		if err := json.Unmarshal(result.Params, &p); err != nil {
			return "", "", err
		}
		err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				`INSERT INTO professional_contacts (user_id, name, company, title, notes)
				 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3, $4)`,
				p.Name, p.Company, p.Title, p.Notes,
			)
			return err
		})
		if err != nil {
			return "", "", err
		}
		return string(brain.ToolAddContact), "Added contact: " + p.Name, nil

	case brain.ToolAddMaintenanceTask:
		var p brain.AddMaintenanceTaskParams
		if err := json.Unmarshal(result.Params, &p); err != nil {
			return "", "", err
		}
		err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
			if p.NextDue != "" {
				_, err := tx.Exec(ctx,
					`INSERT INTO maintenance_tasks (user_id, name, category, location, next_due, notes)
					 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3, $4, $5)`,
					p.Name, p.Category, p.Location, p.NextDue, p.Notes,
				)
				return err
			}
			_, err := tx.Exec(ctx,
				`INSERT INTO maintenance_tasks (user_id, name, category, location, notes)
				 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3, $4)`,
				p.Name, p.Category, p.Location, p.Notes,
			)
			return err
		})
		if err != nil {
			return "", "", err
		}
		return string(brain.ToolAddMaintenanceTask), "Added maintenance task: " + p.Name, nil

	case brain.ToolAddImportantDate:
		var p brain.AddImportantDateParams
		if err := json.Unmarshal(result.Params, &p); err != nil {
			return "", "", err
		}
		if p.EventDate == "" {
			msg, err := a.SaveThought(ctx, "Important date: "+p.Title+" "+p.Notes, "web")
			return string(brain.ToolCaptureThought), msg, err
		}
		err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				`INSERT INTO important_dates (user_id, title, event_date, recurring_yearly, notes, reminder_days_before)
				 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3, $4, 7)`,
				p.Title, p.EventDate, p.RecurringYearly, p.Notes,
			)
			return err
		})
		if err != nil {
			return "", "", err
		}
		return string(brain.ToolAddImportantDate), "Added date: " + p.Title + " (" + p.EventDate + ")", nil

	case brain.ToolAddJobPosting:
		var p brain.AddJobPostingParams
		if err := json.Unmarshal(result.Params, &p); err != nil {
			return "", "", err
		}
		return executeAddJobPosting(ctx, a, p)

	default:
		msg, err := a.SaveThought(ctx, string(result.Params), "web")
		return string(brain.ToolCaptureThought), msg, err
	}
}

func executeLogInteraction(ctx context.Context, a *brain.App, p brain.LogInteractionParams) (string, string, error) {
	var contactID string
	var contactName string

	// Look up contact by name.
	_ = a.WithUserTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT id::text, name FROM professional_contacts
			 WHERE name ILIKE '%' || $1 || '%'
			 ORDER BY last_contacted DESC NULLS LAST LIMIT 1`,
			p.PersonName,
		).Scan(&contactID, &contactName)
	})

	if contactID == "" {
		// Fall back to a thought.
		content := fmt.Sprintf("Met with %s (%s): %s", p.PersonName, p.InteractionType, p.Summary)
		if p.FollowUpNotes != "" {
			content += " | Follow-up: " + p.FollowUpNotes
		}
		_, err := a.SaveThought(ctx, content, "web")
		return string(brain.ToolCaptureThought),
			fmt.Sprintf("No contact found for '%s' — saved as note", p.PersonName),
			err
	}

	err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO contact_interactions
			   (user_id, contact_id, interaction_type, summary, follow_up_needed, follow_up_notes, interaction_date)
			 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3, $4, $5, $6)`,
			contactID, p.InteractionType, p.Summary, p.FollowUpNeeded, p.FollowUpNotes, time.Now(),
		)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`UPDATE professional_contacts SET last_contacted = NOW() WHERE id = $1`,
			contactID,
		)
		return err
	})
	if err != nil {
		return "", "", err
	}

	msg := "Logged " + p.InteractionType + " with " + contactName
	if p.FollowUpNeeded && p.FollowUpNotes != "" {
		msg += " | Follow-up: " + p.FollowUpNotes
	}
	return string(brain.ToolLogInteraction), msg, nil
}

func executeAddJobPosting(ctx context.Context, a *brain.App, p brain.AddJobPostingParams) (string, string, error) {
	var companyID string

	// Find or create the company.
	err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id::text FROM job_companies WHERE name ILIKE $1 LIMIT 1`,
			p.CompanyName,
		).Scan(&companyID)
		if err == nil {
			return nil // found
		}
		// Not found — create it.
		return tx.QueryRow(ctx,
			`INSERT INTO job_companies (user_id, name)
			 VALUES (current_setting('app.current_user_id')::uuid, $1)
			 RETURNING id::text`,
			p.CompanyName,
		).Scan(&companyID)
	})
	if err != nil {
		return "", "", fmt.Errorf("find/create company: %w", err)
	}

	err = a.WithUserTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO job_postings (user_id, company_id, title, url, notes)
			 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3, $4)`,
			companyID, p.Title, p.URL, p.Notes,
		)
		return err
	})
	if err != nil {
		return "", "", err
	}
	return string(brain.ToolAddJobPosting), "Added job posting: " + p.Title + " at " + p.CompanyName, nil
}
