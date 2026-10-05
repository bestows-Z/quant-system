export interface KnowledgeBase {
  id: number
	user_id?: number
  name: string
  description?: string
  visibility?: string
  doc_count?: number
  chunk_count?: number
  embedding_provider?: string
  vector_store?: string
  collection_name?: string
  created_at?: string
}

export interface RagDocument {
  id: number
	user_id?: number
  knowledge_base_id: number
  title: string
  doc_type?: string
  source_url?: string
  status?: string
	chunk_count?: number
  created_at?: string
}

export interface RagChunk {
  id: number
  chunk_id?: number
  document_id?: number
  chunk_index?: number
  content: string
  score?: number
  document_title?: string
  metadata_json?: string
}

export interface RagAskResult {
  answer?: string
  references?: RagChunk[]
  chunks?: RagChunk[]
  provider?: string
  model?: string
  disclaimer?: string
}
