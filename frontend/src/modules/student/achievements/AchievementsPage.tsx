import { LockOutlined, TrophyFilled } from '@ant-design/icons'
import { Spin } from 'antd'

import { useI18n } from '@/core/i18n/useI18n'
import { useAchievements } from './AchievementProvider'
import { achievementKeys } from './achievementTypes'
import styles from './achievements.module.css'

export function AchievementsPage() {
  const { t } = useI18n()
  const { unlocked, loading } = useAchievements()
  const unlockedKeys = new Set(unlocked.map((achievement) => achievement.key))

  return (
    <section>
      <h2 className={styles.title}>{t('student.achievements.title')}</h2>
      <p className={styles.subtitle}>{t('student.achievements.subtitle')}</p>
      {loading ? <Spin /> : (
        <div className={styles.grid}>
          {achievementKeys.map((key) => unlockedKeys.has(key) ? (
            <article className={styles.card} key={key}>
              <TrophyFilled className={styles.icon} aria-hidden />
              <h3>{t(`student.achievements.${key}.title`)}</h3>
              <p>{t(`student.achievements.${key}.description`)}</p>
            </article>
          ) : (
            <article className={`${styles.card} ${styles.locked}`} key={key}>
              <LockOutlined className={styles.icon} aria-hidden />
              <p>{t('student.achievements.locked')}</p>
            </article>
          ))}
        </div>
      )}
    </section>
  )
}
