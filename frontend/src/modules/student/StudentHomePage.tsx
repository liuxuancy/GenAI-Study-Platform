// 学生首页：Course list 与 Answer list 切换。New course 确认后进入详情页，不入库，专业从当前用户只读展示。
import { useEffect, useRef, useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'

import { studentHomePath } from '@/core/auth/homePath'
import { useAuthStore } from '@/core/auth/store'
import type { RoleNavItem } from '@/core/layout/RoleHomeShell'
import { RoleHomeShell } from '@/core/layout/RoleHomeShell'

import {
  createStudentCourse,
  createStudentQuestions,
  getStudentCourses,
  getStudentQuestions,
  type StudentCourse,
  type StudentQuestion,
} from '@/modules/student/api'
import styles from '@/modules/student/StudentHomePage.module.css'

type StudentSection = 'courses' | 'answers'

const sectionLabel: Record<StudentSection, string> = {
  courses: 'Course list',
  answers: 'Answer list',
}

const courseTypeLabel: Record<string, string> = {
  structured: 'Structured',
}

const structuredCourseType = 'Structured'

function visibilityText(visibility: string) {
  if (visibility === 'class') {
    return '同班可见'
  }
  if (visibility === 'assigned') {
    return '教师课程'
  }
  return '仅自己'
}

export function StudentHomePage() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const logout = useAuthStore((state) => state.logout)
  const [section, setSection] = useState<StudentSection>('courses')
  const [courses, setCourses] = useState<StudentCourse[] | null>(null)
  const [questions, setQuestions] = useState<StudentQuestion[] | null>(null)
  const [courseErrorMessage, setCourseErrorMessage] = useState('')
  const [questionErrorMessage, setQuestionErrorMessage] = useState('')

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_STUDENT') {
      return
    }
    let cancelled = false
    getStudentCourses()
      .then((items) => {
        if (!cancelled) {
          setCourses(items)
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setCourseErrorMessage(error.message)
        }
      })
    getStudentQuestions()
      .then((items) => {
        if (!cancelled) {
          setQuestions(items)
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setQuestionErrorMessage(error.message)
        }
      })
    return () => {
      cancelled = true
    }
  }, [currentUser?.role])

  function reloadQuestions() {
    getStudentQuestions()
      .then(setQuestions)
      .catch((error: Error) => {
        setQuestionErrorMessage(error.message)
      })
  }

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_STUDENT') {
    return <Navigate to="/" replace />
  }

  const navItems: RoleNavItem[] = [
    {
      label: 'Course list',
      active: section === 'courses',
      onSelect: () => setSection('courses'),
    },
    {
      label: 'Answer list',
      active: section === 'answers',
      onSelect: () => setSection('answers'),
    },
  ]

  return (
    <RoleHomeShell
      roleLabel="Student"
      homePath={studentHomePath}
      navTitle="Learn"
      navAriaLabel="Student workspace"
      navItems={navItems}
      pageLabel={sectionLabel[section]}
      onSignOut={() => {
        void logout()
      }}
    >
      {section === 'courses' ? (
        <CourseSection courses={courses} errorMessage={courseErrorMessage} />
      ) : (
        <AnswerList
          courses={courses}
          questions={questions}
          errorMessage={questionErrorMessage}
          onCreated={reloadQuestions}
        />
      )}
    </RoleHomeShell>
  )
}

