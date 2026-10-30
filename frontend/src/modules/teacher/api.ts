// 教师课程：自己的课、所在班级、发布和课内进度。
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

import { http } from '@/core/http'
import type { GenerateCourseRequest } from '@/core/api/ai'

export function generateTeacherCourse(courseId: number, input: GenerateCourseRequest = {}) {
  return http.post<{ course: TeacherCourse; meta: { templateId: string } }>(
    `/teacher/courses/${courseId}/generate`,
    input,
  )
}

export function getTeacherCourseProgress(courseId: number) {
  return http.get<{ students: ClassStudentProgress[] }>(`/teacher/courses/${courseId}/progress`)
}
