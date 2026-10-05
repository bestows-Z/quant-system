import request, { requestList } from './request'
import type { AgentChatMessage, AgentChatSession, AgentReport, AgentReportPage, AgentTask } from '@/types/agent'

export const agentApi = {
  analyzeStock: (data: { symbol: string; question?: string; knowledge_base_id?: number; account_id?: number }) =>
    request<AgentReport>({ url: '/api/agent/analyze-stock', method: 'POST', data, timeout: 90000 }),
  ask: (data: { question: string; symbol?: string; knowledge_base_id?: number; account_id?: number }) =>
    request<AgentReport>({ url: '/api/agent/ask', method: 'POST', data, timeout: 90000 }),
  reviewBacktest: (data: { backtest_id: number; question?: string; knowledge_base_id?: number }) =>
    request<AgentReport>({ url: '/api/agent/review-backtest', method: 'POST', data, timeout: 90000 }),
  tasks: (params?: Record<string, unknown>) => requestList<AgentTask>({ url: '/api/agent/tasks', method: 'GET', params }),
  task: (id: number) => request<AgentTask>({ url: `/api/agent/tasks/${id}`, method: 'GET' }),
  report: (id: number) => request<AgentReport>({ url: `/api/agent/reports/${id}`, method: 'GET' }),
  reports: (params?: Record<string, unknown>) => request<AgentReportPage>({ url: '/api/agent/reports', method: 'GET', params }),
  removeReport: (id: number) => request<{ deleted_count:number }>({ url: `/api/agent/reports/${id}`, method: 'DELETE' }),
  removeReports: (ids: number[]) => request<{ deleted_count:number }>({ url: '/api/agent/reports', method: 'DELETE', data: { ids } }),
  chatSessions: () => requestList<AgentChatSession>({ url: '/api/agent/chat/sessions' }),
  createChatSession: (title='新研究会话') => request<AgentChatSession>({ url: '/api/agent/chat/sessions', method:'POST', data:{title} }),
  renameChatSession: (id:number,title:string) => request<AgentChatSession>({ url:`/api/agent/chat/sessions/${id}`, method:'PUT', data:{title} }),
  deleteChatSession: (id:number) => request<{affected:number}>({ url:`/api/agent/chat/sessions/${id}`, method:'DELETE' }),
  clearChatMessages: (id:number) => request<{affected:number}>({ url:`/api/agent/chat/sessions/${id}/messages`, method:'DELETE' }),
  chatMessages: (id:number) => requestList<AgentChatMessage>({ url:`/api/agent/chat/sessions/${id}/messages` })
}
