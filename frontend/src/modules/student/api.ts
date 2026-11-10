// 学生课程和套题的列表与详情。
import { http } from '@/core/http'
import type { AiGenerateProvider, AiGenerationResult, GenerateCourseRequest } from '@/core/api/ai'
import type { BubbleNode } from '@/modules/student/bubbleTree'
import type { CourseAvatar } from '@/modules/student/PixelSprite'

export interface StudentCourse {
  id: number
  title: string
  courseType: string
  description: string
  percent: number
  visibility: string
  owned: boolean
  chapterCount: number
  affection: number
  avatar: CourseAvatar | null
}

export interface StudentQuestion {
  id: number
  title: string
  courseTitle: string
  difficulty: string
  answered: number
  questionCount: number
  percent: number
  courseId: number
  chapterKey?: string
  chapterName?: string
  creatorId: number
  creatorName: string
  creatorRole: string
  isSubmit: boolean
  score?: number | null
  createdAt?: string | null
  creationMethod?: string | null
  publishStatus?: string | null
  sectionKey?: string | null
  sectionName?: string | null
  attemptId?: number | null
}

export function getStudentCourses() {
  return http.get<StudentCourse[]>('/student/courses')
}

export function getStudentQuestions() {
  return http.get<StudentQuestion[]>('/student/questions')
}

export interface StudentCourseDetail {
  id: number
  title: string
  description: string
  audience: string
  language: string
  visibility: string
  shared: boolean
  classmateCount: number
  owned: boolean
  percent: number
  affection: number
  avatar: CourseAvatar | null
  tree: BubbleNode | null
}

export function createStudentCourse(input: { title: string; description: string; provider: AiGenerateProvider }) {
  return http.post<StudentCourse>('/student/courses', input).then((result) => {
    window.dispatchEvent(new Event('student-achievements-changed'))
    return result
  })
}

export function setStudentCourseSharing(courseId: number, shared: boolean) {
  return http.put<{ shared: boolean; classmateCount: number }>(`/student/courses/${courseId}/sharing`, { shared })
}

export function generateStudentCourse(courseId: number, input: GenerateCourseRequest = {}) {
  return http.post<AiGenerationResult<StudentCourseDetail>>(
    `/student/courses/${courseId}/generate`,
    input,
  )
}

export interface StudentQuestionContent {
  id: number
  content: string
  options: string[]
  userOption: string
  isCorrect?: boolean
  correctAnswer: string
  explanation: string
}

export interface StudentQuestionDetail {
  id: number
  title: string
  courseTitle: string
  chapterName?: string
  difficulty: string
  questionCount: number
  percent: number
  isSubmit: boolean
  score?: number
  questions: StudentQuestionContent[]
}

export function getStudentCourse(courseId: string) {
  return http.get<StudentCourseDetail>(`/student/courses/${courseId}`)
}

export function updateStudentCourseProgress(courseId: string, percent: number) {
  return http.put<{ percent: number }>(`/student/courses/${courseId}/progress`, { percent }).then((result) => {
    window.dispatchEvent(new Event('student-achievements-changed'))
    return result
  })
}

export interface ChapterQAMessage {
  role: 'user' | 'assistant'
  content: string
}

export function askStudentChapter(
  courseId: string,
  chapterKey: string,
  input: {
    question: string
    provider?: string
    activityKey?: string
    selection?: string
    conversationId?: number
  },
) {
  return http.post<{ answer: string; conversationId: number }>(
    `/student/courses/${courseId}/chapters/${encodeURIComponent(chapterKey)}/ask`,
    input,
  )
}

export interface StudentConversation {
  id: number
  title: string
  courseId: number
  courseTitle: string
  courseAccessible: boolean
  updatedAt: string
  messageCount: number
}

export interface StudentConversationMessage {
  id: number
  role: 'user' | 'assistant'
  content: string
  chapterKey: string
  activityKey: string
  sectionTitle: string
  createdAt: string
}

export function listStudentConversations(courseId?: string) {
  if (courseId) {
    return http.get<StudentConversation[]>(`/student/courses/${courseId}/conversations`)
  }
  return http.get<StudentConversation[]>('/student/conversations')
}

export function getStudentConversation(courseId: string, conversationId: number) {
  return http.get<{ id: number; title: string; updatedAt: string; messages: StudentConversationMessage[] }>(
    `/student/courses/${courseId}/conversations/${conversationId}`,
  )
}

export function deleteStudentConversation(courseId: string, conversationId: number) {
  return http.delete<{ deleted: boolean }>(`/student/courses/${courseId}/conversations/${conversationId}`)
}

export function getStudentQuestion(questionId: string) {
  return http.get<StudentQuestionDetail>(`/student/questions/${questionId}`)
}

export function answerStudentQuestion(questionId: string, contentId: number, option: string, submit = false) {
  const body: { option: string; submit?: boolean } = { option }
  if (submit) {
    body.submit = true
  }
  return http.put<StudentQuestionDetail>(`/student/questions/${questionId}/contents/${contentId}`, body).then((result) => {
    window.dispatchEvent(new Event('student-achievements-changed'))
    return result
  })
}

export interface CreatedStudentQuestion {
  id: number
  title: string
  courseTitle: string
  percent: number
}

export interface NewQuestionItem {
  content: string
  options: string[]
  correctAnswer: string
}

export function createStudentQuestions(input: {
  courseId: number
  difficulty: string
  questions: NewQuestionItem[]
}) {
  return http.post<CreatedStudentQuestion>('/student/questions', input)
}

export type LearningTaskType = '学习' | '测验' | '复习'

export interface LearningDashboardTask {
  id: string
  title: string
  course: string
  type: LearningTaskType
  completed: boolean
}

export interface LearningRecommendedAction {
  title: string
  detail: string
}

export interface LearningDashboardData {
  tasks: LearningDashboardTask[]
  weeklyMinutes: number[]
  activeDays: number[]
  year: number
  month: number
  ruleSuggestion: string
  recommendedAction: LearningRecommendedAction
}

export function getLearningDashboard(year?: number, month?: number) {
  const params = new URLSearchParams()
  if (year) {
    params.set('year', String(year))
  }
  if (month) {
    params.set('month', String(month))
  }
  const query = params.toString()
  return http.get<LearningDashboardData>(`/student/learning/dashboard${query ? `?${query}` : ''}`)
}

export interface LearningInsightResult {
  suggestion: string
  recommendedAction: LearningRecommendedAction
}

export function postLearningInsight(input: { provider?: string; language?: string } = {}) {
  return http.post<LearningInsightResult>('/student/learning/insights', input)
}

export function postCourseLearningHeartbeat(courseId: string, seconds: number) {
  return http.post<{ recorded: boolean }>(`/student/courses/${courseId}/learning/heartbeat`, {
    seconds,
  })
}
