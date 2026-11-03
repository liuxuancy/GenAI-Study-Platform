import { useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'

import { StudentSidebar } from '@/modules/student/layout/StudentSidebar'
import { StudentTopbar } from '@/modules/student/layout/StudentTopbar'
import styles from '@/modules/student/layout/StudentLayout.module.css'

export function StudentLayout() {
  const { t } = useI18n()
  const currentUser = useAuthStore((state) => state.currentUser)
  const location = useLocation()
  const [sidebarOpen, setSidebarOpen] = useState(false)

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_STUDENT') {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }

  function closeSidebar() {
    setSidebarOpen(false)
  }

  const aiWorkspace = /\/ai\/?$/.test(location.pathname)
  const quizAttempt = /\/student\/questions\//.test(location.pathname)

  return (
    <div className={`${styles.studentLayout} ${aiWorkspace ? styles.studentLayoutFlush : ''}`}>
      {sidebarOpen ? (
        <button
          type="button"
          className={styles.backdrop}
          aria-label={t('nav.common.closeMenu')}
          onClick={closeSidebar}
        />
      ) : null}
      <StudentSidebar open={sidebarOpen} onNavigate={closeSidebar} />
      <div className={styles.mainLayout}>
        <StudentTopbar pathname={location.pathname} onMenuClick={() => setSidebarOpen(true)} />
        <main
          className={`${styles.pageContainer} ${aiWorkspace ? styles.pageContainerFlush : ''} ${quizAttempt ? styles.pageContainerQuiz : ''}`}
        >
          <Outlet />
        </main>
      </div>
    </div>
  )
}
