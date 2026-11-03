import { http } from '@/core/http'

export interface WorkspaceSkillParam {
  key: string
  label: string
  type: 'text' | 'number' | 'select'
  required: boolean
  options?: string[]
  min?: number
  max?: number
}

export interface WorkspaceSkill {
  id: string
  name: string
  description: string
  allowedRoles: string[]
  requiresCourse: boolean
  params: WorkspaceSkillParam[]
  outputType: string
}

export interface WorkspaceConversation {
  id: number
  title: string
  courseId: number
  courseTitle: string
  courseAccessible: boolean
  updatedAt: string
  messageCount: number
}

export interface WorkspaceMessage {
  id: number
  role: 'user' | 'assistant'
  content: string
  chapterKey: string
  activityKey: string
  sectionTitle: string
  createdAt: string
}

export function listWorkspaceSkills() {
  return http.get<WorkspaceSkill[]>('/workspace/skills')
}

export function listWorkspaceConversations() {
  return http.get<WorkspaceConversation[]>('/workspace/conversations')
}

export function getWorkspaceConversation(id: number) {
  return http.get<{ id: number; title: string; messages: WorkspaceMessage[] }>(`/workspace/conversations/${id}`)
}

export function renameWorkspaceConversation(id: number, title: string) {
  return http.put<{ updated: boolean }>(`/workspace/conversations/${id}`, { title })
}

export function deleteWorkspaceConversation(id: number) {
  return http.delete<{ deleted: boolean }>(`/workspace/conversations/${id}`)
}

export function sendWorkspaceMessage(input: {
  conversationId?: number
  courseId?: number
  content: string
  provider?: string
  skillId?: string
  params?: Record<string, string>
}) {
  return http.post<{ conversationId: number; answer: string; title: string }>('/workspace/messages', input)
}
