// ABOUTME: ThoughtService — captures thoughts with parallel embedding and metadata extraction.
// ABOUTME: Consolidates the duplicate capture logic that existed in brain/dispatch.go and core/thoughts.go.

package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	pgvector "github.com/pgvector/pgvector-go"

	"open-brain-go/brain"
	"open-brain-go/brain/repository"
)

// ThoughtService handles thought capture: embedding generation, metadata extraction, and persistence.
type ThoughtService struct {
	app *brain.App
}

// NewThoughtService creates a ThoughtService backed by the given App.
func NewThoughtService(app *brain.App) *ThoughtService {
	return &ThoughtService{app: app}
}

// Capture generates an embedding and metadata for content, persists it as a thought,
// and returns a human-readable summary string. source is stored in the metadata (e.g. "mcp", "web").
func (s *ThoughtService) Capture(ctx context.Context, content, source string) (string, error) {
	type embResult struct {
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
		v, err := s.app.GetEmbedding(ctx, content)
		embCh <- embResult{v, err}
	}()
	go func() {
		meta, err := s.app.ExtractMetadata(ctx, content)
		metaCh <- metaResult{meta, err}
	}()

	er := <-embCh
	mr := <-metaCh

	if er.err != nil {
		return "", fmt.Errorf("embedding: %w", er.err)
	}
	if mr.err != nil {
		return "", fmt.Errorf("metadata: %w", mr.err)
	}

	meta := mr.meta
	meta.Source = source

	if err := s.app.WithUserTx(ctx, func(tx pgx.Tx) error {
		return repository.InsertThought(ctx, tx, content, er.v, meta)
	}); err != nil {
		return "", fmt.Errorf("save thought: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Captured as %s", meta.Type)
	if len(meta.Topics) > 0 {
		fmt.Fprintf(&sb, " — %s", strings.Join(meta.Topics, ", "))
	}
	if len(meta.People) > 0 {
		fmt.Fprintf(&sb, " | people: %s", strings.Join(meta.People, ", "))
	}
	if len(meta.ActionItems) > 0 {
		fmt.Fprintf(&sb, " | actions: %s", strings.Join(meta.ActionItems, "; "))
	}
	return sb.String(), nil
}
