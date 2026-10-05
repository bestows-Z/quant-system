import request, { requestList } from './request'
import type { KnowledgeBase, RagAskResult, RagChunk, RagDocument } from '@/types/rag'

export const ragApi = {
  createKnowledgeBase: (data: { name: string; description?: string; visibility?: string }) =>
    request<KnowledgeBase>({ url: '/api/rag/knowledge-bases', method: 'POST', data }),
  knowledgeBases: () => requestList<KnowledgeBase>({ url: '/api/rag/knowledge-bases', method: 'GET' }),
  knowledgeBase: (id: number) => request<KnowledgeBase>({ url: `/api/rag/knowledge-bases/${id}`, method: 'GET' }),
	deleteKnowledgeBase: (id: number) => request<{ok:boolean;vector_cleanup_pending?:boolean}>({ url: `/api/rag/knowledge-bases/${id}`, method: 'DELETE' }),
  createDocument: (data: Record<string, unknown>) => request<RagDocument>({ url: '/api/rag/documents', method: 'POST', data }),
  documents: (knowledge_base_id: number) => requestList<RagDocument>({ url: '/api/rag/documents', method: 'GET', params: { knowledge_base_id } }),
  document: (id: number) => request<RagDocument>({ url: `/api/rag/documents/${id}`, method: 'GET' }),
	deleteDocument: (id: number) => request<{ok:boolean;vector_cleanup_pending?:boolean}>({ url: `/api/rag/documents/${id}`, method: 'DELETE' }),
  chunks: (document_id: number) => requestList<RagChunk>({ url: '/api/rag/chunks', method: 'GET', params: { document_id } }),
  search: (data: { knowledge_base_id: number; query: string; top_k?: number }) => request<{ items?: RagChunk[]; chunks?: RagChunk[]; provider?: string; model?: string }>({ url: '/api/rag/search', method: 'POST', data }),
  ask: (data: { knowledge_base_id: number; question: string; top_k?: number }) => request<RagAskResult>({ url: '/api/rag/ask', method: 'POST', data })
}
