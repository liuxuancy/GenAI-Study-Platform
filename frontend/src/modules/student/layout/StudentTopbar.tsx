import { MenuOutlined } from '@ant-design/icons'

import { studentPageTitle } from '@/modules/student/layout/studentNav'
import styles from '@/modules/student/layout/StudentLayout.module.css'

interface StudentTopbarProps {
  pathname: string
  onMenuClick: () => void
}

export function StudentTopbar({ pathname, onMenuClick }: StudentTopbarProps) {
  return (
    <header className={styles.topbar}>
      <div className={styles.topbarLeft}>
        <button
          className={styles.menuButton}
          type="button"
          aria-label="打开导航菜单"
          onClick={onMenuClick}
        >
          <MenuOutlined />
        </button>
        <h1 className={styles.pageTitle}>{studentPageTitle(pathname)}</h1>
      </div>
    </header>
  )
}
