import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ArrowRightOutlined,
  BookOutlined,
  CheckCircleOutlined,
  FileTextOutlined,
  FireOutlined,
  PlusOutlined,
  PlayCircleOutlined,
  RobotOutlined,
} from '@ant-design/icons'

import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'
import {
  getLearningDashboard,
  getStudentCourses,
  getStudentQuestions,
  postLearningInsight,
  type LearningDashboardData,
  type StudentCourse,
} from '@/modules/student/api'
import { CreateCourseDialog } from '@/modules/student/learning/CreateCourseDialog'
import {
  readLearningDashboardAiInsight,
  writeLearningDashboardAiInsight,
} from '@/modules/student/learning/learningDashboardAiInsight'
import {
  courseSourceLabel,
  courseSourceOf,
  formatStatCount,
  pickContinueCourse,
  pickRecentCourses,
  visibilityText,
} from '@/modules/student/learning/courseSource'
import {
  localizeLearningTaskTitle,
  localizeLearningTaskType,
  localizeRecommendedAction,
  localizeRuleSuggestion,
} from '@/modules/student/learning/learningDashboardCopy'
import styles from '@/modules/student/learning/learningDashboard.module.css'
import {
  mapStudentQuestionsToViews,
  quizStatistics,
} from '@/modules/student/quiz/mapStudentQuiz'
import {
  courseDashboardAccent,
  courseTopicLabel,
} from '@/modules/student/studentCourseVisuals'

