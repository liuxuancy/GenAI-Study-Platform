import { MenuOutlined } from '@ant-design/icons'

import styles from '@/modules/student/layout/StudentLayout.module.css'
import { teacherPageTitle } from '@/modules/teacher/layout/teacherNav'

interface TeacherTopbarProps {
  pathname: string
  onMenuClick: () => void
}

export function TeacherTopbar({ pathname, onMenuClick }: TeacherTopbarProps) {
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
        <h1 className={styles.pageTitle}>{teacherPageTitle(pathname)}</h1>
      </div>
    </header>
  )
}