function CourseSection({
  courses,
  errorMessage,
}: {
  courses: StudentCourse[] | null
  errorMessage: string
}) {
  const navigate = useNavigate()
  const major = useAuthStore((state) => state.currentUser?.major ?? '')
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [courseName, setCourseName] = useState('')
  const [description, setDescription] = useState('')
  const [courseType, setCourseType] = useState(structuredCourseType)
  const [nameError, setNameError] = useState('')
  const [createError, setCreateError] = useState('')
  const [creating, setCreating] = useState(false)

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
    setCourseName('')
    setDescription('')
    setCourseType(structuredCourseType)
    setNameError('')
    setCreateError('')
    setDialogOpen(true)
  }

  async function createCourse() {
    const title = courseName.trim()
    if (!title) {
      setNameError('Enter a course name.')
      return
    }
    setNameError('')
    setCreating(true)
    setCreateError('')
    try {
      const created = await createStudentCourse({ title, description: description.trim() })
      setDialogOpen(false)
      navigate(`/student/courses/${created.id}`)
    } catch (error) {
      setCreateError(error instanceof Error ? error.message : 'Could not create the course.')
    } finally {
      setCreating(false)
    }
  }

  return (
    <>
      <button className={styles.newCourse} type="button" onClick={openDialog}>
        New course
      </button>
      <dialog
        ref={dialogRef}
        className={styles.dialog}
        aria-labelledby="new-course-title"
        onClose={() => setDialogOpen(false)}
      >
        <form
          className={styles.dialogBody}
          onSubmit={(event) => {
            event.preventDefault()
            createCourse()
          }}
        >
          <h2 id="new-course-title" className={styles.dialogTitle}>
            New course
          </h2>
          <label className={styles.field}>
            Course name
            <input value={courseName} onChange={(event) => setCourseName(event.target.value)} />
          </label>
          {nameError ? <p className={styles.error}>{nameError}</p> : null}
          <label className={styles.field}>
            Description
            <input value={description} onChange={(event) => setDescription(event.target.value)} />
          </label>
          {createError ? <p className={styles.error}>{createError}</p> : null}
          <label className={styles.field}>
            Major
            <input value={major} disabled />
          </label>
          <label className={styles.field}>
            Course type
            <select value={courseType} onChange={(event) => setCourseType(event.target.value)}>
              <option value={structuredCourseType}>{structuredCourseType}</option>
            </select>
          </label>
          <div className={styles.dialogActions}>
            <button className={styles.cancel} type="button" onClick={() => setDialogOpen(false)}>
              Cancel
            </button>
            <button className={styles.create} type="submit" disabled={creating}>
              {creating ? 'Creating…' : 'Create'}
            </button>
          </div>
        </form>
      </dialog>
      <CourseList courses={courses} errorMessage={errorMessage} />
    </>
  )
}

function CourseList({ courses, errorMessage }: { courses: StudentCourse[] | null; errorMessage: string }) {
  if (errorMessage) {
    return <p className={styles.error}>{errorMessage}</p>
  }
  if (!courses) {
    return null
  }
  if (courses.length === 0) {
    return <p className={styles.empty}>No courses yet.</p>
  }

  return (
    <ul className={styles.list}>
      {courses.map((course) => (
        <li key={course.id} className={styles.row}>
          <span className={styles.titles}>
            <Link className={styles.title} to={`/student/courses/${course.id}`}>
              {course.title}
            </Link>
            <span className={styles.course}>
              {visibilityText(course.visibility)}
              {courseTypeLabel[course.courseType] ? ` · ${courseTypeLabel[course.courseType]}` : ''}
            </span>
          </span>
          <ProgressBar percent={course.percent} />
        </li>
      ))}
    </ul>
  )
}

const questionCountChoices = [2, 5, 10]

const difficultyChoices = [
  { value: 'easy', label: 'Easy' },
  { value: 'medium', label: 'Medium' },
  { value: 'hard', label: 'Hard' },
]

