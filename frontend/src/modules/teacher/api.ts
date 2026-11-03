// 教师课程：自己的课、所在班级、发布和课内进度。
import type { AiGenerationMeta, AiGenerationResult, GenerateCourseRequest } from '@/core/api/ai'
import { http } from '@/core/http'

export interface TeacherCourse {
  id: number
  title: string
  description: string
  courseType: string
  audience: string
  language: string
  published: boolean
  classIds: number[]
  chapterCount: number
  hasContent: boolean
  content: CourseContent | null
}

export type ActivityType = 'rich_text' | 'coding' | 'mock_interview' | 'quiz_game'

export interface CourseActivity {
  key: string
  type: ActivityType
  title: string
  order: number
  payload: Record<string, unknown>
}

export interface CourseChapter {
  key: string
  title: string
  summary: string
  order: number
  activities: CourseActivity[]
}

export interface CourseContent {
  schemaVersion: '1.0'
  title: string
  description: string
  audience: 'CS' | 'NON_CS' | 'GENERAL'
  language: 'zh-CN' | 'en-US'
  estimatedMinutes: number
  chapters: CourseChapter[]
}

export interface TeacherQuestion {
  id: number
  setId: number
  setTitle: string
  chapterKey?: string
  sectionKey?: string
  difficulty: 'easy' | 'medium' | 'hard'
  content: string
  options: string[]
  correctAnswer: string
  explanation: string
}

export type TeacherQuestionInput = Omit<TeacherQuestion, 'id' | 'setId' | 'setTitle'> & {
  chapterKey?: string
  sectionKey?: string
}

export interface ImportQuestionRow extends TeacherQuestionInput {
  row: number
}

export interface ImportPreview {
  validCount: number
  rows: ImportQuestionRow[]
  errors: { row: number; field: string; message: string }[]
}

export interface TeacherClass {
  id: number
  name: string
}

export interface ClassStudentProgress {
  userId: number
  username: string
  nickname: string
  percent: number
}

export function getTeacherCourses() {
  return http.get<TeacherCourse[]>('/teacher/courses')
}

export function getTeacherClasses() {
  return http.get<TeacherClass[]>('/teacher/classes')
}

export function createTeacherCourse(input: { title: string; description: string }) {
  return http.post<TeacherCourse>('/teacher/courses', input)
}

export function publishTeacherCourse(courseId: number, classIds: number[]) {
  return http.put<TeacherCourse>(`/teacher/courses/${courseId}/classes`, { classIds })
}

export function generateTeacherCourse(courseId: number, input: GenerateCourseRequest = {}) {
  return http.post<AiGenerationResult<TeacherCourse>>(
    `/teacher/courses/${courseId}/generate`,
    input,
  )
}

export function getTeacherCourseProgress(courseId: number) {
  return http.get<{ students: ClassStudentProgress[] }>(`/teacher/courses/${courseId}/progress`)
}

export function getTeacherCourse(courseId: number) {
  return http.get<TeacherCourse>(`/teacher/courses/${courseId}`)
}

export function saveTeacherCourseContent(courseId: number, content: CourseContent) {
  return http.put<TeacherCourse>(`/teacher/courses/${courseId}/content`, content)
}

export function getTeacherQuestions(courseId: number) {
  return http.get<TeacherQuestion[]>(`/teacher/courses/${courseId}/questions`)
}

export function createTeacherQuestion(courseId: number, input: TeacherQuestionInput) {
  return http.post<TeacherQuestion>(`/teacher/courses/${courseId}/questions`, input)
}

export function updateTeacherQuestion(contentId: number, input: TeacherQuestionInput) {
  return http.put<TeacherQuestion>(`/teacher/questions/${contentId}`, input)
}

export function deleteTeacherQuestion(contentId: number) {
  return http.delete<{ deleted: boolean }>(`/teacher/questions/${contentId}`)
}

export function previewQuestionImport(courseId: number, rows: ImportQuestionRow[]) {
  return http.post<ImportPreview>(`/teacher/courses/${courseId}/questions/import/preview`, { rows })
}

export function confirmQuestionImport(
  courseId: number,
  rows: ImportQuestionRow[],
  options: { chapterKey?: string; sectionKey?: string; creationMethod?: 'manual' | 'excel' | 'ai' } = {},
) {
  return http.post<TeacherQuestion[]>(`/teacher/courses/${courseId}/questions/import/confirm`, {
    rows,
    chapterKey: options.chapterKey,
    sectionKey: options.sectionKey,
    creationMethod: options.creationMethod,
  })
}

export function generateTeacherQuestions(
  courseId: number,
  input: {
    provider?: string
    chapterKey?: string
    count: number
    difficulty: string
    extraRequirements?: string
  },
) {
  return http.post<{ questions: TeacherQuestionInput[]; meta: { templateId: string } }>(
    `/teacher/courses/${courseId}/questions/generate`,
    input,
  )
}

export interface FragmentGenerationInput {
  provider?: string
  topic?: string
  extraRequirements?: string
  targetChapterKey: string
  targetActivityKey?: string
}

export function generateTeacherChapter(courseId: number, input: FragmentGenerationInput) {
  return http.post<AiGenerationResult<TeacherCourse>>(`/teacher/courses/${courseId}/chapters/generate`, input)
}

export function generateTeacherActivity(courseId: number, input: FragmentGenerationInput & { targetActivityKey: string }) {
  return http.post<AiGenerationResult<TeacherCourse>>(`/teacher/courses/${courseId}/activities/generate`, input)
}

export interface QualityDimension {
  name: 'coverage' | 'clarity' | 'practice' | 'audience_fit'
  score: number
  comment: string
}

export interface CourseQualityReport {
  report: {
    overallScore: number
    summaryMarkdown: string
    dimensions: QualityDimension[]
    suggestions: string[]
  }
  meta?: AiGenerationMeta
  createdAt: string
}

export function getCourseQualityReport(courseId: number) {
  return http.get<CourseQualityReport>(`/teacher/courses/${courseId}/quality-report`)
}

export function generateCourseQualityReport(courseId: number, input: { provider?: string; rubric?: string }) {
  return http.post<CourseQualityReport>(`/teacher/courses/${courseId}/quality-report/generate`, input)
}

export function getCourseMermaid(courseId: number) {
  return http.get<{ source: string }>(`/teacher/courses/${courseId}/mermaid`)
}
