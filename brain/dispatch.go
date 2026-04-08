// ABOUTME: LLM-based tool dispatch for the web capture UI.
// ABOUTME: Routes free-form text to the right tool via a single OpenRouter call.

package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DispatchTool names the tool the LLM chose to invoke.
type DispatchTool string

const (
	ToolCaptureThought     DispatchTool = "capture_thought"
	ToolLogInteraction     DispatchTool = "log_interaction"
	ToolAddContact         DispatchTool = "add_professional_contact"
	ToolAddMaintenanceTask DispatchTool = "add_maintenance_task"
	ToolAddImportantDate   DispatchTool = "add_important_date"
	ToolAddJobPosting      DispatchTool = "add_job_posting"
)

// DispatchResult is the parsed output of a single LLM dispatch call.
type DispatchResult struct {
	Tool   DispatchTool    `json:"tool"`
	Params json.RawMessage `json:"params"`
}

// Typed param structs — one per dispatchable tool.

type CaptureThoughtParams struct {
	Content string `json:"content"`
}

type LogInteractionParams struct {
	PersonName      string `json:"person_name"`
	InteractionType string `json:"interaction_type"`
	Summary         string `json:"summary"`
	FollowUpNeeded  bool   `json:"follow_up_needed"`
	FollowUpNotes   string `json:"follow_up_notes"`
}

type AddContactParams struct {
	Name    string `json:"name"`
	Company string `json:"company"`
	Title   string `json:"title"`
	Notes   string `json:"notes"`
}

type AddMaintenanceTaskParams struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Location string `json:"location"`
	NextDue  string `json:"next_due"`
	Notes    string `json:"notes"`
}

type AddImportantDateParams struct {
	Title           string `json:"title"`
	EventDate       string `json:"event_date"`
	RecurringYearly bool   `json:"recurring_yearly"`
	Notes           string `json:"notes"`
}

type AddJobPostingParams struct {
	CompanyName string `json:"company_name"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Notes       string `json:"notes"`
}

var dispatchSystemPrompt = `You are a personal knowledge dispatcher. Given a note from the user, determine which tool to invoke and extract the parameters. Respond with ONLY a JSON object — no markdown, no explanation.

Today's date: %s

TOOLS (choose exactly one):

1. capture_thought — general notes, ideas, observations, reminders, anything that doesn't fit the other tools
   params: {"tool":"capture_thought","params":{"content":"<verbatim or lightly cleaned text>"}}

2. log_interaction — meeting, call, coffee, conversation WITH a specific named person
   Trigger: text mentions talking to or meeting with a named person
   params: {"tool":"log_interaction","params":{"person_name":"<full name>","interaction_type":"meeting|call|coffee|email|other","summary":"<what happened>","follow_up_needed":true|false,"follow_up_notes":"<what to follow up, or empty>"}}

3. add_professional_contact — adding a new person to professional contacts
   Trigger: "just met", "add contact", "new contact", "met someone named"
   params: {"tool":"add_professional_contact","params":{"name":"...","company":"...","title":"...","notes":"..."}}

4. add_maintenance_task — home repair, maintenance, or upkeep task
   Trigger: fix, repair, replace, service, HVAC, plumbing, roof, gutter, filter, paint, clean (home context)
   params: {"tool":"add_maintenance_task","params":{"name":"...","category":"...","location":"...","next_due":"YYYY-MM-DD or empty string","notes":"..."}}

5. add_important_date — birthday, anniversary, deadline, or other recurring/important date
   Trigger: birthday, anniversary, annual, every year, deadline on a specific date
   params: {"tool":"add_important_date","params":{"title":"...","event_date":"YYYY-MM-DD","recurring_yearly":true|false,"notes":"..."}}

6. add_job_posting — a job opening or position to track
   Trigger: job posting, apply for, opening at, position at, hiring
   params: {"tool":"add_job_posting","params":{"company_name":"...","title":"...","url":"...","notes":"..."}}

When in doubt, use capture_thought.`

// DispatchCapture makes a single LLM call to determine which tool to invoke
// and what parameters to pass. It does NOT execute the tool.
// On any failure, it gracefully falls back to capture_thought with the original text.
func (a *App) DispatchCapture(ctx context.Context, text string) (*DispatchResult, error) {
	prompt := fmt.Sprintf(dispatchSystemPrompt, time.Now().Format("2006-01-02"))

	body, _ := json.Marshal(map[string]any{
		"model":           "openai/gpt-4o-mini",
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": prompt},
			{"role": "user", "content": text},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+a.OpenRouterKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return captureThoughtFallback(text), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.ReadAll(resp.Body)
		return captureThoughtFallback(text), nil
	}

	var result struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Choices) == 0 {
		return captureThoughtFallback(text), nil
	}

	var dr DispatchResult
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &dr); err != nil {
		return captureThoughtFallback(text), nil
	}

	// Validate the tool name; fall back if unrecognized.
	switch dr.Tool {
	case ToolCaptureThought, ToolLogInteraction, ToolAddContact,
		ToolAddMaintenanceTask, ToolAddImportantDate, ToolAddJobPosting:
	default:
		return captureThoughtFallback(text), nil
	}

	return &dr, nil
}

func captureThoughtFallback(text string) *DispatchResult {
	p, _ := json.Marshal(CaptureThoughtParams{Content: text})
	return &DispatchResult{Tool: ToolCaptureThought, Params: p}
}
