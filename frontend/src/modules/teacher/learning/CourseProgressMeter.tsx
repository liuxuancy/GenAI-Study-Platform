import { courseProgressPhase } from '@/modules/teacher/learning/learningOverviewStats'
import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/modules/teacher/learning/CourseProgressMeter.module.css'

interface CourseProgressMeterProps {
  percent: number
}

export function CourseProgressMeter({ percent }: CourseProgressMeterProps) {
  const { t } = useI18n()
  const phase = courseProgressPhase(percent)
  const fillClass =
    phase === 'done' ? styles.fillDone : phase === 'active' ? styles.fillActive : styles.fillIdle

  return (
    <div className={styles.wrap}>
      <div className={styles.meta}>
        <span className={styles.phase}>{t(`teacher.learning.meter.${phase}`)}</span>
        <span className={styles.percent}>{percent}%</span>
      </div>
      <div className={styles.track} role="progressbar" aria-valuenow={percent} aria-valuemin={0} aria-valuemax={100}>
        <div className={`${styles.fill} ${fillClass}`} style={{ width: `${percent}%` }} />
      </div>
    </div>
  )
}
