import { useEffect, useRef, useState } from 'react'
import { Alert, Button, Select, Spin, Switch } from 'antd'

import { useI18n } from '@/core/i18n/useI18n'
import { generateStudentGalScript } from '@/modules/student/api'
import { GalGameScreen } from '@/modules/student/gal/GalGameScreen'
import {
  defaultGalSettings,
  mapGalScriptFromApi,
  type GalGameSettings,
  type GalScriptResponse,
} from '@/modules/student/gal/galTypes'
import { GalStorySettingsModal } from '@/modules/student/gal/GalStorySettingsModal'
import styles from '@/modules/student/gal/GalStoryPanel.module.css'

type Phase = 'idle' | 'generating' | 'playing'

export type GalStoryChapterOption = {
  key: string
  title: string
  index: number
}

export function GalStoryPanel({
  courseId,
  chapters,
  chapterKey,
  onChapterKeyChange,
  activityKey,
  provider,
  courseRunMode,
  onCourseRunModeChange,
  onChapterComplete,
  onReplayChapter,
}: {
  courseId: string
  chapters: GalStoryChapterOption[]
  chapterKey: string
  onChapterKeyChange: (key: string) => void
  activityKey: string
  provider?: string
  courseRunMode: boolean
  onCourseRunModeChange: (value: boolean) => void
  onChapterComplete: (chapterKey: string) => void | Promise<void>
  onReplayChapter?: () => void
}) {
  const { t } = useI18n()
  const [phase, setPhase] = useState<Phase>('idle')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [settings, setSettings] = useState<GalGameSettings>(defaultGalSettings)
  const [script, setScript] = useState<GalScriptResponse | null>(null)
  const [error, setError] = useState('')
  const contextKeyRef = useRef('')

  useEffect(() => {
    if (contextKeyRef.current && contextKeyRef.current !== chapterKey && phase === 'playing') {
      setPhase('idle')
      setScript(null)
      setError('')
    }
    contextKeyRef.current = chapterKey
  }, [chapterKey, phase])

  async function runGenerate(nextSettings: GalGameSettings, targetChapterKey = chapterKey) {
    if (!targetChapterKey) {
      setError(t('student.gal.noChapterKey'))
      return
    }
    setSettings(nextSettings)
    setSettingsOpen(false)
    setPhase('generating')
    setError('')
    try {
      const result = await generateStudentGalScript(courseId, targetChapterKey, {
        provider,
        activityKey: activityKey || undefined,
        detailLevel: nextSettings.detailLevel,
        personality: nextSettings.personality,
      })
      setScript({
        title: result.title,
        chapterKey: result.chapterKey ?? targetChapterKey,
        nextChapterKey: result.nextChapterKey,
        script: mapGalScriptFromApi(result.script),
      })
      setPhase('playing')
    } catch (caught) {
      setPhase('idle')
      setError(caught instanceof Error ? caught.message : t('student.gal.generateFailed'))
    }
  }

  if (phase === 'playing' && script) {
    return (
      <div className={styles.playStage}>
        <GalGameScreen
        script={script.script}
        title={script.title}
        nextChapterKey={script.nextChapterKey}
        courseRunMode={courseRunMode}
        onChapterComplete={() => {
          void onChapterComplete(script.chapterKey ?? chapterKey)
        }}
        onNextChapter={(nextKey) => {
          onChapterKeyChange(nextKey)
          if (courseRunMode) {
            void runGenerate(settings, nextKey)
            return
          }
          setPhase('idle')
          setScript(null)
        }}
        onReplay={() => {
          setPhase('idle')
          setScript(null)
          onReplayChapter?.()
          setSettingsOpen(true)
        }}
        onExit={() => {
          setPhase('idle')
          setScript(null)
        }}
      />
      </div>
    )
  }

  const chapterOptions = chapters.map((chapter) => ({
    value: chapter.key,
    label: `${String(chapter.index + 1).padStart(2, '0')} · ${chapter.title}`,
  }))

  return (
    <div className={styles.galPanel}>
      <div className={styles.toolbar}>
        <div className={styles.toolbarMain}>
          <label className={styles.chapterPick}>
            <span>{t('student.gal.pickChapter')}</span>
            <Select
              value={chapterKey || undefined}
              options={chapterOptions}
              placeholder={t('student.gal.pickChapterPlaceholder')}
              onChange={(value) => onChapterKeyChange(value)}
              style={{ minWidth: 220 }}
            />
          </label>
          <label className={styles.runMode}>
            <Switch checked={courseRunMode} onChange={onCourseRunModeChange} />
            <span>{t('student.gal.courseRunMode')}</span>
          </label>
        </div>
        <Button type="primary" disabled={!chapterKey || phase === 'generating'} onClick={() => setSettingsOpen(true)}>
          {phase === 'generating' ? t('student.gal.generating') : t('student.gal.generateButton')}
        </Button>
      </div>

      {phase === 'generating' ? (
        <div className={`${styles.idle} ${styles.idleGenerating}`}>
          <span className={styles.idleBadge}>{t('student.gal.generating')}</span>
          <Spin size="large" />
          <p className={styles.idleHint}>{t('student.gal.generatingHint')}</p>
        </div>
      ) : (
        <div className={styles.idle}>
          <span className={styles.idleBadge}>{t('student.bubble.tabStory')}</span>
          <div className={styles.idleTitle}>{t('student.gal.idleTitle')}</div>
          <p className={styles.idleDesc}>{t('student.gal.idleDescChapter')}</p>
          {error ? <Alert type="error" showIcon message={error} className={styles.error} /> : null}
        </div>
      )}

      <GalStorySettingsModal
        open={settingsOpen}
        settings={settings}
        onCancel={() => setSettingsOpen(false)}
        onConfirm={(next) => {
          void runGenerate(next)
        }}
      />
    </div>
  )
}
