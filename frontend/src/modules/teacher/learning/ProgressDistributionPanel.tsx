import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/modules/teacher/learning/ProgressDistributionPanel.module.css'

interface ProgressDistributionPanelProps {
  notStarted: number
  inProgress: number
  completed: number
  total: number
}

export function ProgressDistributionPanel({
  notStarted,
  inProgress,
  completed,
  total,
}: ProgressDistributionPanelProps) {
  const { t } = useI18n()
  const segments = [
    {
      key: 'completed',
      value: completed,
      color: '#548b76',
      label: t('teacher.learning.progressCompleted'),
    },
    {
      key: 'inProgress',
      value: inProgress,
      color: '#638aa7',
      label: t('teacher.learning.progressLearning'),
    },
    {
      key: 'notStarted',
      value: notStarted,
      color: '#d8d5cd',
      label: t('teacher.learning.progressNotStarted'),
    },
  ]

  return (
    <section className={styles.panel} aria-label={t('teacher.learning.progressPanel.aria')}>
      <p className={styles.eyebrow}>{t('teacher.learning.progressPanel.eyebrow')}</p>
      <h3 className={styles.title}>{t('teacher.learning.progressPanel.title')}</h3>
      {total === 0 ? (
        <p className={styles.empty}>{t('teacher.learning.progressPanel.empty')}</p>
      ) : (
        <>
          <div className={styles.stack} role="img" aria-label={t('teacher.learning.progressBarAria')}>
            {segments.map((segment) =>
              segment.value > 0 ? (
                <div
                  key={segment.key}
                  className={styles.stackPart}
                  style={{
                    flexGrow: segment.value,
                    background: segment.color,
                  }}
                  title={`${segment.label} ${segment.value}`}
                />
              ) : null,
            )}
          </div>
          <div className={styles.legend}>
            {segments.map((segment) => (
              <div key={segment.key} className={styles.row}>
                <span className={styles.rowLabel}>
                  <span className={styles.dot} style={{ background: segment.color }} />
                  {segment.label}
                </span>
                <span className={styles.rowValue}>{segment.value}</span>
              </div>
            ))}
          </div>
        </>
      )}
    </section>
  )
}
