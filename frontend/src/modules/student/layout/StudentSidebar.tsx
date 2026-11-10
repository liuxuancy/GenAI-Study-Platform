import {
  AppstoreOutlined,
  BookOutlined,
  CheckSquareOutlined,
  CommentOutlined,
  TrophyOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { Link, useLocation, useNavigate } from 'react-router-dom'

import { studentHomePath } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'
import { localizedRoleName } from '@/core/i18n/roleName'
import { useI18n } from '@/core/i18n/useI18n'

import {
  isStudentNavActive,
  studentNavSections,
  type StudentNavItem,
} from '@/modules/student/layout/studentNav'
import styles from '@/modules/student/layout/StudentLayout.module.css'

function NavIcon({ icon }: { icon: StudentNavItem['icon'] }) {
  if (icon === 'dashboard') {
    return <AppstoreOutlined className={styles.navIcon} aria-hidden />
  }
  if (icon === 'book') {
    return <BookOutlined className={styles.navIcon} aria-hidden />
  }
  if (icon === 'clipboard') {
    return <CheckSquareOutlined className={styles.navIcon} aria-hidden />
  }
  if (icon === 'spark') {
    return <CommentOutlined className={styles.navIcon} aria-hidden />
  }
  if (icon === 'trophy') {
    return <TrophyOutlined className={styles.navIcon} aria-hidden />
  }
  return <UserOutlined className={styles.navIcon} aria-hidden />
}

interface StudentSidebarProps {
  open: boolean
  onNavigate?: () => void
}

export function StudentSidebar({ open, onNavigate }: StudentSidebarProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const currentUser = useAuthStore((state) => state.currentUser)
  const logout = useAuthStore((state) => state.logout)
  const { t } = useI18n()

  return (
    <aside
      className={`${styles.sidebar} ${open ? styles.sidebarOpen : styles.sidebarClosed}`}
      aria-label={t('nav.student.sidebarAria')}
    >
      <Link className={styles.brand} to={studentHomePath} onClick={onNavigate}>
        <span className={styles.mark} aria-hidden="true">
          C
        </span>
        <span className={styles.brandText}>
          <strong>{t('common.brandName')}</strong>
          <small>{t('nav.student.brandSubtitle')}</small>
        </span>
      </Link>

      <div className={styles.navScroll}>
        {studentNavSections.map((section) => (
          <div key={section.titleKey} className={styles.navSection}>
            <p className={styles.navSectionTitle}>{t(section.titleKey)}</p>
            {section.items.map((item) => {
              const active = isStudentNavActive(location.pathname, item)
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
            {currentUser?.nickname || currentUser?.username || t('nav.student.brandSubtitle')}
          </span>
          <span className={styles.userRole}>
            {currentUser ? localizedRoleName(currentUser.role) : t('roles.ROLE_STUDENT')}
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
