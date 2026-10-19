// 路由守卫：没有 token 跳到登录页；有 token 但还没拿到当前用户时，先请求 /me 再显示页面。
import { Button, Result, Spin } from 'antd'
import { useEffect, useState, type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'

export function RequireAuth({ children }: { children: ReactNode }) {
  const token = useAuthStore((state) => state.token)
  const currentUser = useAuthStore((state) => state.currentUser)
  const loadCurrentUser = useAuthStore((state) => state.loadCurrentUser)
  const location = useLocation()
  const [loadErrorMessage, setLoadErrorMessage] = useState<string | null>(null)

  useEffect(() => {
    if (token && !currentUser) {
      // 401 时请求封装已经清掉 token，这里只处理网络错误等其他情况
      loadCurrentUser().catch((error: Error) => setLoadErrorMessage(error.message))
    }
  }, [token, currentUser, loadCurrentUser])

  if (!token) {
    // 记下原来要去的页面，登录后跳回去
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  if (loadErrorMessage) {
    return (
      <Result
        status="error"
        title="Failed to load your account"
        subTitle={loadErrorMessage}
        extra={<Button onClick={() => window.location.reload()}>Retry</Button>}
      />
    )
  }
  if (!currentUser) {
    return <Spin fullscreen />
  }
  return children
}
