import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'
import type {
  TeacherQuizOverviewItem,
  TeacherStudentLearningRow,
} from '@/modules/teacher/api'

export type QuizDisplayStatus = 'idle' | 'progress' | 'submitted'

export interface QuizCellDisplay {
  status: QuizDisplayStatus
  answered: number
  total: number
  submitted: boolean
  score: number | null
  label: string
}

function t(key: string, params?: Record<string, string | number>) {
  return translateApp(getAppLocale(), key, params)
}

export function getQuizStat(
  row: TeacherStudentLearningRow,
  quizId: number,
  questionCount: number,
): QuizCellDisplay {
  const stat = row.quizzes.find((item) => item.quizId === quizId)
  const total = stat?.total ?? questionCount
  const answered = stat?.answered ?? 0
  const submitted = stat?.submitted ?? false
  const score = stat?.score ?? null

  if (answered === 0 && !submitted) {
    return { status: 'idle', answered, total, submitted, score, label: t('teacher.learning.quizStat.idle') }
  }
  if (submitted) {
    const parts = [t('teacher.learning.quizStat.submitted', { answered, total })]
    if (score != null) {
      parts.push(t('teacher.learning.quizStat.score', { score }))
    }
    return { status: 'submitted', answered, total, submitted, score, label: parts.join(' · ') }
  }
  return {
    status: 'progress',
    answered,
    total,
    submitted,
    score,
    label: t('teacher.learning.quizStat.inProgress', { answered, total }),
  }
}

export function computeProgressDistribution(students: TeacherStudentLearningRow[]) {
  let notStarted = 0
  let inProgress = 0
  let completed = 0
  for (const student of students) {
    if (student.percent <= 0) {
      notStarted += 1
    } else if (student.percent >= 100) {
      completed += 1
    } else {
      inProgress += 1
    }
  }
  return { notStarted, inProgress, completed, total: students.length }
}

export function computeQuizAnalytics(
  students: TeacherStudentLearningRow[],
  quizzes: TeacherQuizOverviewItem[],
) {
  const slots = students.length * quizzes.length
  let submitted = 0
  let inProgress = 0
  let notStarted = 0

  for (const student of students) {
    for (const quiz of quizzes) {
      const cell = getQuizStat(student, quiz.quizId, quiz.questionCount)
      if (cell.status === 'submitted') {
        submitted += 1
      } else if (cell.status === 'progress') {
        inProgress += 1
      } else {
        notStarted += 1
      }
    }
  }

  const completionRate = slots === 0 ? 0 : Math.round((submitted / slots) * 100)

  return { slots, submitted, inProgress, notStarted, completionRate }
}

export function averagePercent(students: TeacherStudentLearningRow[]) {
  if (students.length === 0) {
    return 0
  }
  return Math.round(students.reduce((sum, row) => sum + row.percent, 0) / students.length)
}

export function countSubmittedAttempts(students: TeacherStudentLearningRow[]) {
  return students.reduce(
    (sum, row) => sum + row.quizzes.filter((item) => item.submitted).length,
    0,
  )
}

export function studentDisplayName(row: TeacherStudentLearningRow) {
  return row.nickname?.trim() || row.username
}

export function studentInitial(row: TeacherStudentLearningRow) {
  const name = studentDisplayName(row)
  return name.slice(0, 1).toUpperCase()
}

export type CourseProgressPhase = 'idle' | 'active' | 'done'

export function courseProgressPhase(percent: number): CourseProgressPhase {
  if (percent <= 0) {
    return 'idle'
  }
  if (percent >= 100) {
    return 'done'
  }
  return 'active'
}
