import { Link } from 'react-router-dom'

import type { LearningDashboardTask } from '@/modules/student/api'
import { useI18n } from '@/core/i18n/useI18n'
import { localizeLearningTaskTitle, localizeLearningTaskType } from './learningDashboardCopy'
import styles from './learningDashboard.module.css'

export function LearningTaskLink({ task, destination }: { task: LearningDashboardTask; destination: string }) {
  const { t } = useI18n()
  return (
    <div className={styles.task}>
      <span
        className={`${styles.taskCheck} ${task.completed ? styles.taskCheckDone : ''}`}
        role="img"
        aria-label={t(task.completed ? 'student.dashboard.taskCompleted' : 'student.dashboard.taskIncomplete')}
      >
        {task.completed ? '✓' : ''}
      </span>
      <Link className={styles.taskLink} to={destination}>
        <div>
          <strong className={task.completed ? styles.done : ''}>
            {localizeLearningTaskTitle(task.title, t)}
          </strong>
          <small>{task.course}</small>
        </div>
        <span className={styles.taskType}>{localizeLearningTaskType(task.type, t)}</span>
      </Link>
    </div>
  )
}
