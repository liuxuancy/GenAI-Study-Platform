import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/modules/teacher/learning/QuizCompletionPanel.module.css'

interface QuizCompletionPanelProps {
  slots: number
  submitted: number
  inProgress: number
  notStarted: number
  completionRate: number
}

function barWidth(value: number, total: number) {
  if (total === 0) {
    return 0
  }
  return Math.round((value / total) * 100)
}

export function QuizCompletionPanel({
  slots,
  submitted,
  inProgress,
  notStarted,
  completionRate,
}: QuizCompletionPanelProps) {
  const { t } = useI18n()
  const rows = [
    {
      key: 'submitted',
      label: t('teacher.learning.quizSubmitted'),
      value: submitted,
      className: styles.barFillSubmitted,
    },
    {
      key: 'progress',
      label: t('teacher.learning.quizInProgress'),
      value: inProgress,
      className: styles.barFillProgress,
    },
    {
      key: 'idle',
      label: t('teacher.learning.quizNotStarted'),
      value: notStarted,
      className: styles.barFillIdle,
    },
  ] as const

  return (
    <section className={styles.panel} aria-label={t('teacher.learning.quizPanel.aria')}>
      <p className={styles.eyebrow}>{t('teacher.learning.quizPanel.eyebrow')}</p>
      <h3 className={styles.title}>{t('teacher.learning.quizPanel.title')}</h3>
      {slots === 0 ? (
        <p className={styles.empty}>{t('teacher.learning.quizPanel.empty')}</p>
      ) : (
        <>
          <p className={styles.rate}>{completionRate}%</p>
          <p className={styles.rateHint}>
            {t('teacher.learning.quizPanel.rateHint', { submitted, slots })}
          </p>
          <div className={styles.bars}>
            {rows.map((row) => (
              <div key={row.key} className={styles.barRow}>
                <span className={styles.barLabel}>{row.label}</span>
                <div className={styles.barTrack}>
                  <div
                    className={row.className}
                    style={{ width: `${barWidth(row.value, slots)}%` }}
                  />
                </div>
                <span className={styles.barValue}>{row.value}</span>
              </div>
            ))}
          </div>
        </>
      )}
    </section>
  )
}
