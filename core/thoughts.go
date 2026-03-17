// ABOUTME: Core Open Brain tools: capture, search, list, and summarize thoughts.
// ABOUTME: Registered into the single MCP server alongside any active extensions.

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	pgvector "github.com/pgvector/pgvector-go"

	"open-brain-go/brain"
)

// Register adds the four core thought tools to the MCP server.
func Register(s *server.MCPServer, a *brain.App) {
	s.AddTool(mcp.NewTool("search_thoughts",
		mcp.WithDescription("Search captured thoughts by meaning. Use this when the user asks about a topic, person, or idea they've previously captured."),
		mcp.WithString("query", mcp.Required(), mcp.Description("What to search for")),
		mcp.WithNumber("limit", mcp.Description("Max results to return (default 10)")),
		mcp.WithNumber("threshold", mcp.Description("Similarity threshold 0–1 (default 0.5). Lower = broader results.")),
	), searchThoughts(a))

	s.AddTool(mcp.NewTool("list_thoughts",
		mcp.WithDescription("List recently captured thoughts with optional filters by type, topic, person, or time range."),
		mcp.WithNumber("limit", mcp.Description("Max results (default 10)")),
		mcp.WithString("type", mcp.Description("Filter by type: observation, task, idea, reference, person_note")),
		mcp.WithString("topic", mcp.Description("Filter by topic tag")),
		mcp.WithString("person", mcp.Description("Filter by person mentioned")),
		mcp.WithNumber("days", mcp.Description("Only thoughts from the last N days")),
	), listThoughts(a))

	s.AddTool(mcp.NewTool("thought_stats",
		mcp.WithDescription("Get a summary of all captured thoughts: totals, types, top topics, and people."),
	), thoughtStats(a))

	s.AddTool(mcp.NewTool("capture_thought",
		mcp.WithDescription("Save a new thought to the Open Brain. Generates an embedding and extracts metadata automatically."),
		mcp.WithString("content", mcp.Required(), mcp.Description("The thought to capture — a clear, standalone statement that will make sense when retrieved later")),
	), captureThought(a))
}

func searchThoughts(a *brain.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, _ := req.GetArguments()["query"].(string)
		if query == "" {
			return brain.ToolError("query is required"), nil
		}
		limit := 10
		if v, ok := req.GetArguments()["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}
		threshold := 0.5
		if v, ok := req.GetArguments()["threshold"].(float64); ok {
			threshold = v
		}

		emb, err := a.GetEmbedding(ctx, query)
		if err != nil {
			return brain.ToolError("Failed to generate embedding: " + err.Error()), nil
		}

		type result struct {
			Content    string
			Metadata   brain.ThoughtMetadata
			Similarity float64
			CreatedAt  time.Time
		}
		var results []result

		err = a.WithUserTx(ctx, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx,
				"SELECT content, metadata, similarity, created_at FROM match_thoughts($1, $2, $3)",
				emb, threshold, limit,
			)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var r result
				var metaRaw []byte
				if err := rows.Scan(&r.Content, &metaRaw, &r.Similarity, &r.CreatedAt); err != nil {
					return err
				}
				json.Unmarshal(metaRaw, &r.Metadata)
				results = append(results, r)
			}
			return rows.Err()
		})
		if err != nil {
			return brain.ToolError("Search error: " + err.Error()), nil
		}

		if len(results) == 0 {
			return brain.TextResult(fmt.Sprintf(`No thoughts found matching "%s".`, query)), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Found %d thought(s):\n\n", len(results))
		for i, r := range results {
			fmt.Fprintf(&sb, "--- Result %d (%.1f%% match) ---\n", i+1, r.Similarity*100)
			fmt.Fprintf(&sb, "Captured: %s\nType: %s\n", r.CreatedAt.Format("2006-01-02"), r.Metadata.Type)
			if len(r.Metadata.Topics) > 0 {
				fmt.Fprintf(&sb, "Topics: %s\n", strings.Join(r.Metadata.Topics, ", "))
			}
			if len(r.Metadata.People) > 0 {
				fmt.Fprintf(&sb, "People: %s\n", strings.Join(r.Metadata.People, ", "))
			}
			if len(r.Metadata.ActionItems) > 0 {
				fmt.Fprintf(&sb, "Actions: %s\n", strings.Join(r.Metadata.ActionItems, "; "))
			}
			fmt.Fprintf(&sb, "\n%s\n\n", r.Content)
		}
		return brain.TextResult(sb.String()), nil
	}
}