export function LearningDashboard() {
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const currentUser = useAuthStore((state) => state.currentUser)
  const {
    provider: aiProvider,
    selector: aiProviderSelector,
    error: aiProviderError,
  } = useAiProviderSelect()

  const [courses, setCourses] = useState<StudentCourse[] | null>(null)
  const [dashboard, setDashboard] = useState<LearningDashboardData | null>(null)
  const [pendingQuizCount, setPendingQuizCount] = useState(0)
  const [errorMessage, setErrorMessage] = useState('')
  const [createOpen, setCreateOpen] = useState(false)

  const studentUserId = currentUser?.id

  const [aiSuggestion, setAiSuggestion] = useState('')
  const [aiAction, setAiAction] = useState({ title: '', detail: '' })
  const [aiFromModel, setAiFromModel] = useState(false)
  const [aiLoading, setAiLoading] = useState(false)
  const [aiError, setAiError] = useState('')

  useEffect(() => {
    const stored = readLearningDashboardAiInsight(studentUserId)
    if (!stored) {
      return
    }
    setAiSuggestion(stored.suggestion)
    setAiAction(stored.recommendedAction)
    setAiFromModel(true)
  }, [studentUserId])

  const studentName =
    currentUser?.nickname?.trim() || currentUser?.username?.trim() || t('student.defaultName')

  const loadData = useCallback(() => {
    return Promise.all([
      getStudentCourses(),
      getStudentQuestions(),
      getLearningDashboard(),
    ])
      .then(([courseItems, questionItems, dashboardData]) => {
        setCourses(courseItems)
        const views = mapStudentQuestionsToViews(questionItems, courseItems)
        setPendingQuizCount(quizStatistics(views).pending)
        setDashboard(dashboardData)
        if (!readLearningDashboardAiInsight(studentUserId)) {
          setAiSuggestion(dashboardData.ruleSuggestion)
          setAiAction(dashboardData.recommendedAction)
          setAiFromModel(false)
        }
        setErrorMessage('')
      })
      .catch((error: Error) => {
        setErrorMessage(error.message)
      })
  }, [studentUserId])

  useEffect(() => {
    let cancelled = false
    void loadData().then(() => {
      if (cancelled) {
        return
      }
    })
    return () => {
      cancelled = true
    }
  }, [loadData])

  const continueCourse = useMemo(
    () => (courses ? pickContinueCourse(courses) : null),
    [courses],
  )
  const recentCourses = useMemo(
    () => (courses ? pickRecentCourses(courses, 3) : []),
    [courses],
  )

  const stats = useMemo(() => {
    const list = courses ?? []
    return {
      courseCount: list.length,
      activeCount: list.filter((course) => course.percent > 0 && course.percent < 100).length,
      completedCount: list.filter((course) => course.percent === 100).length,
      pendingQuizCount,
    }
  }, [courses, pendingQuizCount])

  const tasks = dashboard?.tasks ?? []
  const completedTasks = tasks.filter((task) => task.completed).length
  const displayedSuggestion = aiFromModel ? aiSuggestion : localizeRuleSuggestion(aiSuggestion, t)
  const displayedAction = aiFromModel ? aiAction : localizeRecommendedAction(aiAction, t)

  async function generateAiInsight() {
    setAiLoading(true)
    setAiError('')
    try {
      const result = await postLearningInsight({
        provider: aiProvider,
        language: locale === 'zh' ? 'zh-CN' : 'en-US',
      })
      setAiSuggestion(result.suggestion)
      setAiAction(result.recommendedAction)
      setAiFromModel(true)
      writeLearningDashboardAiInsight(studentUserId, {
        suggestion: result.suggestion,
        recommendedAction: result.recommendedAction,
      })
    } catch (error) {
      setAiError(error instanceof Error ? error.message : t('student.dashboard.aiGenerateFailed'))
    } finally {
      setAiLoading(false)
    }
  }

  const statCards = [
    {
      id: 'courses',
      label: t('student.dashboard.statCourses'),
      value: stats.courseCount,
      icon: <BookOutlined />,
      tone: 'green' as const,
    },
    {
      id: 'active',
      label: t('student.dashboard.statActive'),
      value: stats.activeCount,
      icon: <PlayCircleOutlined />,
      tone: 'orange' as const,
    },
    {
      id: 'completed',
      label: t('student.dashboard.statCompleted'),
      value: stats.completedCount,
      icon: <CheckCircleOutlined />,
      tone: 'green' as const,
    },
    {
      id: 'quiz',
      label: t('student.dashboard.statPendingQuiz'),
      value: stats.pendingQuizCount,
      icon: <FileTextOutlined />,
      tone: 'purple' as const,
    },
  ]

  if (errorMessage) {
    return (
      <div className={styles.dashboard}>
        <div className={styles.statePanel}>
          <strong>{t('student.dashboard.loadFailed')}</strong>
          <p>{errorMessage}</p>
        </div>
      </div>
    )
  }

  if (!courses || !dashboard) {
    return (
      <div className={styles.dashboard}>
        <div className={styles.skeletonBlock} aria-label={t('student.dashboard.loadingAria')} />
      </div>
    )
  }

  return (
    <div className={styles.dashboard}>
      <header className={styles.welcome}>
        <div>
          <div className={styles.eyebrow}>{t('student.dashboard.eyebrow')}</div>
          <h1>
            {t('student.dashboard.title')}
            <span>.</span>
          </h1>
          <p>{t('student.dashboard.greeting', { name: studentName })}</p>
        </div>
        <button className={styles.primaryButton} type="button" onClick={() => setCreateOpen(true)}>
          <PlusOutlined /> {t('student.dashboard.createCourse')}
        </button>
      </header>

      <CreateCourseDialog open={createOpen} onClose={() => setCreateOpen(false)} />

      <section className={styles.statsGrid}>
        {statCards.map((stat) => (
          <div className={styles.statCard} key={stat.id}>
            <div className={styles.statTop}>
              <span>{stat.label}</span>
              <span className={styles[stat.tone]}>{stat.icon}</span>
            </div>
            <strong>{formatStatCount(stat.value)}</strong>
            <small>{t('student.dashboard.statHint')}</small>
          </div>
        ))}
      </section>

      <section className={styles.mainGrid}>
        <article className={styles.featured}>
          <div className={styles.featuredContent}>
            <div className={styles.featuredEyebrow}>{t('student.dashboard.continueEyebrow')}</div>
            <h2>{continueCourse?.title ?? t('student.dashboard.continueDefaultTitle')}</h2>
            <p>
              {continueCourse?.description ||
                (continueCourse
                  ? t('student.dashboard.continueDesc')
                  : t('student.dashboard.continueEmpty'))}
            </p>

            {continueCourse ? (
              <div className={styles.courseTags}>
                <span>{courseTopicLabel(continueCourse.title, continueCourse.description, locale, continueCourse.id)}</span>
                <span>{courseSourceLabel(courseSourceOf(continueCourse))}</span>
                <span>{visibilityText(continueCourse.visibility)}</span>
              </div>
            ) : null}
          </div>

          <div className={styles.featuredFooter}>
            {continueCourse ? (
              <div className={styles.featuredProgress}>
                <div className={styles.progressMeta}>
                  <span>{t('student.dashboard.progressLabel')}</span>
                  <b>{continueCourse.percent}%</b>
                </div>
                <div className={styles.progressTrack}>
                  <div style={{ width: `${continueCourse.percent}%` }} />
                </div>
              </div>
            ) : (
              <span />
            )}
            <button
              className={`${styles.coralButton} ${styles.featuredAction}`}
              type="button"
              onClick={() =>
                continueCourse
                  ? navigate(`/student/courses/${continueCourse.id}`)
                  : navigate('/student/courses')
              }
            >
              {continueCourse
                ? continueCourse.percent > 0
                  ? t('student.dashboard.continueLearning')
                  : t('student.dashboard.startLearning')
                : t('student.dashboard.exploreCourses')}
              <ArrowRightOutlined />
            </button>
          </div>

          <div className={styles.featuredArt} aria-hidden="true">
            <div className={styles.artCircle} />
            <div className={styles.artWindow}>
              <i />
              <i />
              <i />
              <svg viewBox="0 0 160 55">
                <path
                  d="M5 45 C30 0, 48 65, 77 23 S122 55, 154 8"
                  fill="none"
                  stroke="#be7792"
                  strokeWidth="5"
                  strokeLinecap="round"
                />
              </svg>
            </div>
          </div>
        </article>

        <article className={`${styles.panel} ${styles.tasksPanel}`}>
          <div className={styles.tasksPanelBody}>
            <div className={styles.panelHeader}>
              <div>
                <h3>{t('student.dashboard.tasksTitle')}</h3>
                <p>{t('student.dashboard.tasksSub')}</p>
              </div>
              <span className={styles.accentText}>
                {t('student.dashboard.tasksDone', {
                  done: completedTasks,
                  total: tasks.length,
                })}
              </span>
            </div>

            {tasks.length === 0 ? (
              <div className={styles.placeholderBox}>
                <p>{t('student.dashboard.tasksEmpty')}</p>
              </div>
            ) : (
              <div
                className={styles.tasksScroll}
                tabIndex={0}
                aria-label={t('student.dashboard.tasksListAria')}
              >
                <div className={styles.tasks}>
                  {tasks.map((task) => (
                    <div className={styles.task} key={task.id}>
                      <input
                        type="checkbox"
                        checked={task.completed}
                        readOnly
                        aria-label={t('student.dashboard.taskCompleteAria')}
                      />
                      <div>
                        <strong className={task.completed ? styles.done : ''}>
                          {localizeLearningTaskTitle(task.title, t)}
                        </strong>
                        <small>{task.course}</small>
                      </div>
                      <span className={styles.taskType}>{localizeLearningTaskType(task.type, t)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className={styles.taskTip}>
              <FireOutlined />
              {completedTasks === tasks.length && tasks.length > 0
                ? t('student.dashboard.tasksAllDone')
                : t('student.dashboard.taskTip')}
            </div>
          </div>
        </article>
      </section>

      <section className={styles.mainGrid}>
        <article className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <h3>{t('student.dashboard.recentTitle')}</h3>
              <p>{t('student.dashboard.recentSub')}</p>
            </div>
            <button
              className={styles.textButton}
              type="button"
              onClick={() => navigate('/student/courses')}
            >
              {t('student.dashboard.allCourses')} <ArrowRightOutlined />
            </button>
          </div>

          {recentCourses.length === 0 ? (
            <div className={styles.placeholderBox}>
              <p>{t('student.dashboard.recentEmpty')}</p>
            </div>
          ) : (
            <div className={styles.courseList}>
              {recentCourses.map((course) => {
                const accent = courseDashboardAccent(course.title, course.description, course.id)
                return (
                  <button
                    className={styles.courseRow}
                    key={course.id}
                    type="button"
                    onClick={() => navigate(`/student/courses/${course.id}`)}
                  >
                    <div
                      className={styles.courseIcon}
                      style={{ backgroundColor: accent.color }}
                    >
                      {accent.symbol}
                    </div>
                    <div className={styles.courseInfo}>
                      <strong>{course.title}</strong>
                      <span>
                        {course.description || courseTopicLabel(course.title, course.description, locale, course.id)}
                      </span>
                      <div className={styles.smallTrack}>
                        <div style={{ width: `${course.percent}%` }} />
                      </div>
                    </div>
                    <div className={styles.courseArrow}>
                      <b>{course.percent}%</b>
                      <ArrowRightOutlined />
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </article>

        <article className={`${styles.panel} ${styles.aiPanel}`}>
          <div className={styles.panelHeader}>
            <div>
              <h3>
                <RobotOutlined /> {t('student.dashboard.aiTitle')}
              </h3>
              <p>{aiFromModel ? t('student.dashboard.aiSubModel') : t('student.dashboard.aiSubRule')}</p>
            </div>
            <button
              className={styles.aiGenerate}
              type="button"
              disabled={aiLoading}
              onClick={() => void generateAiInsight()}
            >
              {aiLoading ? t('student.dashboard.aiGenerating') : t('student.dashboard.aiGenerate')}
            </button>
          </div>

          <p className={styles.aiText}>{displayedSuggestion}</p>

          <div className={styles.aiAction}>
            <div>
              <span className={styles.aiActionLabel}>{t('student.dashboard.aiTodayPick')}</span>
              <strong>{displayedAction.title}</strong>
              <p>{displayedAction.detail}</p>
            </div>
          </div>

          <div className={styles.aiFooter}>
            <div className={styles.aiProvider}>{aiProviderSelector}</div>
            <button
              className={styles.textButton}
              type="button"
              onClick={() => navigate('/student/quizzes')}
            >
              {t('student.dashboard.goQuizzes')} <ArrowRightOutlined />
            </button>
          </div>
          {aiProviderError ? <p className={styles.aiError}>{aiProviderError}</p> : null}
          {aiError ? <p className={styles.aiError}>{aiError}</p> : null}
        </article>
      </section>

      <footer className={styles.footer}>
        <span>{t('student.dashboard.footerCopy')}</span>
        <span>{t('student.dashboard.footerTag')}</span>
      </footer>
    </div>
  )
}
