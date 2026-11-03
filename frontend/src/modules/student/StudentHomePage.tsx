import { Navigate } from 'react-router-dom'

/** 保留重定向，兼容直接访问旧入口。 */
export function StudentHomePage() {
  return <Navigate to="/student" replace />
}
