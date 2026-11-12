import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { DownOutlined, ReadOutlined } from '@ant-design/icons'
import { Select } from 'antd'

import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import { ApiError } from '@/core/http'
import { useI18n } from '@/core/i18n/useI18n'
import {
  generateStudentQuiz,
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
  const [treesLoadedFor, setTreesLoadedFor] = useState<StudentCourse[] | null>(null)
  const [errorMessage, setErrorMessage] = useState('')
  const [viewMode, setViewMode] = useState<QuizViewMode>('grouped')
  const [statusTab, setStatusTab] = useState<QuizStatusFilter>('all')
  const [courseFilter, setCourseFilter] = useState('all')
  const [chapterFilter, setChapterFilter] = useState('all')
  const [creationFilter, setCreationFilter] = useState<QuizCreationFilter>('all')
  const [scrollQuizId, setScrollQuizId] = useState<string | null>(null)

  function reload(created?: { id: number }) {
    Promise.all([getStudentCourses(), getStudentQuestions()])
      .then(([courseItems, questionItems]) => {
        setCourses(courseItems)
        setQuestions(questionItems)
        if (created) {
          if (!Number.isSafeInteger(created.id) || created.id <= 0 ||
              !questionItems.some((question) => question.id === created.id)) {
            setErrorMessage(t('student.quiz.createdNotFound'))
            return
          }
          setStatusTab('all')
          setCourseFilter('all')
          setChapterFilter('all')
          setCreationFilter('all')
          setScrollQuizId(String(created.id))
        }
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
    if (!courses) {
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
        setTreesLoadedFor(courses)
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
          onChange={(value) => { setScrollQuizId(null); setViewMode(value) }}
          options={[
            { value: 'grouped', label: t('student.quiz.viewGrouped') },
            { value: 'flat', label: t('student.quiz.viewFlat') },
          ]}
        />
        <Select<string>
          value={courseFilter}
          aria-label={t('student.quiz.courseFilterAria')}
          onChange={(value) => {
            setScrollQuizId(null)
            setCourseFilter(value)
            setChapterFilter('all')
          }}
          options={[{ value: 'all', label: t('student.quiz.allCourses') }, ...courseOptions]}
        />
        <Select<string>
          value={chapterFilter}
          aria-label={t('student.quiz.chapterFilterAria')}
          onChange={(value) => { setScrollQuizId(null); setChapterFilter(value) }}
          options={chapterOptions}
        />
        <Select<QuizCreationFilter>
          value={creationFilter}
          aria-label={t('student.quiz.creationFilterAria')}
          onChange={(value) => { setScrollQuizId(null); setCreationFilter(value) }}
          options={[
            { value: 'all', label: t('student.quiz.allCreation') },
            { value: 'manual', label: t('student.quiz.manual') },
            { value: 'excel', label: t('student.quiz.excel') },
            { value: 'ai', label: t('student.quiz.ai') },
          ]}
        />
        <CreateQuizDialog courses={courses} onCreated={(id) => reload({ id })} />
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
            onClick={() => { setScrollQuizId(null); setStatusTab(value) }}
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
        <FlatQuizList views={filteredViews} scrollQuizId={scrollQuizId} onScrolled={() => setScrollQuizId(null)} />
      ) : (
        <CourseQuizList groups={courseGroups} scrollQuizId={treesLoadedFor === courses ? scrollQuizId : null} onScrolled={() => setScrollQuizId(null)} />
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

interface QuizScrollProps {
  scrollQuizId: string | null
  onScrolled: () => void
}

function CourseQuizList({ groups, scrollQuizId, onScrolled }: { groups: ReturnType<typeof buildCourseQuizGroups> } & QuizScrollProps) {
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
      {groups.map((group) => <CourseQuizGroup key={group.courseId} group={group} scrollQuizId={scrollQuizId} onScrolled={onScrolled} />)}
    </div>
  )
}

function CourseQuizGroup({ group, scrollQuizId, onScrolled }: { group: ReturnType<typeof buildCourseQuizGroups>[number] } & QuizScrollProps) {
  const { t } = useI18n()
  const [open, setOpen] = useState(true)
  const contentId = useId()
  const containsTarget = scrollQuizId !== null && group.chapters.some((chapter) =>
    chapter.chapterQuizzes.some((quiz) => quiz.id === scrollQuizId) ||
    chapter.sections.some((section) => section.quizzes.some((quiz) => quiz.id === scrollQuizId)),
  )
  useEffect(() => {
    if (containsTarget) setOpen(true)
  }, [containsTarget])
  return (
    <section className={styles.courseGroup}>
      <CourseHeader group={group} open={open} contentId={contentId} onToggle={() => setOpen((value) => !value)} />
      <div id={contentId} hidden={!open}>
        {group.quizTotal === 0 ? (
          <p className={styles.courseEmpty}>{t('student.quiz.courseEmpty')}</p>
        ) : (
          group.chapters.map((chapter) => (
            <ChapterPanel key={`${group.courseId}-${chapter.chapterKey}`} chapter={chapter} visible={open} scrollQuizId={scrollQuizId} onScrolled={onScrolled} />
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
  visible,
  scrollQuizId,
  onScrolled,
}: {
  chapter: ReturnType<typeof buildCourseQuizGroups>[number]['chapters'][number]
  visible: boolean
} & QuizScrollProps) {
  const { t } = useI18n()
  const [open, setOpen] = useState(true)
  const containsTarget = scrollQuizId !== null && (chapter.chapterQuizzes.some((quiz) => quiz.id === scrollQuizId) ||
    chapter.sections.some((section) => section.quizzes.some((quiz) => quiz.id === scrollQuizId)))
  useEffect(() => {
    if (containsTarget) setOpen(true)
  }, [containsTarget])
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
                  <QuizItemRow key={quiz.id} quiz={quiz} visible={visible} scrollQuizId={scrollQuizId} onScrolled={onScrolled} />
                ))}
              </ul>
            </div>
          ))}
          {chapter.chapterQuizzes.length > 0 ? (
            <ul className={styles.quizRows}>
              {chapter.chapterQuizzes.map((quiz) => (
                <QuizItemRow key={quiz.id} quiz={quiz} visible={visible} scrollQuizId={scrollQuizId} onScrolled={onScrolled} />
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

function FlatQuizList({ views, scrollQuizId, onScrolled }: { views: StudentQuizView[] } & QuizScrollProps) {
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
        <QuizItemRow key={quiz.id} quiz={quiz} showCourse visible scrollQuizId={scrollQuizId} onScrolled={onScrolled} />
      ))}
    </ul>
  )
}

function QuizItemRow({ quiz, showCourse, visible, scrollQuizId, onScrolled }: {
  quiz: StudentQuizView
  showCourse?: boolean
  visible: boolean
} & QuizScrollProps) {
  const { t } = useI18n()
  const rowRef = useRef<HTMLLIElement>(null)
  useEffect(() => {
    if (visible && scrollQuizId === quiz.id && rowRef.current) {
      const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
      rowRef.current.scrollIntoView({ behavior: reducedMotion ? 'instant' : 'smooth', block: 'center' })
      onScrolled()
    }
  }, [visible, scrollQuizId, quiz.id, onScrolled])
  const progress =
    quiz.questionCount > 0
      ? Math.min(100, Math.floor((quiz.attempt.completedCount * 100) / quiz.questionCount))
      : 0

  return (
    <li ref={rowRef} className={styles.quizRow}>
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
  onCreated: (id: number) => void
}) {
  const { t } = useI18n()
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [courseId, setCourseId] = useState('')
  const [questionCount, setQuestionCount] = useState(2)
  const [difficulty, setDifficulty] = useState('easy')
  const [chapterKey, setChapterKey] = useState('')
  const [courseTree, setCourseTree] = useState<BubbleNode | null>(null)
  const [courseLoading, setCourseLoading] = useState(false)
  const [courseError, setCourseError] = useState('')
  const [courseLoadAttempt, setCourseLoadAttempt] = useState(0)
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)
  const savingRef = useRef(false)
  const selection = useAiProviderSelect()

  useEffect(() => {
    if (!dialogOpen || !courseId) {
      setCourseTree(null)
      setCourseError('')
      setCourseLoading(false)
      return
    }
    let cancelled = false
    setCourseTree(null)
    setCourseError('')
    setCourseLoading(true)
    getStudentCourse(courseId)
      .then((detail) => {
        if (!cancelled) setCourseTree(detail.tree)
      })
      .catch((error: Error) => {
        if (!cancelled) setCourseError(error.message)
      })
      .finally(() => {
        if (!cancelled) setCourseLoading(false)
      })
    return () => { cancelled = true }
  }, [courseId, dialogOpen, courseLoadAttempt])

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
    setChapterKey('')
    setFormError('')
    setDialogOpen(true)
  }

  function generateQuiz() {
    const course = (courses ?? []).find((item) => String(item.id) === courseId)
    if (!course) {
      setFormError(t('student.quiz.create.pickCourseError'))
      return
    }
    if (savingRef.current || courseLoading || courseError || !courseTree || selection.loading || !selection.optionsAvailable) {
      return
    }
    savingRef.current = true
    setSaving(true)
    setFormError('')
    generateStudentQuiz(course.id, {
      chapterKey,
      count: questionCount,
      difficulty,
      provider: selection.provider,
    })
      .then((created) => {
        dialogRef.current?.close()
        setDialogOpen(false)
        onCreated(created.id)
      })
      .catch((error: Error) => {
        setFormError(error instanceof ApiError && (error.code === 0 || error.message.startsWith('Unexpected server response'))
          ? t('student.quiz.create.outcomeUnknown')
          : error.message)
      })
      .finally(() => {
        savingRef.current = false
        setSaving(false)
      })
  }

  return (
    <>
      <button className={styles.createButton} type="button" onClick={openDialog}>
        {t('student.quiz.create.button')}
      </button>
      <dialog ref={dialogRef} className={homeStyles.dialog} onCancel={(event) => {
        if (savingRef.current) event.preventDefault()
      }} onClose={() => setDialogOpen(false)}>
        <form
          className={homeStyles.dialogBody}
          onSubmit={(event) => {
            event.preventDefault()
            generateQuiz()
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
              disabled={saving}
              onChange={setQuestionCount}
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
              disabled={saving}
              onChange={setDifficulty}
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
              disabled={saving}
              onChange={(value) => {
                setCourseId(value)
                setChapterKey('')
                setFormError('')
              }}
              options={[
                { value: '', label: t('student.quiz.create.selectCourse') },
                ...(courses ?? []).map((course) => ({ value: String(course.id), label: course.title })),
              ]}
            />
          </div>
          <div className={homeStyles.field}>
            <label htmlFor="quiz-create-chapter">{t('student.quiz.create.chapterLabel')}</label>
            <Select<string>
              id="quiz-create-chapter"
              className={homeStyles.dialogSelect}
              getPopupContainer={(trigger) => trigger.parentElement!}
              value={chapterKey}
              disabled={saving || courseLoading || !courseTree}
              onChange={setChapterKey}
              options={[
                { value: '', label: t('student.quiz.create.allChapters') },
                ...(courseTree?.children ?? []).filter((chapter) => chapter.key).map((chapter) => ({ value: chapter.key!, label: chapter.label })),
              ]}
            />
          </div>
          <div className={homeStyles.field}>
            <label htmlFor="quiz-create-provider">{t('student.quiz.create.providerLabel')}</label>
            <Select
              id="quiz-create-provider"
              className={homeStyles.dialogSelect}
              getPopupContainer={(trigger) => trigger.parentElement!}
              value={selection.provider}
              options={selection.options}
              onChange={selection.setProvider}
              loading={selection.loading}
              disabled={saving || !selection.optionsAvailable}
            />
          </div>
          {courseId && !courseLoading && !courseError && !courseTree ? <p className={homeStyles.error}>{t('student.quiz.create.contentMissing')}</p> : null}
          {courseError ? (
            <p className={homeStyles.error}>
              {t('student.quiz.create.courseLoadFailed')} {courseError}{' '}
              <button type="button" onClick={() => setCourseLoadAttempt((value) => value + 1)}>{t('student.quiz.create.retry')}</button>
            </p>
          ) : null}
          {selection.error ? <p className={homeStyles.error}>{selection.error}</p> : null}
          {!selection.loading && !selection.error && !selection.optionsAvailable ? <p className={homeStyles.error}>{t('student.quiz.create.providerMissing')}</p> : null}
          {formError ? <p className={homeStyles.error}>{formError}</p> : null}
          <div className={homeStyles.dialogActions}>
            <button className={`${homeStyles.cancel} ${styles.cancelButton}`} type="button" disabled={saving} onClick={() => setDialogOpen(false)}>
              {t('student.quiz.create.cancel')}
            </button>
            <button className={homeStyles.create} type="submit" disabled={saving || courseLoading || Boolean(courseError) || !courseTree || selection.loading || !selection.optionsAvailable}>
              {t(saving ? 'student.quiz.create.generating' : 'student.quiz.create.generate')}
            </button>
          </div>
        </form>
      </dialog>
    </>
  )
}
