import type { RoleCode } from '@/core/api/account'

import { adminHomePath, homePathFor, studentHomePath, teacherHomePath } from '@/core/auth/homePath'

/** 登录成功后应进入的工作台路径；管理员可回到原先访问的管理页。 */
export function pathAfterLogin(role: RoleCode, from?: string) {
  const home = homePathFor(role)
  if (role === 'ROLE_STUDENT') {
    return home
  }
  if (role === 'ROLE_TEACHER') {
    return home
  }
  if (!from || from === '/' || from === '/login') {
    return home
  }
  if (from.startsWith('/admin') || from === '/account') {
    return from
  }
  return home
}

export function isRoleHomePath(role: RoleCode, pathname: string) {
  if (role === 'ROLE_STUDENT') {
    return pathname === studentHomePath || pathname.startsWith(`${studentHomePath}/`)
  }
  if (role === 'ROLE_TEACHER') {
    return pathname === teacherHomePath || pathname.startsWith(`${teacherHomePath}/`)
  }
  return pathname === adminHomePath || pathname.startsWith(`${adminHomePath}/`) || pathname === '/account'
}