func listThoughts(a *brain.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := 10
		if v, ok := req.GetArguments()["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}
		typeFilter, _   := req.GetArguments()["type"].(string)
		topicFilter, _  := req.GetArguments()["topic"].(string)
		personFilter, _ := req.GetArguments()["person"].(string)
		var days int
		if v, ok := req.GetArguments()["days"].(float64); ok && v > 0 {
			days = int(v)
		}

		type thought struct {
			Content   string
			Metadata  brain.ThoughtMetadata
			CreatedAt time.Time
		}
		var thoughts []thought

		err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
			sql := `SELECT content, metadata, created_at FROM thoughts WHERE true`
			args := []any{}
			n := 1

			if typeFilter != "" {
				b, _ := json.Marshal(map[string]string{"type": typeFilter})
				sql += fmt.Sprintf(" AND metadata @> $%d::jsonb", n)
				args = append(args, string(b))
				n++
			}
			if topicFilter != "" {
				b, _ := json.Marshal(map[string][]string{"topics": {topicFilter}})
				sql += fmt.Sprintf(" AND metadata @> $%d::jsonb", n)
				args = append(args, string(b))
				n++
			}
			if personFilter != "" {
				b, _ := json.Marshal(map[string][]string{"people": {personFilter}})
				sql += fmt.Sprintf(" AND metadata @> $%d::jsonb", n)
				args = append(args, string(b))
				n++
			}
			if days > 0 {
				sql += fmt.Sprintf(" AND created_at >= now() - $%d * interval '1 day'", n)
				args = append(args, days)
				n++
			}
			sql += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", n)
			args = append(args, limit)

			rows, err := tx.Query(ctx, sql, args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var t thought
				var metaRaw []byte
				if err := rows.Scan(&t.Content, &metaRaw, &t.CreatedAt); err != nil {
					return err
				}
				json.Unmarshal(metaRaw, &t.Metadata)
				thoughts = append(thoughts, t)
			}
			return rows.Err()
		})
		if err != nil {
			return brain.ToolError("Error: " + err.Error()), nil
		}

		if len(thoughts) == 0 {
			return brain.TextResult("No thoughts found."), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "%d recent thought(s):\n\n", len(thoughts))
		for i, t := range thoughts {
			ttype := t.Metadata.Type
			if ttype == "" {
				ttype = "??"
			}
			meta := ttype
			if tags := strings.Join(t.Metadata.Topics, ", "); tags != "" {
				meta += " - " + tags
			}
			fmt.Fprintf(&sb, "%d. [%s] (%s)\n   %s\n\n", i+1, t.CreatedAt.Format("2006-01-02"), meta, t.Content)
		}
		return brain.TextResult(sb.String()), nil
	}
}

