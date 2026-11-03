import { CheckCircleOutlined, FileDoneOutlined, TeamOutlined } from '@ant-design/icons'
import { useEffect, useMemo, useState } from 'react'
import { Alert, Select, Skeleton, Tooltip } from 'antd'
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'
import { getAppLocale } from '@/core/i18n/store'
import { CourseProgressMeter } from '@/modules/teacher/learning/CourseProgressMeter'
import { LearningStatCard } from '@/modules/teacher/learning/LearningStatCard'
import {
  averagePercent,
  computeProgressDistribution,
  computeQuizAnalytics,
  countSubmittedAttempts,
  getQuizStat,
  studentDisplayName,
  studentInitial,
} from '@/modules/teacher/learning/learningOverviewStats'
import { ProgressDistributionPanel } from '@/modules/teacher/learning/ProgressDistributionPanel'
import { QuizCompletionPanel } from '@/modules/teacher/learning/QuizCompletionPanel'
import { QuizStatusTag } from '@/modules/teacher/learning/QuizStatusTag'
import {
  getTeacherCourseLearningOverview,
  getTeacherCourses,
  type TeacherCourse,
  type TeacherCourseLearningOverview,
  type TeacherQuizOverviewItem,
} from '@/modules/teacher/api'
import styles from '@/modules/teacher/TeacherLearningOverviewPage.module.css'

