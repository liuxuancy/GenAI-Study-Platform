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
  courseSourceLabels,
  courseSourceOf,
  formatStatCount,
  pickContinueCourse,
  pickRecentCourses,
  visibilityText,
} from '@/modules/student/learning/courseSource'
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

  const [aiSuggestion, setAiSuggestion] = useState('')
  const [aiAction, setAiAction] = useState({ title: '', detail: '' })
  const [aiFromModel, setAiFromModel] = useState(false)
  const [aiLoading, setAiLoading] = useState(false)
  const [aiError, setAiError] = useState('')

  const studentName =
    currentUser?.nickname?.trim() || currentUser?.username?.trim() || '同学'

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
        if (!aiFromModel) {
          setAiSuggestion(dashboardData.ruleSuggestion)
          setAiAction(dashboardData.recommendedAction)
        }
        setErrorMessage('')
      })
      .catch((error: Error) => {
        setErrorMessage(error.message)
      })
  }, [aiFromModel])

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

  async function generateAiInsight() {
    setAiLoading(true)
    setAiError('')
    try {
      const result = await postLearningInsight({ provider: aiProvider })
      setAiSuggestion(result.suggestion)
      setAiAction(result.recommendedAction)
      setAiFromModel(true)
    } catch (error) {
      setAiError(error instanceof Error ? error.message : '生成 AI 建议失败')
    } finally {
      setAiLoading(false)
    }
  }

  const statCards = [
    {
      label: '可学习课程',
      value: stats.courseCount,
      icon: <BookOutlined />,
      tone: 'green' as const,
    },
    {
      label: '正在学习',
      value: stats.activeCount,
      icon: <PlayCircleOutlined />,
      tone: 'orange' as const,
    },
    {
      label: '已完成课程',
      value: stats.completedCount,
      icon: <CheckCircleOutlined />,
      tone: 'green' as const,
    },
    {
      label: '待完成测验',
      value: stats.pendingQuizCount,
      icon: <FileTextOutlined />,
      tone: 'purple' as const,
    },
  ]

  if (errorMessage) {
    return (
      <div className={styles.dashboard}>
        <div className={styles.statePanel}>
          <strong>学习中心加载失败</strong>
          <p>{errorMessage}</p>
        </div>
      </div>
    )
  }

  if (!courses || !dashboard) {
    return (
      <div className={styles.dashboard}>
        <div className={styles.skeletonBlock} aria-label="正在加载学习中心" />
      </div>
    )
  }

  return (
    <div className={styles.dashboard}>
      <header className={styles.welcome}>
        <div>
          <div className={styles.eyebrow}>YOUR LEARNING SPACE</div>
          <h1>
            每一天，都有新的收获<span>.</span>
          </h1>
          <p>你好，{studentName}。让我们继续今天的学习旅程。</p>
        </div>
        <button className={styles.primaryButton} type="button" onClick={() => setCreateOpen(true)}>
          <PlusOutlined /> 创建课程
        </button>
      </header>

      <CreateCourseDialog open={createOpen} onClose={() => setCreateOpen(false)} />

      <section className={styles.statsGrid}>
        {statCards.map((stat) => (
          <div className={styles.statCard} key={stat.label}>
            <div className={styles.statTop}>
              <span>{stat.label}</span>
              <span className={styles[stat.tone]}>{stat.icon}</span>
            </div>
            <strong>{formatStatCount(stat.value)}</strong>
            <small>基于当前课程与测验数据</small>
          </div>
        ))}
      </section>

      <section className={styles.mainGrid}>
        <article className={styles.featured}>
          <div className={styles.featuredContent}>
            <div className={styles.featuredEyebrow}>✦ PICK UP WHERE YOU LEFT OFF</div>
            <h2>{continueCourse?.title ?? '开始你的学习旅程'}</h2>
            <p>
              {continueCourse?.description ||
                (continueCourse
                  ? '继续探索知识的脉络，巩固已学习的内容'
                  : '从课程库中选择一门课程，开启学习。')}
            </p>

            {continueCourse ? (
              <div className={styles.courseTags}>
                <span>{courseTopicLabel(continueCourse.title, continueCourse.description)}</span>
                <span>{courseSourceLabels[courseSourceOf(continueCourse)]}</span>
                <span>{visibilityText(continueCourse.visibility)}</span>
              </div>
            ) : null}
          </div>

          <div className={styles.featuredFooter}>
            {continueCourse ? (
              <div className={styles.featuredProgress}>
                <div className={styles.progressMeta}>
                  <span>课程学习进度</span>
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
              {continueCourse ? (continueCourse.percent > 0 ? '继续学习' : '开始学习') : '探索课程'}
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
                <h3>今日学习计划</h3>
                <p>系统根据你的课程与测验进度自动生成，可在此浏览全部待办</p>
              </div>
              <span className={styles.accentText}>
                {completedTasks}/{tasks.length} 已完成
              </span>
            </div>

            {tasks.length === 0 ? (
              <div className={styles.placeholderBox}>
                <p>暂无待办。继续课程学习或完成测验后，计划会出现在这里。</p>
              </div>
            ) : (
              <div className={styles.tasksScroll} tabIndex={0} aria-label="今日待办列表，可滚动查看">
                <div className={styles.tasks}>
                  {tasks.map((task) => (
                    <div className={styles.task} key={task.id}>
                      <input type="checkbox" checked={task.completed} readOnly aria-label="完成状态" />
                      <div>
                        <strong className={task.completed ? styles.done : ''}>{task.title}</strong>
                        <small>{task.course}</small>
                      </div>
                      <span className={styles.taskType}>{task.type}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className={styles.taskTip}>
              <FireOutlined />
              {completedTasks === tasks.length && tasks.length > 0
                ? '今天的任务全部完成，做得很好！'
                : '完成一项任务，就离目标更近一步。'}
            </div>
          </div>
        </article>
      </section>

      <section className={styles.mainGrid}>
        <article className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <h3>最近在学</h3>
              <p>继续你的知识探索</p>
            </div>
            <button
              className={styles.textButton}
              type="button"
              onClick={() => navigate('/student/courses')}
            >
              全部课程 <ArrowRightOutlined />
            </button>
          </div>

          {recentCourses.length === 0 ? (
            <div className={styles.placeholderBox}>
              <p>还没有课程。创建或加入一门课程后，会出现在这里。</p>
            </div>
          ) : (
            <div className={styles.courseList}>
              {recentCourses.map((course) => {
                const accent = courseDashboardAccent(course.title, course.description)
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
                        {course.description || courseTopicLabel(course.title, course.description)}
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
                <RobotOutlined /> AI 学习建议
              </h3>
              <p>{aiFromModel ? '由模型根据你的学习摘要生成' : '基于当前进度与测验状态'}</p>
            </div>
            <button
              className={styles.aiGenerate}
              type="button"
              disabled={aiLoading}
              onClick={() => void generateAiInsight()}
            >
              {aiLoading ? '生成中…' : '生成建议'}
            </button>
          </div>

          <p className={styles.aiText}>{aiSuggestion}</p>

          <div className={styles.aiAction}>
            <div>
              <span className={styles.aiActionLabel}>今日推荐</span>
              <strong>{aiAction.title}</strong>
              <p>{aiAction.detail}</p>
            </div>
          </div>

          <div className={styles.aiFooter}>
            <div className={styles.aiProvider}>{aiProviderSelector}</div>
            <button
              className={styles.textButton}
              type="button"
              onClick={() => navigate('/student/quizzes')}
            >
              前往测验中心 <ArrowRightOutlined />
            </button>
          </div>
          {aiProviderError ? <p className={styles.aiError}>{aiProviderError}</p> : null}
          {aiError ? <p className={styles.aiError}>{aiError}</p> : null}
        </article>
      </section>

      <footer className={styles.footer}>
        <span>© CaveClass · Learn with intention.</span>
        <span>Student Workspace</span>
      </footer>
    </div>
  )
}
