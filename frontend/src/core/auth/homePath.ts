// 登录后按角色决定落点。学生进学生首页，讲师进讲师首页，管理员进现有首页。
import type { RoleCode } from '@/core/api/account'

export const studentHomePath = '/student'
export const teacherHomePath = '/teacher'

export function homePathFor(role: RoleCode) {
  if (role === 'ROLE_STUDENT') {
    return studentHomePath
  }
  if (role === 'ROLE_TEACHER') {
    return teacherHomePath
  }
  return '/'
}
