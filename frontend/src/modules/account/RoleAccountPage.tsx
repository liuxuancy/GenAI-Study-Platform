import { Navigate } from 'react-router-dom'

import { homePathFor, studentHomePath } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'

import { AccountPage } from '@/modules/account/AccountPage'

export function StudentAccountPage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_STUDENT') {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }
  return <AccountPage />
}

export function TeacherAccountPage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_TEACHER') {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }
  return <AccountPage />
}

/** 管理员仍走 AppLayout；学生 / 讲师访问 /account 时重定向到角色专属路径。 */
export function AccountRoute() {
  const currentUser = useAuthStore((state) => state.currentUser)
  if (!currentUser) {
    return null
  }
  if (currentUser.role === 'ROLE_STUDENT') {
    return <Navigate to={`${studentHomePath}/account`} replace />
  }
  if (currentUser.role === 'ROLE_TEACHER') {
    return <Navigate to="/teacher/account" replace />
  }
  return <AccountPage />
}
