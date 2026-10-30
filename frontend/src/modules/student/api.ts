// 学生课程和套题的列表与详情。
import { http } from '@/core/http'

export interface StudentCourse {
  id: number
  title: string
  courseType: string
  percent: number
}

export interface StudentQuestion {
  id: number
  title: string
  courseTitle: string
  difficulty: string
  answered: number
  questionCount: number
  percent: number
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
}

export interface StudentQuestionContent {
  id: number
  content: string
  options: string[]
  userOption: string
  correctAnswer: string
  explanation: string
}

export interface StudentQuestionDetail {
  id: number
  title: string
  courseTitle: string
  difficulty: string
  questionCount: number
  percent: number
  questions: StudentQuestionContent[]
}

export function getStudentCourse(courseId: string) {
  return http.get<StudentCourseDetail>(`/student/courses/${courseId}`)
}

export function updateStudentCourseProgress(courseId: string, percent: number) {
  return http.put<{ percent: number }>(`/student/courses/${courseId}/progress`, { percent })
}

export function getStudentQuestion(questionId: string) {
  return http.get<StudentQuestionDetail>(`/student/questions/${questionId}`)
}

export function answerStudentQuestion(questionId: string, contentId: number, option: string, submit = false) {
  const body: { option: string; submit?: boolean } = { option }
  if (submit) {
    body.submit = true
  }
  return http.put<StudentQuestionDetail>(`/student/questions/${questionId}/contents/${contentId}`, body)
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
