import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { ReadOutlined } from '@ant-design/icons'

import { useI18n } from '@/core/i18n/useI18n'
import {
  createStudentQuestions,
  getStudentCourse,
  getStudentCourses,
  getStudentQuestions,
  type StudentCourse,
  type StudentQuestion,
} from '@/modules/student/api'
import type { BubbleNode } from '@/modules/student/bubbleTree'
import homeStyles from '@/modules/student/StudentHomePage.module.css'
import { resolveCourseCoverKey } from '@/modules/student/studentCourseVisuals'
import {
  buildCourseQuizGroups,
  chapterFilterOptions,
  filterQuizViews,
  allViewsFromQuestions,
} from '@/modules/student/quiz/buildQuizHierarchy'
import {
  attemptStatusLabels,
  creationMethodLabels,
  difficultyLabel,
  nodePathLabel,
  quizStatistics,
} from '@/modules/student/quiz/mapStudentQuiz'
import type {
  QuizCreationFilter,
  QuizStatusFilter,
  QuizViewMode,
  StudentQuizView,
} from '@/modules/student/quiz/quizTypes'
import styles from '@/modules/student/quiz/quizDashboard.module.css'

const questionCountChoices = [2, 5, 10]

export function QuizDashboard() {
  const { t } = useI18n()
  const [courses, setCourses] = useState<StudentCourse[] | null>(null)
  const [questions, setQuestions] = useState<StudentQuestion[] | null>(null)
  const [courseTrees, setCourseTrees] = useState<Map<number, BubbleNode | null>>(new Map())
  const [errorMessage, setErrorMessage] = useState('')
  const [viewMode, setViewMode] = useState<QuizViewMode>('grouped')
  const [statusTab, setStatusTab] = useState<QuizStatusFilter>('all')
  const [courseFilter, setCourseFilter] = useState('all')
  const [chapterFilter, setChapterFilter] = useState('all')
  const [creationFilter, setCreationFilter] = useState<QuizCreationFilter>('all')

  function reload() {
    Promise.all([getStudentCourses(), getStudentQuestions()])
      .then(([courseItems, questionItems]) => {
        setCourses(courseItems)
        setQuestions(questionItems)
        setErrorMessage('')
      })
      .catch((error: Error) => {
        setErrorMessage(error.message)
      })
  }

  useEffect(() => {
    reload()
  }, [])

  useEffect(() => {
    if (!courses?.length) {
      return
    }
    let cancelled = false
    Promise.all(
      courses.map((course) =>
        getStudentCourse(String(course.id))
          .then((detail) => [course.id, detail.tree] as const)
          .catch(() => [course.id, null] as const),
      ),
    ).then((pairs) => {
      if (!cancelled) {
        setCourseTrees(new Map(pairs))
      }
    })
    return () => {
      cancelled = true
    }
  }, [courses])

  const allViews = useMemo(
    () => (courses && questions ? allViewsFromQuestions(courses, questions) : []),
    [courses, questions],
  )

  const filteredViews = useMemo(
    () =>
      filterQuizViews(allViews, {
        status: statusTab,
        courseId: courseFilter,
        chapterKey: chapterFilter,
        creationMethod: creationFilter,
      }),
    [allViews, chapterFilter, courseFilter, creationFilter, statusTab],
  )

  const stats = useMemo(() => quizStatistics(filteredViews), [filteredViews])

  const courseGroups = useMemo(() => {
    if (!courses) {
      return []
    }
    return buildCourseQuizGroups(courses, filteredViews, courseTrees, {
      courseId: courseFilter,
    })
  }, [courseFilter, courseTrees, courses, filteredViews, viewMode])

  const chapterOptions = useMemo(
    () => chapterFilterOptions(allViews, courseFilter),
    [allViews, courseFilter],
  )

  const courseOptions = useMemo(() => {
    const options = (courses ?? []).map((course) => ({
      value: String(course.id),
      label: course.title,
    }))
    const known = new Set(options.map((option) => option.value))
    for (const view of allViews) {
      if (known.has(view.courseId)) {
        continue
      }
      known.add(view.courseId)
      options.push({ value: view.courseId, label: view.courseName })
    }
    return options
  }, [allViews, courses])

  useEffect(() => {
    if (chapterFilter === 'all') {
      return
    }
    if (!chapterOptions.some((option) => option.value === chapterFilter)) {
      setChapterFilter('all')
    }
  }, [chapterFilter, chapterOptions])

  return (
    <div>
      <header className={styles.pageHeading}>
        <p className={styles.eyebrow}>{t('student.quiz.eyebrow')}</p>
        <h1>{t('student.quiz.title')}</h1>
        <p>{t('student.quiz.subtitle')}</p>
      </header>

      <QuizStatistics stats={stats} />

      <div className={styles.toolbar}>
        <select
          value={viewMode}
          aria-label={t('student.quiz.viewModeAria')}
          onChange={(event) => setViewMode(event.target.value as QuizViewMode)}
        >
          <option value="grouped">{t('student.quiz.viewGrouped')}</option>
          <option value="flat">{t('student.quiz.viewFlat')}</option>
        </select>
        <select
          value={courseFilter}
          aria-label={t('student.quiz.courseFilterAria')}
          onChange={(event) => {
            setCourseFilter(event.target.value)
            setChapterFilter('all')
          }}
        >
          <option value="all">{t('student.quiz.allCourses')}</option>
          {courseOptions.map((course) => (
            <option key={course.value} value={course.value}>
              {course.label}
            </option>
          ))}
        </select>
        <select
          value={chapterFilter}
          aria-label={t('student.quiz.chapterFilterAria')}
          onChange={(event) => setChapterFilter(event.target.value)}
        >
          {chapterOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <select
          value={creationFilter}
          aria-label={t('student.quiz.creationFilterAria')}
          onChange={(event) => setCreationFilter(event.target.value as QuizCreationFilter)}
        >
          <option value="all">{t('student.quiz.allCreation')}</option>
          <option value="manual">{t('student.quiz.manual')}</option>
          <option value="excel">{t('student.quiz.excel')}</option>
          <option value="ai">{t('student.quiz.ai')}</option>
        </select>
        <CreateQuizDialog courses={courses} onCreated={reload} />
      </div>

      <div className={styles.filterTabs} role="tablist" aria-label={t('student.quiz.statusFilterAria')}>
        {(
          [
            ['all', t('student.quiz.statusTab.all')],
            ['pending', t('student.quiz.statusTab.pending')],
            ['in_progress', t('student.quiz.statusTab.in_progress')],
            ['completed', t('student.quiz.statusTab.completed')],
          ] as const
        ).map(([value, label]) => (
          <button
            key={value}
            type="button"
            role="tab"
            aria-selected={statusTab === value}
            className={`${styles.filterTab} ${statusTab === value ? styles.filterTabActive : ''}`}
            onClick={() => setStatusTab(value)}
          >
            {label}
          </button>
        ))}
      </div>

      {errorMessage ? (
        <div className={styles.empty}>
          <strong>{t('student.quiz.loadFailed')}</strong>
          <p>{errorMessage}</p>
        </div>
      ) : !questions || !courses ? (
        <div className={styles.skeletonBlock} aria-label={t('student.quiz.loadingAria')} />
      ) : viewMode === 'flat' ? (
        <FlatQuizList views={filteredViews} />
      ) : (
        <CourseQuizList groups={courseGroups} />
      )}
    </div>
  )
}

function QuizStatistics({
  stats,
}: {
  stats: { total: number; pending: number; inProgress: number; completed: number }
}) {
  const { t } = useI18n()
  return (
    <section className={styles.stats} aria-label={t('student.quiz.statsAria')}>
      <article className={styles.statCard}>
        <strong>{stats.total}</strong>
        <span>{t('student.quiz.statTotal')}</span>
      </article>
      <article className={styles.statCard}>
        <strong>{stats.pending}</strong>
        <span>{t('student.quiz.statPending')}</span>
      </article>
      <article className={styles.statCard}>
        <strong>{stats.inProgress}</strong>
        <span>{t('student.quiz.statInProgress')}</span>
      </article>
      <article className={styles.statCard}>
        <strong>{stats.completed}</strong>
        <span>{t('student.quiz.statCompleted')}</span>
      </article>
    </section>
  )
}

function CourseQuizList({ groups }: { groups: ReturnType<typeof buildCourseQuizGroups> }) {
  const { t } = useI18n()
  if (groups.length === 0) {
    return (
      <div className={styles.empty}>
        <strong>{t('student.quiz.emptyTitle')}</strong>
        <p>{t('student.quiz.emptyBody')}</p>
      </div>
    )
  }

  return (
    <div className={styles.courseList}>
      {groups.map((group) => (
        <section key={group.courseId} className={styles.courseGroup}>
          <CourseHeader group={group} />
          {group.quizTotal === 0 ? (
            <p className={styles.courseEmpty}>{t('student.quiz.courseEmpty')}</p>
          ) : (
            group.chapters.map((chapter) => (
              <ChapterPanel key={`${group.courseId}-${chapter.chapterKey}`} chapter={chapter} />
            ))
          )}
        </section>
      ))}
    </div>
  )
}

function CourseHeader({ group }: { group: ReturnType<typeof buildCourseQuizGroups>[number] }) {
  const { t } = useI18n()
  const coverKey = resolveCourseCoverKey(group.courseTitle, '', group.courseId)
  return (
    <header className={styles.courseHeader}>
      <div className={`${styles.courseIcon} ${styles[`cover_${coverKey}`]}`} aria-hidden>
        <ReadOutlined />
      </div>
      <div className={styles.courseHeaderText}>
        <h2>{group.courseTitle}</h2>
        <p>
          {t('student.quiz.courseHeaderMeta', {
            source: group.courseSourceLabel,
            creator: group.creatorLabel,
            total: group.quizTotal,
            completed: group.completedCount,
          })}
        </p>
      </div>
    </header>
  )
}

function ChapterPanel({
  chapter,
}: {
  chapter: ReturnType<typeof buildCourseQuizGroups>[number]['chapters'][number]
}) {
  const { t } = useI18n()
  const [open, setOpen] = useState(true)
  const quizCount =
    chapter.chapterQuizzes.length + chapter.sections.reduce((sum, section) => sum + section.quizzes.length, 0)

  return (
    <div className={styles.chapterPanel}>
      <button
        type="button"
        className={styles.chapterHeader}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span className={styles.chapterTitle}>{chapter.chapterTitle}</span>
        <span className={styles.chapterMeta}>{t('student.quiz.quizSetCount', { count: quizCount })}</span>
      </button>
      {open ? (
        <div className={styles.chapterBody}>
          {chapter.sections.map((section) => (
            <div key={section.sectionKey ?? section.sectionTitle} className={styles.sectionBlock}>
              <p className={styles.sectionTitle}>{section.sectionTitle}</p>
              <ul className={styles.quizRows}>
                {section.quizzes.map((quiz) => (
                  <QuizItemRow key={quiz.id} quiz={quiz} />
                ))}
              </ul>
            </div>
          ))}
          {chapter.chapterQuizzes.length > 0 ? (
            <ul className={styles.quizRows}>
              {chapter.chapterQuizzes.map((quiz) => (
                <QuizItemRow key={quiz.id} quiz={quiz} />
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

function FlatQuizList({ views }: { views: StudentQuizView[] }) {
  const { t } = useI18n()
  if (views.length === 0) {
    return (
      <div className={styles.empty}>
        <strong>{t('student.quiz.emptyTitle')}</strong>
        <p>{t('student.quiz.emptyFlatBody')}</p>
      </div>
    )
  }
  return (
    <ul className={styles.quizRows}>
      {views.map((quiz) => (
        <QuizItemRow key={quiz.id} quiz={quiz} showCourse />
      ))}
    </ul>
  )
}

function QuizItemRow({ quiz, showCourse }: { quiz: StudentQuizView; showCourse?: boolean }) {
  const { t } = useI18n()
  const progress =
    quiz.questionCount > 0
      ? Math.min(100, Math.floor((quiz.attempt.completedCount * 100) / quiz.questionCount))
      : 0

  return (
    <li className={styles.quizRow}>
      <div className={styles.quizRowMain}>
        <div className={styles.quizRowTitleLine}>
          <Link className={styles.quizRowTitle} to={`/student/questions/${quiz.id}`}>
            {quiz.title}
          </Link>
          {quiz.creationMethod ? (
            <span className={styles.methodTag}>{creationMethodLabels[quiz.creationMethod]}</span>
          ) : null}
          <span className={styles.statusTag}>{attemptStatusLabels[quiz.attempt.status]}</span>
        </div>
        <p className={styles.quizRowSub}>
          {showCourse ? `${quiz.courseName} · ` : ''}
          {t('student.quiz.rowMeta', {
            path: nodePathLabel(quiz),
            count: quiz.questionCount,
            difficulty: difficultyLabel(quiz.difficulty),
            done: quiz.attempt.completedCount,
            total: quiz.attempt.totalCount,
          })}
        </p>
        <div className={styles.quizRowProgress} aria-hidden="true">
          <span className={styles.progressTrack}>
            <span className={styles.progressFill} style={{ width: `${progress}%` }} />
          </span>
        </div>
      </div>
      <div className={styles.quizRowAside}>
        <span className={styles.scoreCell}>
          {quiz.attempt.status === 'completed' && quiz.attempt.score != null
            ? `${quiz.attempt.score}%`
            : '—'}
        </span>
        <Link className={styles.actionLink} to={`/student/questions/${quiz.id}`}>
          {quiz.attempt.status === 'completed' ? t('student.quiz.viewAction') : t('student.quiz.startAction')}
        </Link>
      </div>
    </li>
  )
}

function CreateQuizDialog({
  courses,
  onCreated,
}: {
  courses: StudentCourse[] | null
  onCreated: () => void
}) {
  const { t } = useI18n()
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [courseId, setCourseId] = useState('')
  const [questionCount, setQuestionCount] = useState(2)
  const [difficulty, setDifficulty] = useState('easy')
  const [formError, setFormError] = useState('')
  const [preview, setPreview] = useState<QuestionPreviewItem[] | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) {
      return
    }
    if (dialogOpen && !dialog.open) {
      dialog.showModal()
    }
    if (!dialogOpen && dialog.open) {
      dialog.close()
    }
  }, [dialogOpen])

  function openDialog() {
    setCourseId('')
    setQuestionCount(2)
    setDifficulty('easy')
    setFormError('')
    setPreview(null)
    setDialogOpen(true)
  }

  function generatePreview() {
    const course = (courses ?? []).find((item) => String(item.id) === courseId)
    if (!course) {
      setFormError(t('student.quiz.create.pickCourseError'))
      setPreview(null)
      return
    }
    setFormError('')
    setPreview(buildQuestionPreview(course.title, difficulty, questionCount))
  }

  function confirmPreview() {
    const course = (courses ?? []).find((item) => String(item.id) === courseId)
    if (!course || !preview || saving) {
      setFormError(t('student.quiz.create.pickCourseError'))
      return
    }
    setSaving(true)
    setFormError('')
    createStudentQuestions({
      courseId: course.id,
      difficulty,
      questions: preview,
    })
      .then(() => {
        setDialogOpen(false)
        onCreated()
      })
      .catch((error: Error) => {
        setFormError(error.message)
      })
      .finally(() => {
        setSaving(false)
      })
  }

  return (
    <>
      <button className={styles.createButton} type="button" onClick={openDialog}>
        {t('student.quiz.create.button')}
      </button>
      <dialog ref={dialogRef} className={homeStyles.dialog} onClose={() => setDialogOpen(false)}>
        <form
          className={homeStyles.dialogBody}
          onSubmit={(event) => {
            event.preventDefault()
            if (preview) {
              confirmPreview()
              return
            }
            generatePreview()
          }}
        >
          <h2 className={homeStyles.dialogTitle}>{t('student.quiz.create.title')}</h2>
          <label className={homeStyles.field}>
            {t('student.quiz.create.countLabel')}
            <select
              value={questionCount}
              onChange={(event) => {
                setQuestionCount(Number(event.target.value))
                setPreview(null)
              }}
            >
              {questionCountChoices.map((count) => (
                <option key={count} value={count}>
                  {count}
                </option>
              ))}
            </select>
          </label>
          <label className={homeStyles.field}>
            {t('student.quiz.create.difficultyLabel')}
            <select
              value={difficulty}
              onChange={(event) => {
                setDifficulty(event.target.value)
                setPreview(null)
              }}
            >
              {(['easy', 'medium', 'hard'] as const).map((level) => (
                <option key={level} value={level}>
                  {difficultyLabel(level)}
                </option>
              ))}
            </select>
          </label>
          <label className={homeStyles.field}>
            {t('student.quiz.create.courseLabel')}
            <select
              value={courseId}
              onChange={(event) => {
                setCourseId(event.target.value)
                setPreview(null)
              }}
            >
              <option value="">{t('student.quiz.create.selectCourse')}</option>
              {(courses ?? []).map((course) => (
                <option key={course.id} value={course.id}>
                  {course.title}
                </option>
              ))}
            </select>
          </label>
          {formError ? <p className={homeStyles.error}>{formError}</p> : null}
          {preview ? (
            <ol className={homeStyles.preview}>
              {preview.map((item) => (
                <li key={item.content}>
                  <p>{item.content}</p>
                  <p>{item.options.join(' / ')}</p>
                </li>
              ))}
            </ol>
          ) : null}
          <div className={homeStyles.dialogActions}>
            <button className={homeStyles.cancel} type="button" onClick={() => setDialogOpen(false)}>
              {t('student.quiz.create.cancel')}
            </button>
            {preview ? (
              <button className={homeStyles.create} type="submit" disabled={saving}>
                {t('student.quiz.create.confirmCreate')}
              </button>
            ) : (
              <button className={homeStyles.create} type="submit">
                {t('student.quiz.create.generatePreview')}
              </button>
            )}
          </div>
        </form>
      </dialog>
    </>
  )
}

interface QuestionPreviewItem {
  content: string
  options: string[]
  correctAnswer: string
}

function buildQuestionPreview(courseTitle: string, difficultyValue: string, count: number) {
  const questions: QuestionPreviewItem[] = []
  for (let index = 1; index <= count; index += 1) {
    const first = `Choice A for question ${index}`
    questions.push({
      content: `${courseTitle}: ${difficultyValue} question ${index}`,
      options: [first, `Choice B for question ${index}`],
      correctAnswer: first,
    })
  }
  return questions
}
