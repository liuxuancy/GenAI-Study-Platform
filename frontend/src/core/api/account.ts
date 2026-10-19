// account 接口：登录、退出、当前用户。对应后端 /api/account/*。
import { http } from '@/core/http'

export type RoleCode = 'ROLE_ADMIN' | 'ROLE_TEACHER' | 'ROLE_STUDENT'

export interface LoginInput {
  username: string
  password: string
}

export interface LoginResult {
  token: string
  expiresAt: string
}

// CurrentUser 和后端 CurrentUserResult 字段一致。管理员 permissions 为空数组，按 isAdmin 全部放行。
export interface CurrentUser {
  id: number
  username: string
  nickname: string
  email: string
  role: RoleCode
  roleName: string
  isAdmin: boolean
  permissions: string[]
}

export function login(input: LoginInput) {
  return http.post<LoginResult>('/account/login', input)
}

export function logout() {
  return http.post<null>('/account/logout')
}

export function getCurrentUser() {
  return http.get<CurrentUser>('/account/me')
}
