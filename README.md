# Engram

A self-hosted MCP server for persistent AI memory, built in Go. Engram gives any MCP-compatible AI client (Claude Desktop, etc.) a personal memory layer backed by PostgreSQL and pgvector.

## What it does

- **Capture thoughts** — save freeform text with automatic metadata extraction and vector embeddings
- **Semantic search** — find relevant memories using natural language, not just keywords
- **Structured extensions** — household inventory, home maintenance, family calendar, meal planning, professional CRM, and job hunt tracking
- **Multi-user with RLS** — each user's data is isolated via PostgreSQL Row Level Security

## Quick start

```bash
# Configure secrets
cat > .env <<'EOF'
OPENROUTER_API_KEY=sk-or-your-key-here
POSTGRES_PASSWORD=pick-a-pg-password
APP_USER_PASSWORD=pick-an-app-password
EOF

# Start the server
docker compose up -d

# Watch logs
docker compose logs -f
```

The server listens on `http://localhost:8080`. Point your MCP client at it with your user access key.

## Architecture

- **Go + mcp-go** — lightweight MCP server using the Streamable HTTP transport
- **PostgreSQL 17 + pgvector** — vector similarity search for semantic memory recall
- **OpenRouter** — embeddings (text-embedding-3-small) and metadata extraction (gpt-4o-mini)
- **Docker Compose** — two-container setup (Postgres + Go binary), schema auto-applied on first boot

## Origins

Engram is derived from [Open Brain](https://github.com/NateBJones-Projects/OB1) by Nate B. Jones, a persistent AI memory system. Open Brain provides the database schema, extension architecture, and learning path that Engram builds on.
