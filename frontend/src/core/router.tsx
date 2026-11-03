// 路由表：登录页公开，其余页面都包在 RequireAuth 里。阶段 7 起各模块的页面自动加到 children。
import { createBrowserRouter } from 'react-router-dom'

import { RequireAuth } from '@/core/auth/RequireAuth'
import { AppLayout } from '@/core/layout/AppLayout'
import { HomePage } from '@/core/pages/HomePage'
import { LoginPage } from '@/core/pages/LoginPage'
import { NotFoundPage } from '@/core/pages/NotFoundPage'
import { AccountRoute, StudentAccountPage, TeacherAccountPage } from '@/modules/account/RoleAccountPage'
import { AdminClassesPage } from '@/modules/admin/AdminClassesPage'
import { AdminCoursesPage } from '@/modules/admin/AdminCoursesPage'
import { AdminSystemPage } from '@/modules/admin/AdminSystemPage'
import { AdminUsersPage } from '@/modules/admin/AdminUsersPage'
import { CourseBubblePage } from '@/modules/student/CourseBubblePage'
import { QuestionDetailPage } from '@/modules/student/QuestionDetailPage'
import { LearningDashboard } from '@/modules/student/learning/LearningDashboard'
import { MyCoursesPage } from '@/modules/student/learning/MyCoursesPage'
import { StudentLayout } from '@/modules/student/layout/StudentLayout'
import { QuizDashboard } from '@/modules/student/quiz/QuizDashboard'
import { StudentAiPage } from '@/modules/student/learning/StudentAiPage'
import { TeacherLayout } from '@/modules/teacher/layout/TeacherLayout'
import { TeacherAiPage } from '@/modules/teacher/TeacherAiPage'
import { TeacherLearningOverviewPage } from '@/modules/teacher/TeacherLearningOverviewPage'
import { TeacherHomePage } from '@/modules/teacher/TeacherHomePage'
import { TeacherCourseWorkspacePage } from '@/modules/teacher/TeacherCourseWorkspacePage'

export const appRouter = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/student',
    element: (
      <RequireAuth>
        <StudentLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <LearningDashboard /> },
      { path: 'courses', element: <MyCoursesPage /> },
      { path: 'ai', element: <StudentAiPage /> },
      { path: 'quizzes', element: <QuizDashboard /> },
      { path: 'account', element: <StudentAccountPage /> },
      { path: 'questions/:questionId', element: <QuestionDetailPage /> },
    ],
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
    path: '/teacher',
    element: (
      <RequireAuth>
        <TeacherLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <TeacherHomePage /> },
      { path: 'learning', element: <TeacherLearningOverviewPage /> },
      { path: 'ai', element: <TeacherAiPage /> },
      { path: 'account', element: <TeacherAccountPage /> },
    ],
  },
  {
    path: '/teacher/courses/:courseId',
    element: (
      <RequireAuth>
        <TeacherCourseWorkspacePage />
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
      { path: 'account', element: <AccountRoute /> },
      { path: 'admin', element: <AdminUsersPage /> },
      { path: 'admin/classes', element: <AdminClassesPage /> },
      { path: 'admin/courses', element: <AdminCoursesPage /> },
      { path: 'admin/system', element: <AdminSystemPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
