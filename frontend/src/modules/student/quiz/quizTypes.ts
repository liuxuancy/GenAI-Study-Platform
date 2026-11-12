/**
 * 前端测验契约（适配层）。
 *
 * 后端实体映射（未改库）：
 * - Quiz ≈ course_question（一套题，多道 course_question_content）
 * - Question ≈ course_question_content 单行
 * - QuizAttempt ≈ student_question_progress + student_question_record（每学生每套题一条提交态）
 * - Chapter / Section ≈ course.content_json 中的 chapter / activity；section 级测验字段尚未入库
 */

export type QuizCreationMethod = 'manual' | 'excel' | 'ai'

export type QuizTargetType = 'chapter' | 'section' | 'unlinked'

export type QuizDifficulty = 'easy' | 'medium' | 'hard'

export type QuizAttemptStatus = 'pending' | 'in_progress' | 'completed'

export type QuizViewMode = 'grouped' | 'flat'

export type QuizStatusFilter = 'all' | QuizAttemptStatus

export type QuizCreationFilter = 'all' | QuizCreationMethod

/** 学生可见的一套测验（由 course_question 列表项映射） */
export interface StudentQuizView {
  id: string
  courseId: string
  courseName: string
  title: string
  targetType: QuizTargetType
  chapterKey: string | null
  chapterName: string | null
  chapterMissing: boolean
  sectionKey: string | null
  sectionName: string | null
  creationMethod: QuizCreationMethod | null
  difficulty: QuizDifficulty | null
  questionCount: number
  creatorId: string | null
  creatorName: string | null
  creatorRole: string
  /** 课程可见性来源（与创建方式无关） */
  courseSourceLabel: string | null
  attempt: QuizAttemptView
}

/** 当前登录学生对该套题的答题态（无独立 attempt 表时的兼容视图） */
export interface QuizAttemptView {
  quizId: string
  completedCount: number
  totalCount: number
  status: QuizAttemptStatus
  score: number | null
}

export interface SectionQuizGroup {
  sectionKey: string | null
  sectionTitle: string
  quizzes: StudentQuizView[]
}

export interface ChapterQuizGroup {
  chapterKey: string
  chapterTitle: string
  sections: SectionQuizGroup[]
  /** 挂在章节上的测验（当前后端 chapter_key 均在此） */
  chapterQuizzes: StudentQuizView[]
}

export interface CourseQuizGroup {
  courseId: string
  courseTitle: string
  courseSourceLabel: string
  creatorLabel: string
  quizTotal: number
  completedCount: number
  chapters: ChapterQuizGroup[]
}

export const UNLINKED_CHAPTER_KEY = '__unlinked__'
export const UNLINKED_CHAPTER_TITLE = '未关联教学节点'
