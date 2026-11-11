import { Button } from 'antd'

import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/modules/student/gal/GalGameScreen.module.css'

export function GalChapterComplete({
  hasNextChapter,
  courseRunMode,
  onNextChapter,
  onReplay,
  onExit,
}: {
  hasNextChapter: boolean
  courseRunMode: boolean
  onNextChapter: () => void
  onReplay: () => void
  onExit: () => void
}) {
  const { t } = useI18n()
  return (
    <div className={styles.completeOverlay}>
      <div className={styles.completeCard}>
        <div className={styles.completeIcon} aria-hidden="true">
          ✓
        </div>
        <h2>{t('student.gal.chapterCompleteTitle')}</h2>
        <p>{t('student.gal.chapterCompleteDesc')}</p>
        <div className={styles.completeActions}>
          {hasNextChapter ? (
            <Button type="primary" onClick={onNextChapter}>
              {courseRunMode ? t('student.gal.continueNextChapter') : t('student.gal.nextChapter')}
            </Button>
          ) : (
            <p className={styles.courseDone}>{t('student.gal.courseStoryDone')}</p>
          )}
          <Button onClick={onReplay}>{t('student.gal.replayChapter')}</Button>
          <Button type="link" onClick={onExit}>
            {t('student.gal.exitPlay')}
          </Button>
        </div>
      </div>
    </div>
  )
}
