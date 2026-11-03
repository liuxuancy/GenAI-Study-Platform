import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { ReadOutlined } from '@ant-design/icons'

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
const difficultyChoices = [
  { value: 'easy', label: 'Easy' },
  { value: 'medium', label: 'Medium' },
  { value: 'hard', label: 'Hard' },
]

export function QuizDashboard() {
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
        <p className={styles.eyebrow}>YOUR ASSESSMENTS</p>
        <h1>我的测验</h1>
        <p>按课程、章节和小节查找测验，跟踪你的答题进度。</p>
      </header>

      <QuizStatistics stats={stats} />

      <div className={styles.toolbar}>
        <select
          value={viewMode}
          aria-label="展示模式"
          onChange={(event) => setViewMode(event.target.value as QuizViewMode)}
        >
          <option value="grouped">按课程分组</option>
          <option value="flat">全部测验列表</option>
        </select>
        <select
          value={courseFilter}
          aria-label="课程筛选"
          onChange={(event) => {
            setCourseFilter(event.target.value)
            setChapterFilter('all')
          }}
        >
          <option value="all">全部课程</option>
          {courseOptions.map((course) => (
            <option key={course.value} value={course.value}>
              {course.label}
            </option>
          ))}
        </select>
        <select
          value={chapterFilter}
          aria-label="章节筛选"
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
          aria-label="创建方式筛选"
          onChange={(event) => setCreationFilter(event.target.value as QuizCreationFilter)}
        >
          <option value="all">全部创建方式</option>
          <option value="manual">手动创建</option>
          <option value="excel">Excel 导入</option>
          <option value="ai">AI 生成</option>
        </select>
        <CreateQuizDialog courses={courses} onCreated={reload} />
      </div>

      <div className={styles.filterTabs} role="tablist" aria-label="测验状态">
        {(
          [
            ['all', '全部测验'],
            ['pending', '待完成'],
            ['in_progress', '进行中'],
            ['completed', '已完成'],
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
          <strong>测验加载失败</strong>
          <p>{errorMessage}</p>
        </div>
      ) : !questions || !courses ? (
        <div className={styles.skeletonBlock} aria-label="正在加载测验" />
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
  return (
    <section className={styles.stats} aria-label="测验概览">
      <article className={styles.statCard}>
        <strong>{stats.total}</strong>
        <span>全部测验</span>
      </article>
      <article className={styles.statCard}>
        <strong>{stats.pending}</strong>
        <span>待完成</span>
      </article>
      <article className={styles.statCard}>
        <strong>{stats.inProgress}</strong>
        <span>进行中</span>
      </article>
      <article className={styles.statCard}>
        <strong>{stats.completed}</strong>
        <span>已完成</span>
      </article>
    </section>
  )
}

function CourseQuizList({ groups }: { groups: ReturnType<typeof buildCourseQuizGroups> }) {
  if (groups.length === 0) {
    return (
      <div className={styles.empty}>
        <strong>暂无测验</strong>
        <p>当前筛选下没有测验，请调整条件或创建新测验。</p>
      </div>
    )
  }

  return (
    <div className={styles.courseList}>
      {groups.map((group) => (
        <section key={group.courseId} className={styles.courseGroup}>
          <CourseHeader group={group} />
          {group.quizTotal === 0 ? (
            <p className={styles.courseEmpty}>这门课程暂时没有测验</p>
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
  const coverKey = resolveCourseCoverKey(group.courseTitle)
  return (
    <header className={styles.courseHeader}>
      <div className={`${styles.courseIcon} ${styles[`cover_${coverKey}`]}`} aria-hidden>
        <ReadOutlined />
      </div>
      <div className={styles.courseHeaderText}>
        <h2>{group.courseTitle}</h2>
        <p>
          {group.courseSourceLabel} · {group.creatorLabel} · 共 {group.quizTotal} 套测验 · 已完成{' '}
          {group.completedCount}
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
        <span className={styles.chapterMeta}>{quizCount} 套测验</span>
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
  if (views.length === 0) {
    return (
      <div className={styles.empty}>
        <strong>暂无测验</strong>
        <p>当前筛选下没有测验。</p>
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
          {nodePathLabel(quiz)} · {quiz.questionCount} 题 · 难度 {difficultyLabel(quiz.difficulty)} · 进度{' '}
          {quiz.attempt.completedCount}/{quiz.attempt.totalCount}
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
          {quiz.attempt.status === 'completed' ? '查看' : '开始'}
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
      setFormError('请选择课程。')
      setPreview(null)
      return
    }
    setFormError('')
    setPreview(buildQuestionPreview(course.title, difficulty, questionCount))
  }

  function confirmPreview() {
    const course = (courses ?? []).find((item) => String(item.id) === courseId)
    if (!course || !preview || saving) {
      setFormError('请选择课程。')
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
        创建测验
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
          <h2 className={homeStyles.dialogTitle}>创建测验</h2>
          <label className={homeStyles.field}>
            题目数量
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
            难度
            <select
              value={difficulty}
              onChange={(event) => {
                setDifficulty(event.target.value)
                setPreview(null)
              }}
            >
              {difficultyChoices.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </select>
          </label>
          <label className={homeStyles.field}>
            课程
            <select
              value={courseId}
              onChange={(event) => {
                setCourseId(event.target.value)
                setPreview(null)
              }}
            >
              <option value="">选择课程</option>
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
              取消
            </button>
            {preview ? (
              <button className={homeStyles.create} type="submit" disabled={saving}>
                确认创建
              </button>
            ) : (
              <button className={homeStyles.create} type="submit">
                生成预览
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
