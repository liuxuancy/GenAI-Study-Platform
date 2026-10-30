// 管理员：用户资料、重置密码、班级和成员。
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

export function getAdminUsers() {
  return http.get<{ items: AdminUser[]; total: number }>('/system/users?page=1&pageSize=100')
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
