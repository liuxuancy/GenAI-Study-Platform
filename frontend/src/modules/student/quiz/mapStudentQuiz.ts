import type { StudentCourse, StudentQuestion } from '@/modules/student/api'
import {
  courseSourceLabels,
  courseSourceOf,
} from '@/modules/student/learning/courseSource'
import type {
  QuizAttemptView,
  QuizCreationMethod,
  QuizDifficulty,
  StudentQuizView,
} from '@/modules/student/quiz/quizTypes'

const studentGeneratedTitle =
  /^(.*) · (easy|medium|hard) · \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/
const chapterQuizTitle = /^Quiz · (.+) · \d+ · \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/
const excelImportTitle = /^Excel 导入 · \d+ · \d+$/
const teacherBankTitle = /^教师题库 · \d+ · \d+$/

function parseDifficulty(value: string): QuizDifficulty | null {
  if (value === 'easy' || value === 'medium' || value === 'hard') {
    return value
  }
  return null
}

function displayTitle(title: string, chapterName?: string | null) {
  const studentMatch = studentGeneratedTitle.exec(title)
  if (studentMatch) {
    return studentMatch[1]
  }
  if (excelImportTitle.test(title)) {
    return 'Excel 导入测验'
  }
  if (teacherBankTitle.test(title)) {
    return '手动测验'
  }
  const chapterMatch = chapterQuizTitle.exec(title)
  if (chapterMatch) {
    const chapter = chapterName?.trim()
    return chapter ? `${chapter}测验` : chapterMatch[1]
  }
  return title
}

function inferCreationMethod(item: StudentQuestion): QuizCreationMethod | null {
  const fromServer = item.creationMethod
  if (fromServer === 'manual' || fromServer === 'excel' || fromServer === 'ai') {
    return fromServer
  }
  if (chapterQuizTitle.test(item.title)) {
    return 'ai'
  }
  if (item.title.startsWith('教师题库')) {
    return 'manual'
  }
  if (studentGeneratedTitle.test(item.title)) {
    return 'manual'
  }
  if (item.creatorRole === 'ROLE_TEACHER' && item.chapterKey?.trim()) {
    return 'ai'
  }
  if (item.creatorRole === 'ROLE_TEACHER') {
    return 'manual'
  }
  if (item.creatorRole === 'ROLE_STUDENT') {
    return 'manual'
  }
  return null
}

function mapAttempt(item: StudentQuestion): QuizAttemptView {
  let status: QuizAttemptView['status'] = 'pending'
  if (item.isSubmit) {
    status = 'completed'
  } else if (item.answered > 0) {
    status = 'in_progress'
  }
  return {
    quizId: String(item.id),
    completedCount: item.answered,
    totalCount: item.questionCount,
    status,
    score: item.isSubmit ? (item.score ?? null) : null,
  }
}

function resolveCourse(
  item: StudentQuestion,
  courseById: Map<number, StudentCourse>,
  courseByTitle: Map<string, StudentCourse>,
) {
  if (typeof item.courseId === 'number' && item.courseId > 0) {
    const byId = courseById.get(item.courseId)
    if (byId) {
      return byId
    }
  }
  return courseByTitle.get(item.courseTitle)
}

export function mapStudentQuestionToView(
  item: StudentQuestion,
  courseById: Map<number, StudentCourse>,
  courseByTitle: Map<string, StudentCourse> = new Map(),
): StudentQuizView {
  const chapterKey = item.chapterKey?.trim() || null
  const sectionKey = item.sectionKey?.trim() || null
  const course = resolveCourse(item, courseById, courseByTitle)
  const courseSource = course ? courseSourceLabels[courseSourceOf(course)] : null
  const courseId = course
    ? String(course.id)
    : typeof item.courseId === 'number' && item.courseId > 0
      ? String(item.courseId)
      : `title:${item.courseTitle}`

  return {
    id: String(item.id),
    courseId,
    courseName: course?.title || item.courseTitle,
    title: displayTitle(item.title, item.chapterName),
    targetType: sectionKey ? 'section' : chapterKey ? 'chapter' : 'unlinked',
    chapterKey,
    chapterName: item.chapterName?.trim() || null,
    sectionKey,
    sectionName: item.sectionName?.trim() || null,
    creationMethod: inferCreationMethod(item),
    difficulty: parseDifficulty(item.difficulty),
    questionCount: item.questionCount,
    creatorId: item.creatorId ? String(item.creatorId) : null,
    creatorName: item.creatorName?.trim() || null,
    creatorRole: item.creatorRole,
    courseSourceLabel: courseSource,
    attempt: mapAttempt(item),
  }
}

export function mapStudentQuestionsToViews(
  items: StudentQuestion[],
  courses: StudentCourse[],
): StudentQuizView[] {
  const courseById = new Map(courses.map((course) => [course.id, course]))
  const courseByTitle = new Map(courses.map((course) => [course.title, course]))
  return items.map((item) => mapStudentQuestionToView(item, courseById, courseByTitle))
}

export function quizStatistics(views: StudentQuizView[]) {
  const unique = views
  return {
    total: unique.length,
    pending: unique.filter((view) => view.attempt.status === 'pending').length,
    inProgress: unique.filter((view) => view.attempt.status === 'in_progress').length,
    completed: unique.filter((view) => view.attempt.status === 'completed').length,
  }
}

export function nodePathLabel(view: StudentQuizView) {
  if (!view.chapterKey) {
    return '未关联教学节点'
  }
  const chapter = view.chapterName || view.chapterKey
  if (view.sectionName) {
    return `${chapter} / ${view.sectionName}`
  }
  return chapter
}

export const creationMethodLabels: Record<QuizCreationMethod, string> = {
  manual: '手动创建',
  excel: 'Excel 导入',
  ai: 'AI 生成',
}

export const attemptStatusLabels: Record<QuizAttemptView['status'], string> = {
  pending: '待完成',
  in_progress: '进行中',
  completed: '已完成',
}

export function difficultyLabel(difficulty: QuizDifficulty | null) {
  if (difficulty === 'easy') {
    return '简单'
  }
  if (difficulty === 'medium') {
    return '中等'
  }
  if (difficulty === 'hard') {
    return '困难'
  }
  return '—'
}

/** @deprecated 使用 StudentQuizView */
export type { StudentQuizView as Quiz }
