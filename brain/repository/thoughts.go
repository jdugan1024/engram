// ABOUTME: Repository functions for the thoughts table.
// ABOUTME: Each function operates inside a caller-supplied pgx.Tx (RLS already set on it).

package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	pgvector "github.com/pgvector/pgvector-go"

	"open-brain-go/brain"
)

// InsertThought writes a thought row inside an existing transaction.
// RLS is enforced via app.current_user_id set on the transaction.
func InsertThought(ctx context.Context, tx pgx.Tx, content string, embedding pgvector.Vector, meta *brain.ThoughtMetadata) error {
	metaJSON, _ := json.Marshal(meta)
	_, err := tx.Exec(ctx,
		`INSERT INTO thoughts (user_id, content, embedding, metadata)
		 VALUES (current_setting('app.current_user_id')::uuid, $1, $2, $3)`,
		content, embedding, metaJSON,
	)
	return err
}