function AnswerList({
  courses,
  questions,
  errorMessage,
  onCreated,
}: {
  courses: StudentCourse[] | null
  questions: StudentQuestion[] | null
  errorMessage: string
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
      setFormError('Choose a course.')
      setPreview(null)
      return
    }
    setFormError('')
    setPreview(buildQuestionPreview(course.title, difficultyLabel(difficulty), questionCount))
  }

  function confirmPreview() {
    const course = (courses ?? []).find((item) => String(item.id) === courseId)
    if (!course || !preview || saving) {
      setFormError('Choose a course.')
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
      <button className={styles.newCourse} type="button" onClick={openDialog}>
        New questions
      </button>
      <dialog
        ref={dialogRef}
        className={styles.dialog}
        onClose={() => setDialogOpen(false)}
      >
        <form
          className={styles.dialogBody}
          onSubmit={(event) => {
            event.preventDefault()
            if (preview) {
              confirmPreview()
              return
            }
            generatePreview()
          }}
        >
          <h2 className={styles.dialogTitle}>New questions</h2>
          <label className={styles.field}>
            Question count
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
          <label className={styles.field}>
            Difficulty
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
          <label className={styles.field}>
            Course
            <select
              value={courseId}
              onChange={(event) => {
                setCourseId(event.target.value)
                setPreview(null)
              }}
            >
              <option value="">Choose a course</option>
              {(courses ?? []).map((course) => (
                <option key={course.id} value={course.id}>
                  {course.title}
                </option>
              ))}
            </select>
          </label>
          {formError ? <p className={styles.error}>{formError}</p> : null}
          {preview ? (
            <ol className={styles.preview}>
              {preview.map((item) => (
                <li key={item.content}>
                  <p>{item.content}</p>
                  <p>{item.options.join(' / ')}</p>
                </li>
              ))}
            </ol>
          ) : null}
          <div className={styles.dialogActions}>
            <button className={styles.cancel} type="button" onClick={() => setDialogOpen(false)}>
              Cancel
            </button>
            {preview ? (
              <button className={styles.create} type="submit" disabled={saving}>
                Confirm
              </button>
            ) : (
              <button className={styles.create} type="submit">
                Generate
              </button>
            )}
          </div>
        </form>
      </dialog>
      {renderQuestionList(questions, errorMessage)}
    </>
  )
}

function renderQuestionList(questions: StudentQuestion[] | null, errorMessage: string) {
  if (errorMessage) {
    return <p className={styles.error}>{errorMessage}</p>
  }
  if (!questions) {
    return null
  }
  if (questions.length === 0) {
    return <p className={styles.empty}>No questions yet.</p>
  }

  return (
    <ul className={styles.list}>
      {questions.map((question) => {
        const label = questionListLabel(question.title, question.difficulty)
        return (
          <li key={question.id} className={styles.row}>
            <span className={styles.titles}>
              <Link className={styles.title} to={`/student/questions/${question.id}`}>
                {label.title}
              </Link>
              <span className={styles.questionMeta}>
                <span className={styles.course}>{question.courseTitle}</span>
                {label.difficulty ? <span className={styles.course}>{label.difficulty}</span> : null}
              </span>
            </span>
            <ProgressBar
              percent={answerRatio(question.answered, question.questionCount)}
              label={`${question.answered}/${question.questionCount}`}
            />
          </li>
        )
      })}
    </ul>
  )
}

interface QuestionPreviewItem {
  content: string
  options: string[]
  correctAnswer: string
}

const generatedTitle = /^(.*) · (easy|medium|hard) · \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/

function questionListLabel(title: string, difficulty: string) {
  const match = generatedTitle.exec(title)
  const visibleTitle = match ? match[1] : title
  if (difficulty) {
    return { title: visibleTitle, difficulty: difficultyLabel(difficulty) }
  }
  if (!match) {
    return { title, difficulty: '' }
  }
  return { title: match[1], difficulty: difficultyLabel(match[2]) }
}

function answerRatio(answered: number, questionCount: number) {
  if (questionCount <= 0 || answered <= 0) {
    return 0
  }
  const percent = Math.floor((answered * 100) / questionCount)
  if (percent > 100) {
    return 100
  }
  return percent
}

function difficultyLabel(value: string) {
  for (const item of difficultyChoices) {
    if (item.value === value) {
      return item.label
    }
  }
  return value
}

function buildQuestionPreview(courseTitle: string, difficulty: string, count: number) {
  const questions: QuestionPreviewItem[] = []
  for (let index = 1; index <= count; index += 1) {
    const first = `Choice A for question ${index}`
    questions.push({
      content: `${courseTitle}: ${difficulty} question ${index}`,
      options: [first, `Choice B for question ${index}`],
      correctAnswer: first,
    })
  }
  return questions
}

function ProgressBar({ percent, label }: { percent: number; label?: string }) {
  return (
    <>
      <span className={styles.track} aria-hidden="true">
        <span className={styles.fill} style={{ width: `${percent}%` }} />
      </span>
      <span className={styles.percent}>{label ?? `${percent}%`}</span>
    </>
  )
}
