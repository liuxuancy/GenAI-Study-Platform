// 路由表：登录页公开，其余页面都包在 RequireAuth 里。阶段 7 起各模块的页面自动加到 children。
import { createBrowserRouter } from 'react-router-dom'

import { RequireAuth } from '@/core/auth/RequireAuth'
import { AppLayout } from '@/core/layout/AppLayout'
import { HomePage } from '@/core/pages/HomePage'
import { LoginPage } from '@/core/pages/LoginPage'
import { NotFoundPage } from '@/core/pages/NotFoundPage'

export const appRouter = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
