// 路由表：登录页公开，其余页面都包在 RequireAuth 里。阶段 7 起各模块的页面自动加到 children。
import { createBrowserRouter } from 'react-router-dom'

import { RequireAuth } from '@/core/auth/RequireAuth'
import { AppLayout } from '@/core/layout/AppLayout'
import { HomePage } from '@/core/pages/HomePage'
import { LoginPage } from '@/core/pages/LoginPage'
import { NotFoundPage } from '@/core/pages/NotFoundPage'
import { AccountPage } from '@/modules/account/AccountPage'
import { AdminClassesPage } from '@/modules/admin/AdminClassesPage'
import { AdminUsersPage } from '@/modules/admin/AdminUsersPage'
import { CourseBubblePage } from '@/modules/student/CourseBubblePage'
import { QuestionDetailPage } from '@/modules/student/QuestionDetailPage'
import { StudentHomePage } from '@/modules/student/StudentHomePage'
import { TeacherHomePage } from '@/modules/teacher/TeacherHomePage'

export const appRouter = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/student',
    element: (
      <RequireAuth>
        <StudentHomePage />
      </RequireAuth>
    ),
  },
  {
    path: '/student/courses/:courseId',
    element: (
      <RequireAuth>
        <CourseBubblePage />
      </RequireAuth>
    ),
  },
  {
    path: '/student/questions/:questionId',
    element: (
      <RequireAuth>
        <QuestionDetailPage />
      </RequireAuth>
    ),
  },
  {
    path: '/',
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: 'teacher', element: <TeacherHomePage /> },
      { path: 'account', element: <AccountPage /> },
      { path: 'admin', element: <AdminUsersPage /> },
      { path: 'admin/classes', element: <AdminClassesPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
