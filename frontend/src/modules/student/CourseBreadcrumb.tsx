import { Link } from 'react-router-dom'

import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/modules/student/CourseBubblePage.module.css'

export function CourseBreadcrumb({ title }: { title: string }) {
  const { t } = useI18n()
  return (
    <nav className={styles.topLeft} aria-label={t('student.bubble.breadcrumbAria')}>
      <Link className={styles.backLink} to="/student">{t('student.bubble.home')}</Link>
      <span className={styles.topSep} aria-hidden="true">&gt;</span>
      <Link className={styles.backLink} to="/student/courses">{t('nav.student.myCourses')}</Link>
      <span className={styles.topSep} aria-hidden="true">&gt;</span>
      <span className={styles.courseTitle} aria-current="page" title={title}>{title}</span>
    </nav>
  )
}
