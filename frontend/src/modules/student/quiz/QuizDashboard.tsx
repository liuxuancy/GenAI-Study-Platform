import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { DownOutlined, ReadOutlined } from '@ant-design/icons'
import { Select } from 'antd'

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
        <Select<QuizViewMode>
          value={viewMode}
          aria-label={t('student.quiz.viewModeAria')}
          onChange={setViewMode}
          options={[
            { value: 'grouped', label: t('student.quiz.viewGrouped') },
            { value: 'flat', label: t('student.quiz.viewFlat') },
          ]}
        />
        <Select<string>
          value={courseFilter}
          aria-label={t('student.quiz.courseFilterAria')}
          onChange={(value) => {
            setCourseFilter(value)
            setChapterFilter('all')
          }}
          options={[{ value: 'all', label: t('student.quiz.allCourses') }, ...courseOptions]}
        />
        <Select<string>
          value={chapterFilter}
          aria-label={t('student.quiz.chapterFilterAria')}
          onChange={setChapterFilter}
          options={chapterOptions}
        />
        <Select<QuizCreationFilter>
          value={creationFilter}
          aria-label={t('student.quiz.creationFilterAria')}
          onChange={setCreationFilter}
          options={[
            { value: 'all', label: t('student.quiz.allCreation') },
            { value: 'manual', label: t('student.quiz.manual') },
            { value: 'excel', label: t('student.quiz.excel') },
            { value: 'ai', label: t('student.quiz.ai') },
          ]}
        />
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
      {groups.map((group) => <CourseQuizGroup key={group.courseId} group={group} />)}
    </div>
  )
}

function CourseQuizGroup({ group }: { group: ReturnType<typeof buildCourseQuizGroups>[number] }) {
  const { t } = useI18n()
  const [open, setOpen] = useState(true)
  const contentId = useId()
  return (
    <section className={styles.courseGroup}>
      <CourseHeader group={group} open={open} contentId={contentId} onToggle={() => setOpen((value) => !value)} />
      <div id={contentId} hidden={!open}>
        {group.quizTotal === 0 ? (
          <p className={styles.courseEmpty}>{t('student.quiz.courseEmpty')}</p>
        ) : (
          group.chapters.map((chapter) => (
            <ChapterPanel key={`${group.courseId}-${chapter.chapterKey}`} chapter={chapter} />
          ))
        )}
      </div>
    </section>
  )
}

function CourseHeader({ group, open, contentId, onToggle }: {
  group: ReturnType<typeof buildCourseQuizGroups>[number]
  open: boolean
  contentId: string
  onToggle: () => void
}) {
  const { t } = useI18n()
  const courseIdNum = Number(group.courseId)
  const coverKey = resolveCourseCoverKey(
    group.courseTitle,
    '',
    Number.isFinite(courseIdNum) ? courseIdNum : undefined,
  )
  return (
    <h2 className={styles.courseHeading}>
      <button type="button" className={styles.courseHeader} aria-expanded={open} aria-controls={contentId} onClick={onToggle}>
        <span className={`${styles.courseIcon} ${styles[`cover_${coverKey}`]}`} aria-hidden>
          <ReadOutlined />
        </span>
        <span className={styles.courseHeaderText}>
          <span className={styles.courseTitle}>{group.courseTitle}</span>
          <span className={styles.courseMeta}>
            {t('student.quiz.courseHeaderMeta', {
              source: group.courseSourceLabel,
              creator: group.creatorLabel,
              total: group.quizTotal,
              completed: group.completedCount,
            })}
          </span>
        </span>
        <DownOutlined className={`${styles.courseChevron} ${open ? styles.courseChevronOpen : ''}`} aria-hidden />
      </button>
    </h2>
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
          <div className={homeStyles.field}>
            <label htmlFor="quiz-create-count">{t('student.quiz.create.countLabel')}</label>
            <Select<number>
              id="quiz-create-count"
              className={homeStyles.dialogSelect}
              getPopupContainer={(trigger) => trigger.parentElement!}
              value={questionCount}
              onChange={(value) => {
                setQuestionCount(value)
                setPreview(null)
              }}
              options={questionCountChoices.map((count) => ({ value: count, label: String(count) }))}
            />
          </div>
          <div className={homeStyles.field}>
            <label htmlFor="quiz-create-difficulty">{t('student.quiz.create.difficultyLabel')}</label>
            <Select<string>
              id="quiz-create-difficulty"
              className={homeStyles.dialogSelect}
              getPopupContainer={(trigger) => trigger.parentElement!}
              value={difficulty}
              onChange={(value) => {
                setDifficulty(value)
                setPreview(null)
              }}
              options={(['easy', 'medium', 'hard'] as const).map((level) => ({ value: level, label: difficultyLabel(level) }))}
            />
          </div>
          <div className={homeStyles.field}>
            <label htmlFor="quiz-create-course">{t('student.quiz.create.courseLabel')}</label>
            <Select<string>
              id="quiz-create-course"
              className={homeStyles.dialogSelect}
              getPopupContainer={(trigger) => trigger.parentElement!}
              value={courseId}
              onChange={(value) => {
                setCourseId(value)
                setPreview(null)
              }}
              options={[
                { value: '', label: t('student.quiz.create.selectCourse') },
                ...(courses ?? []).map((course) => ({ value: String(course.id), label: course.title })),
              ]}
            />
          </div>
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