function formatUpdatedAt(date: Date) {
  const locale = getAppLocale() === 'zh' ? 'zh-CN' : 'en-US'
  return date.toLocaleString(locale, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function courseStatusLabel(
  course: TeacherCourse | null,
  t: (key: string, params?: Record<string, string | number>) => string,
) {
  if (!course) {
    return '—'
  }
  if (!course.published) {
    return t('teacher.learning.draft')
  }
  if (course.classIds.length === 0) {
    return t('teacher.learning.publishedNoClass')
  }
  return t('teacher.learning.published')
}

function QuizColumnHeader({ quiz }: { quiz: TeacherQuizOverviewItem }) {
  const title = quiz.title
  if (title.length <= 28) {
    return <span>{title}</span>
  }
  return (
    <Tooltip title={title} placement="topLeft">
      <span>{title}</span>
    </Tooltip>
  )
}

export function TeacherLearningOverviewPage() {
  const { t } = useI18n()
  const currentUser = useAuthStore((state) => state.currentUser)
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [courses, setCourses] = useState<TeacherCourse[] | null>(null)
  const [loadError, setLoadError] = useState('')
  const [overview, setOverview] = useState<TeacherCourseLearningOverview | null>(null)
  const [overviewError, setOverviewError] = useState('')
  const [overviewLoading, setOverviewLoading] = useState(false)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)

  const courseIdParam = searchParams.get('courseId')
  const selectedCourseId = courseIdParam ? Number(courseIdParam) : null
  const selectedCourse = useMemo(() => {
    if (!courses || selectedCourseId == null || Number.isNaN(selectedCourseId)) {
      return null
    }
    return courses.find((item) => item.id === selectedCourseId) ?? null
  }, [courses, selectedCourseId])

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_TEACHER') {
      return
    }
    let cancelled = false
    getTeacherCourses()
      .then((items) => {
        if (!cancelled) {
          setCourses(items)
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setLoadError(error.message)
        }
      })
    return () => {
      cancelled = true
    }
  }, [currentUser?.role])

  useEffect(() => {
    if (courses === null || courses.length === 0) {
      return
    }
    if (selectedCourseId != null && !Number.isNaN(selectedCourseId) && selectedCourse) {
      return
    }
    const fallback = courses[0]?.id
    if (fallback != null && courseIdParam == null) {
      setSearchParams({ courseId: String(fallback) }, { replace: true })
    }
  }, [courseIdParam, courses, selectedCourse, selectedCourseId, setSearchParams])

  useEffect(() => {
    if (selectedCourseId == null || Number.isNaN(selectedCourseId) || !selectedCourse) {
      setOverview(null)
      return
    }
    let cancelled = false
    setOverviewLoading(true)
    setOverviewError('')
    getTeacherCourseLearningOverview(selectedCourseId)
      .then((data) => {
        if (!cancelled) {
          setOverview(data)
          setLastUpdated(new Date())
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setOverview(null)
          setOverviewError(error.message)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setOverviewLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [selectedCourse, selectedCourseId])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_TEACHER') {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }

  const courseOptions = (courses ?? []).map((course) => ({
    value: course.id,
    label: course.published ? course.title : `${course.title}${t('teacher.learning.draftSuffix')}`,
  }))

  const students = overview?.students ?? []
  const quizzes = overview?.quizzes ?? []
  const avgPercent = averagePercent(students)
  const submittedAttempts = countSubmittedAttempts(students)
  const progressDistribution = computeProgressDistribution(students)
  const quizAnalytics = computeQuizAnalytics(students, quizzes)
  const loading = overviewLoading || courses === null

  const courseMetaParts: string[] = []
  if (selectedCourse) {
    courseMetaParts.push(courseStatusLabel(selectedCourse, t))
    if (selectedCourse.chapterCount > 0) {
      courseMetaParts.push(t('teacher.learning.chapterCount', { count: selectedCourse.chapterCount }))
    }
    if (selectedCourse.classIds.length > 0) {
      courseMetaParts.push(t('teacher.learning.classCountMeta', { count: selectedCourse.classIds.length }))
    }
  }

  return (
    <div className={styles.page}>
      <header className={styles.hero}>
        <div className={styles.heroMain}>
          <p className={styles.eyebrow}>{t('teacher.learning.pageEyebrow')}</p>
          <h1>{t('teacher.learning.pageTitle')}</h1>
          <p className={styles.subtitle}>{t('teacher.learning.pageSubtitle')}</p>
        </div>
        <aside className={styles.heroMeta} aria-label={t('teacher.learning.contextAria')}>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>{t('teacher.learning.metaCurrentCourse')}</span>
            <span className={styles.metaValue}>{selectedCourse?.title ?? '—'}</span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>{t('teacher.learning.metaUpdated')}</span>
            <span className={styles.metaValue}>
              {lastUpdated
                ? formatUpdatedAt(lastUpdated)
                : loading
                  ? t('teacher.learning.loadingShort')
                  : '—'}
            </span>
          </div>
        </aside>
      </header>

      {loadError ? <Alert type="error" showIcon message={loadError} className={styles.alert} /> : null}

      <section className={styles.courseBar} aria-label={t('teacher.learning.courseBarAria')}>
        <div>
          <p className={styles.courseBarLabel}>{t('teacher.learning.courseBarLabel')}</p>
          <p className={styles.courseBarTitle}>{selectedCourse?.title ?? t('teacher.learning.selectCourse')}</p>
          {courseMetaParts.length > 0 ? (
            <p className={styles.courseBarMeta}>{courseMetaParts.join(' · ')}</p>
          ) : null}
        </div>
        <div className={styles.courseSelectWrap}>
          <Select
            id="teacher-learning-course"
            showSearch
            optionFilterProp="label"
            placeholder={t('teacher.learning.switchCourse')}
            loading={courses === null}
            disabled={courseOptions.length === 0}
            options={courseOptions}
            value={selectedCourse?.id}
            onChange={(value) => setSearchParams({ courseId: String(value) })}
            aria-label={t('teacher.learning.switchCourseAria')}
          />
        </div>
      </section>

      {selectedCourse && !selectedCourse.published ? (
        <Alert
          type="info"
          showIcon
          className={styles.alert}
          message={t('teacher.learning.unpublishedTitle')}
          description={t('teacher.learning.unpublishedDesc')}
        />
      ) : null}

      {overviewError ? (
        <Alert type="error" showIcon message={overviewError} className={styles.alert} />
      ) : null}

      {loading ? (
        <div className={styles.skeletonGrid} aria-hidden>
          <Skeleton active paragraph={{ rows: 2 }} />
          <Skeleton active paragraph={{ rows: 2 }} />
          <Skeleton active paragraph={{ rows: 2 }} />
        </div>
      ) : (
        <section className={styles.statGrid} aria-label={t('teacher.learning.statsAria')}>
          <LearningStatCard
            icon={<TeamOutlined />}
            label={t('teacher.learning.studentCount')}
            value={String(students.length)}
            hint={t('teacher.learning.studentHint')}
          />
          <LearningStatCard
            icon={<CheckCircleOutlined />}
            label={t('teacher.learning.avgProgress')}
            value={`${avgPercent}%`}
            hint={t('teacher.learning.avgProgressHint')}
            meterPercent={avgPercent}
          />
          <LearningStatCard
            icon={<FileDoneOutlined />}
            label={t('teacher.learning.quizSubmissions')}
            value={String(submittedAttempts)}
            hint={
              quizzes.length > 0
                ? t('teacher.learning.quizSubmissionsHint', { count: quizzes.length })
                : t('teacher.learning.noQuizzes')
            }
          />
        </section>
      )}

      <section className={styles.rosterSection} aria-labelledby="roster-heading">
        <div className={styles.rosterHeader}>
          <div>
            <p className={styles.eyebrow}>{t('teacher.learning.rosterEyebrow')}</p>
            <h2 id="roster-heading">{t('teacher.learning.rosterTitle')}</h2>
            <p className={styles.rosterSubtitle}>{t('teacher.learning.rosterSubtitle')}</p>
          </div>
          <div className={styles.rosterMeta}>
            <span className={styles.metaPill}>
              {t('teacher.learning.studentCountPill', { count: students.length })}
            </span>
            <span className={styles.metaPill}>
              {t('teacher.learning.quizCountPill', { count: quizzes.length })}
            </span>
          </div>
        </div>

        {quizzes.length === 0 && students.length > 0 ? (
          <p className={styles.noQuizHint}>{t('teacher.learning.noQuizHint')}</p>
        ) : null}

        {loading ? (
          <Skeleton active paragraph={{ rows: 8 }} />
        ) : students.length === 0 ? (
          <div className={styles.emptyRoster}>
            <div className={styles.emptyIcon} aria-hidden>
              <TeamOutlined />
            </div>
            <p className={styles.emptyTitle}>
              {selectedCourse?.published ? t('teacher.learning.noStudents') : t('teacher.learning.waitPublish')}
            </p>
            <p className={styles.emptyText}>
              {selectedCourse?.published
                ? t('teacher.learning.noStudentsPublished')
                : t('teacher.learning.noStudentsDraft')}
            </p>
            {!selectedCourse?.published ? (
              <button type="button" className={styles.linkButton} onClick={() => navigate('/teacher')}>
                {t('teacher.learning.goCourseHub')}
              </button>
            ) : null}
          </div>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.dataTable}>
              <thead>
                <tr>
                  <th scope="col">{t('teacher.learning.colStudent')}</th>
                  <th scope="col">{t('teacher.learning.colProgress')}</th>
                  {quizzes.map((quiz) => (
                    <th key={quiz.quizId} scope="col" className={styles.quizHead}>
                      <QuizColumnHeader quiz={quiz} />
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {students.map((row) => (
                  <tr key={row.userId}>
                    <td>
                      <div className={styles.studentCell}>
                        <span className={styles.avatar} aria-hidden>
                          {studentInitial(row)}
                        </span>
                        <span>
                          <span className={styles.studentName}>{studentDisplayName(row)}</span>
                          <span className={styles.studentUsername}>@{row.username}</span>
                        </span>
                      </div>
                    </td>
                    <td>
                      <CourseProgressMeter percent={row.percent} />
                    </td>
                    {quizzes.map((quiz) => {
                      const cell = getQuizStat(row, quiz.quizId, quiz.questionCount)
                      return (
                        <td key={quiz.quizId}>
                          <QuizStatusTag status={cell.status} label={cell.label} />
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {!loading ? (
        <section className={styles.analyticsGrid} aria-label={t('teacher.learning.analyticsAria')}>
          <ProgressDistributionPanel
            notStarted={progressDistribution.notStarted}
            inProgress={progressDistribution.inProgress}
            completed={progressDistribution.completed}
            total={progressDistribution.total}
          />
          <QuizCompletionPanel
            slots={quizAnalytics.slots}
            submitted={quizAnalytics.submitted}
            inProgress={quizAnalytics.inProgress}
            notStarted={quizAnalytics.notStarted}
            completionRate={quizAnalytics.completionRate}
          />
        </section>
      ) : null}

      {courses !== null && courses.length === 0 ? (
        <Alert
          type="info"
          showIcon
          className={styles.alert}
          message={t('teacher.learning.noCourses')}
          description={
            <>
              {t('teacher.learning.noCoursesDescBefore')}{' '}
              <button type="button" className={styles.linkButton} onClick={() => navigate('/teacher')}>
                {t('teacher.learning.noCoursesDescLink')}
              </button>{' '}
              {t('teacher.learning.noCoursesDescAfter')}
            </>
          }
        />
      ) : null}
    </div>
  )
}
