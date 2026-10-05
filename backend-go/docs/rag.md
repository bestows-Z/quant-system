# RAG Design

RAG service owns financial text knowledge base features.

Tables:
- `knowledge_bases`
- `rag_documents`
- `rag_chunks`
- `rag_embeddings`
- `rag_retrieval_logs`
- `rag_qa_logs`

Provider interfaces:
- embedding provider: mock first, later OpenAI, DeepSeek, Qwen, Ollama or local model
- vector store: Qdrant first
- LLM client: mock first, replaceable through `common/xllm`

Current round status:
- SQL tables exist.
- Qdrant client wrapper exists in `common/xqdrant`.
- LLM mock client exists in `common/xllm`.
- `rag-rpc` config template includes MySQL, Redis, Qdrant, RabbitMQ and LLM.

First real implementation planned for round 5:
- create knowledge base
- register document
- chunk text
- generate mock embedding
- perform mock keyword/vector fallback search
- write retrieval and QA logs