func thoughtStats(a *brain.App) server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var lines []string

		err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
			var total int
			var earliest, latest time.Time
			if err := tx.QueryRow(ctx,
				`SELECT COUNT(*), MIN(created_at), MAX(created_at) FROM thoughts`,
			).Scan(&total, &earliest, &latest); err != nil {
				return err
			}
			lines = append(lines, fmt.Sprintf("Total thoughts: %d", total))
			if total > 0 {
				lines = append(lines, fmt.Sprintf("Date range: %s → %s",
					earliest.Format("2006-01-02"), latest.Format("2006-01-02")))
			}

			for _, query := range []struct {
				label string
				sql   string
			}{
				{"Types", `SELECT metadata->>'type', COUNT(*) FROM thoughts WHERE metadata ? 'type' GROUP BY 1 ORDER BY 2 DESC`},
			} {
				rows, err := tx.Query(ctx, query.sql)
				if err != nil {
					return err
				}
				var section []string
				for rows.Next() {
					var k string
					var c int
					if err := rows.Scan(&k, &c); err != nil {
						rows.Close()
						return err
					}
					section = append(section, fmt.Sprintf("  %s: %d", k, c))
				}
				rows.Close()
				if err := rows.Err(); err != nil {
					return err
				}
				if len(section) > 0 {
					lines = append(lines, "", query.label+":")
					lines = append(lines, section...)
				}
			}

			for _, query := range []struct {
				label string
				sql   string
			}{
				{"Top topics", `SELECT topic, COUNT(*) FROM thoughts, jsonb_array_elements_text(metadata->'topics') AS topic WHERE metadata ? 'topics' GROUP BY topic ORDER BY 2 DESC LIMIT 10`},
				{"People mentioned", `SELECT person, COUNT(*) FROM thoughts, jsonb_array_elements_text(metadata->'people') AS person WHERE metadata ? 'people' GROUP BY person ORDER BY 2 DESC LIMIT 10`},
			} {
				rows, err := tx.Query(ctx, query.sql)
				if err != nil {
					return err
				}
				var section []string
				for rows.Next() {
					var k string
					var c int
					if err := rows.Scan(&k, &c); err != nil {
						rows.Close()
						return err
					}
					section = append(section, fmt.Sprintf("  %s: %d", k, c))
				}
				rows.Close()
				if err := rows.Err(); err != nil {
					return err
				}
				if len(section) > 0 {
					lines = append(lines, "", query.label+":")
					lines = append(lines, section...)
				}
			}
			return nil
		})
		if err != nil {
			return brain.ToolError("Error: " + err.Error()), nil
		}

		return brain.TextResult(strings.Join(lines, "\n")), nil
	}
}

func captureThought(a *brain.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, _ := req.GetArguments()["content"].(string)
		if content == "" {
			return brain.ToolError("content is required"), nil
		}

		// Fetch embedding and metadata in parallel.
		type embResult  struct {
			v   pgvector.Vector
			err error
		}
		type metaResult struct {
			meta *brain.ThoughtMetadata
			err  error
		}

		embCh  := make(chan embResult, 1)
		metaCh := make(chan metaResult, 1)

		go func() {
			v, err := a.GetEmbedding(ctx, content)
			embCh <- embResult{v, err}
		}()
		go func() {
			meta, err := a.ExtractMetadata(ctx, content)
			metaCh <- metaResult{meta, err}
		}()

		er := <-embCh
		mr := <-metaCh

		if er.err != nil {
			return brain.ToolError("Embedding error: " + er.err.Error()), nil
		}
		if mr.err != nil {
			return brain.ToolError("Metadata error: " + mr.err.Error()), nil
		}

		meta := mr.meta
		meta.Source = "mcp"
		metaJSON, _ := json.Marshal(meta)

		err := a.WithUserTx(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				`INSERT INTO thoughts (user_id, content, embedding, metadata)
				 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3)`,
				content, er.v, metaJSON,
			)
			return err
		})
		if err != nil {
			return brain.ToolError("Failed to capture: " + err.Error()), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Captured as %s", meta.Type)
		if len(meta.Topics) > 0 {
			fmt.Fprintf(&sb, " — %s", strings.Join(meta.Topics, ", "))
		}
		if len(meta.People) > 0 {
			fmt.Fprintf(&sb, " | People: %s", strings.Join(meta.People, ", "))
		}
		if len(meta.ActionItems) > 0 {
			fmt.Fprintf(&sb, " | Actions: %s", strings.Join(meta.ActionItems, "; "))
		}
		return brain.TextResult(sb.String()), nil
	}
}
