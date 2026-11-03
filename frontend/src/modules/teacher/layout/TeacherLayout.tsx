import { useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'

import styles from '@/modules/student/layout/StudentLayout.module.css'
import { TeacherSidebar } from '@/modules/teacher/layout/TeacherSidebar'
import { TeacherTopbar } from '@/modules/teacher/layout/TeacherTopbar'

export function TeacherLayout() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const location = useLocation()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const { t } = useI18n()

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_TEACHER') {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }

  function closeSidebar() {
    setSidebarOpen(false)
  }

  const aiWorkspace = /\/teacher\/ai\/?$/.test(location.pathname)

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
      <TeacherSidebar open={sidebarOpen} onNavigate={closeSidebar} />
      <div className={styles.mainLayout}>
        <TeacherTopbar pathname={location.pathname} onMenuClick={() => setSidebarOpen(true)} />
        <main
          className={`${styles.pageContainer} ${aiWorkspace ? styles.pageContainerFlush : ''}`}
        >
          <Outlet />
        </main>
      </div>
    </div>
  )
}
