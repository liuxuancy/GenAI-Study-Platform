import { MenuOutlined } from '@ant-design/icons'

import { LanguageSwitcher } from '@/core/i18n/LanguageSwitcher'
import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/modules/student/layout/StudentLayout.module.css'
import { teacherPageTitle } from '@/modules/teacher/layout/teacherNav'

interface TeacherTopbarProps {
  pathname: string
  onMenuClick: () => void
}

export function TeacherTopbar({ pathname, onMenuClick }: TeacherTopbarProps) {
  const { t } = useI18n()

  return (
    <header className={styles.topbar}>
      <div className={styles.topbarLeft}>
        <button
          className={styles.menuButton}
          type="button"
          aria-label={t('nav.common.openMenu')}
          onClick={onMenuClick}
        >
          <MenuOutlined />
        </button>
        <h1 className={styles.pageTitle}>{teacherPageTitle(pathname, t)}</h1>
      </div>
      <div className={styles.topbarRight}>
        <LanguageSwitcher compact />
      </div>
    </header>
  )
}
