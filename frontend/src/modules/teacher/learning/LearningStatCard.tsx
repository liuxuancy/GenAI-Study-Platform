import type { ReactNode } from 'react'

import styles from '@/modules/teacher/learning/LearningStatCard.module.css'

interface LearningStatCardProps {
  icon: ReactNode
  label: string
  value: string
  hint?: string
  meterPercent?: number
}

export function LearningStatCard({ icon, label, value, hint, meterPercent }: LearningStatCardProps) {
  return (
    <article className={styles.card}>
      <div className={styles.top}>
        <p className={styles.label}>{label}</p>
        <span className={styles.iconWrap} aria-hidden>
          {icon}
        </span>
      </div>
      <p className={styles.value}>{value}</p>
      {hint ? <p className={styles.hint}>{hint}</p> : null}
      {meterPercent != null ? (
        <div className={styles.meter} role="presentation">
          <div className={styles.meterFill} style={{ width: `${Math.min(100, Math.max(0, meterPercent))}%` }} />
        </div>
      ) : null}
    </article>
  )
}
