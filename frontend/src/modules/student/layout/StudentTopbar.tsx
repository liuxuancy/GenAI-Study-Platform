import { MenuOutlined } from '@ant-design/icons'

import { LanguageSwitcher } from '@/core/i18n/LanguageSwitcher'
import { studentPageTitle } from '@/modules/student/layout/studentNav'
import styles from '@/modules/student/layout/StudentLayout.module.css'
import { useI18n } from '@/core/i18n/useI18n'

interface StudentTopbarProps {
  pathname: string
  onMenuClick: () => void
}

export function StudentTopbar({ pathname, onMenuClick }: StudentTopbarProps) {
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
        <h1 className={styles.pageTitle}>{studentPageTitle(pathname, t)}</h1>
      </div>
      <div className={styles.topbarRight}>
        <LanguageSwitcher compact />
      </div>
    </header>
  )
}
