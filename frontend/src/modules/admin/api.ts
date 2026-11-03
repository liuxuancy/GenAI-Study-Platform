// 管理员：用户资料、重置密码、班级、课程下架、模型用量。
import { http } from '@/core/http'
import type { RoleCode } from '@/core/api/account'

export interface AdminUser {
  id: number
  username: string
  nickname: string
  email: string
  major: string
  role: RoleCode
  roleName: string
  status: number
}

export interface AdminClassMember {
  userId: number
  username: string
  nickname: string
  role: string
}

export interface AdminClass {
  id: number
  name: string
  members: AdminClassMember[]
}

export interface AdminUserQuery {
  keyword?: string
  role?: string
  status?: number
  page?: number
  pageSize?: number
}

function queryString(values: Record<string, string | number | undefined>) {
  const params = new URLSearchParams()
  Object.entries(values).forEach(([key, value]) => {
    if (value !== undefined && value !== '') {
      params.set(key, String(value))
    }
  })
  const encoded = params.toString()
  return encoded ? `?${encoded}` : ''
}

export function getAdminUsers(query: AdminUserQuery = {}) {
  return http.get<{ items: AdminUser[]; total: number }>(
    `/system/users${queryString({ page: 1, pageSize: 20, ...query })}`,
  )
}

export function updateAdminUser(userId: number, input: { nickname: string; email: string; major: string; status: number }) {
  return http.put<null>(`/system/users/${userId}`, input)
}

export function resetAdminUserPassword(userId: number, password: string) {
  return http.put<null>(`/system/users/${userId}/password`, { password })
}

export function getAdminClasses() {
  return http.get<AdminClass[]>('/system/classes')
}

export function createAdminClass(name: string) {
  return http.post<AdminClass>('/system/classes', { name })
}

export function renameAdminClass(classId: number, name: string) {
  return http.put<AdminClass>(`/system/classes/${classId}`, { name })
}

export function setAdminClassMembers(classId: number, userIds: number[]) {
  return http.put<AdminClass>(`/system/classes/${classId}/members`, { userIds })
}

export interface AdminCourse {
  id: number
  title: string
  description: string
  published: boolean
  ownerId: number
  ownerUsername: string
  ownerNickname: string
  ownerRole: string
  ownerRoleName: string
  classCount: number
  createdAt: string
}

export interface AdminCourseList {
  items: AdminCourse[]
  total: number
  publishedCount: number
}

export interface AdminUsageCall {
  id: number
  scene: string
  model: string
  promptTokens: number
  completionTokens: number
  totalTokens: number
  callStatus: string
  createdAt: string
}

export interface AdminUsageUser {
  userId: number
  username: string
  nickname: string
  role: string
  roleName: string
  callCount: number
  promptTokens: number
  completionTokens: number
  totalTokens: number
  calls: AdminUsageCall[]
}

export interface AdminUsageReport {
  accountCount: number
  callCount: number
  promptTokens: number
  completionTokens: number
  totalTokens: number
  users: AdminUsageUser[]
  total: number
  page: number
  pageSize: number
}

export function getAdminCourses() {
  return http.get<AdminCourseList>('/system/courses')
}

export function setAdminCoursePublished(courseId: number, published: boolean) {
  return http.put<AdminCourse>(`/system/courses/${courseId}/publish`, { published })
}

export interface AdminUsageQuery {
  from?: string
  to?: string
  user?: string
  role?: string
  scene?: string
  model?: string
  call_status?: string
  page?: number
  pageSize?: number
}

export function getAdminUsage(query: AdminUsageQuery = {}) {
  return http.get<AdminUsageReport>(
    `/system/usage${queryString({
      from: query.from,
      to: query.to,
      user: query.user,
      role: query.role,
      scene: query.scene,
      model: query.model,
      call_status: query.call_status,
      page: query.page,
      pageSize: query.pageSize,
    })}`,
  )
}
