import {
  BookOutlined,
  CommentOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { Link, useLocation, useNavigate } from 'react-router-dom'

import { teacherHomePath } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'
import { localizedRoleName } from '@/core/i18n/roleName'
import { useI18n } from '@/core/i18n/useI18n'

import styles from '@/modules/student/layout/StudentLayout.module.css'
import {
  isTeacherNavActive,
  teacherNavSections,
  type TeacherNavItem,
} from '@/modules/teacher/layout/teacherNav'

function NavIcon({ icon }: { icon: TeacherNavItem['icon'] }) {
  if (icon === 'courses') {
    return <BookOutlined className={styles.navIcon} aria-hidden />
  }
  if (icon === 'spark') {
    return <CommentOutlined className={styles.navIcon} aria-hidden />
  }
  if (icon === 'learning') {
    return <TeamOutlined className={styles.navIcon} aria-hidden />
  }
  return <UserOutlined className={styles.navIcon} aria-hidden />
}

interface TeacherSidebarProps {
  open: boolean
  onNavigate?: () => void
}

export function TeacherSidebar({ open, onNavigate }: TeacherSidebarProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const currentUser = useAuthStore((state) => state.currentUser)
  const logout = useAuthStore((state) => state.logout)
  const { t } = useI18n()

  return (
    <aside
      className={`${styles.sidebar} ${open ? styles.sidebarOpen : styles.sidebarClosed}`}
      aria-label={t('nav.teacher.sidebarAria')}
    >
      <Link className={styles.brand} to={teacherHomePath} onClick={onNavigate}>
        <span className={styles.mark} aria-hidden="true">
          C
        </span>
        <span className={styles.brandText}>
          <strong>{t('common.brandName')}</strong>
          <small>{t('nav.teacher.brandSubtitle')}</small>
        </span>
      </Link>

      <div className={styles.navScroll}>
        {teacherNavSections.map((section) => (
          <div key={section.titleKey} className={styles.navSection}>
            <p className={styles.navSectionTitle}>{t(section.titleKey)}</p>
            {section.items.map((item) => {
              const active = isTeacherNavActive(location.pathname, item)
              return (
                <Link
                  key={item.to}
                  to={item.to}
                  className={`${styles.navItem} ${active ? styles.navItemActive : ''}`}
                  aria-current={active ? 'page' : undefined}
                  onClick={onNavigate}
                >
                  <NavIcon icon={item.icon} />
                  {t(item.labelKey)}
                </Link>
              )
            })}
          </div>
        ))}
      </div>

      <div className={styles.sidebarBottom}>
        <div className={styles.userProfile}>
          <span className={styles.userName}>
            {currentUser?.nickname || currentUser?.username || t('nav.teacher.brandSubtitle')}
          </span>
          <span className={styles.userRole}>
            {currentUser ? localizedRoleName(currentUser.role) : t('nav.teacher.defaultRole')}
          </span>
        </div>
        <button
          className={styles.logoutButton}
          type="button"
          onClick={() => {
            onNavigate?.()
            void logout()
            navigate('/login')
          }}
        >
          {t('common.signOut')}
        </button>
      </div>
    </aside>
  )
}
