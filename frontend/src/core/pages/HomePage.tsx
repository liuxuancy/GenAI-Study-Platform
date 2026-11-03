// 根路径：按角色进入对应工作台，不再停留于旧的权限调试页。
import { Navigate } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'

export function HomePage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  if (!currentUser) {
    return null
  }
  return <Navigate to={homePathFor(currentUser.role)} replace />
}
