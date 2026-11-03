import type { QuizDisplayStatus } from '@/modules/teacher/learning/learningOverviewStats'
import styles from '@/modules/teacher/learning/QuizStatusTag.module.css'

interface QuizStatusTagProps {
  status: QuizDisplayStatus
  label: string
}

export function QuizStatusTag({ status, label }: QuizStatusTagProps) {
  const tone =
    status === 'submitted' ? styles.submitted : status === 'progress' ? styles.progress : styles.idle
  return <span className={`${styles.tag} ${tone}`}>{label}</span>
}
