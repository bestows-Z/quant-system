# rag-rpc

`rag-rpc` provides the runnable RAG loop for the quant platform. It stores knowledge bases, documents and chunks, builds local embeddings, writes them to Qdrant, and exposes search and ask interfaces through `gateway-api`.

## Current Scope

- Knowledge base create/list/detail
- Text document ingestion
- Deterministic chunking with overlap
- Deterministic 384-dimensional feature-hashing embedding with Chinese token support
- Per-knowledge-base Qdrant collection creation and point upsert
- Qdrant vector search with document/chunk payload and similarity score
- MySQL vector search as an explicit disaster-recovery fallback
- Traceable extractive answer based on retrieved chunks
- RabbitMQ events:
  - `rag.document.uploaded`
  - `rag.document.chunked`
  - `rag.embedding.completed`

## Run

```bash
cd backend-go
go run ./apps/rag-rpc/rag.go -f ./apps/rag-rpc/etc/rag-rpc.yaml
```

Required dependencies:

- MySQL initialized with `deploy/init.sql`
- etcd
- RabbitMQ, optional at runtime; unavailable RabbitMQ degrades to logging
- Qdrant 1.9 or later

## Verify

```bash
cd backend-go
GOCACHE=$PWD/.gocache go test ./...
GOCACHE=$PWD/.gocache go build ./...
```

All gateway responses are wrapped as `{code,message,data,trace_id}`. Generated summaries are traceable research outputs and do not constitute investment advice.
