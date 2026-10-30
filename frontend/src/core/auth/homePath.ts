// 登录后按角色决定落点。学生进学生首页，其余角色进现有首页。
import type { RoleCode } from '@/core/api/account'

export const studentHomePath = '/student'

export function homePathFor(role: RoleCode) {
  if (role === 'ROLE_STUDENT') {
    return studentHomePath
  }
  return '/'
}
