import { http, uploadForm } from '@/core/http'
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'
import type { CourseChapter, CourseContent } from '@/modules/teacher/api'

export interface CourseResourceItem {
  id: number
  courseId: number
  displayName: string
  originalName: string
  fileExtension: string
  mimeType: string
  fileSize: number
  scopeType: 'course' | 'chapter' | 'section'
  chapterKey: string
  sectionKey: string
  publishStatus: 'draft' | 'published'
  previewStatus: string
  createdAt: string
  updatedAt: string
}

export interface CourseResourceListResult {
  items: CourseResourceItem[]
  total: number
}

export interface ResourcePreviewResult {
  kind: string
  url?: string
  message?: string
}

export interface ResourceDownloadResult {
  url: string
  fileName: string
  expiresInSec: number
}

export function listTeacherCourseResources(
  courseId: number,
  query: { publish?: string; type?: string; chapterKey?: string; search?: string },
) {
  const params = new URLSearchParams()
  if (query.publish) params.set('publish', query.publish)
  if (query.type) params.set('type', query.type)
  if (query.chapterKey) params.set('chapterKey', query.chapterKey)
  if (query.search) params.set('search', query.search)
  const q = params.toString()
  return http.get<CourseResourceListResult>(`/teacher/courses/${courseId}/resources${q ? `?${q}` : ''}`)
}

export function uploadTeacherCourseResource(
  courseId: number,
  formData: FormData,
  onProgress?: (percent: number) => void,
) {
  return uploadForm<CourseResourceItem>(`/teacher/courses/${courseId}/resources`, formData, onProgress)
}

export function patchTeacherCourseResource(
  courseId: number,
  resourceId: number,
  body: {
    displayName?: string
    scopeType?: string
    chapterKey?: string
    sectionKey?: string
  },
) {
  return http.patch<CourseResourceItem>(`/teacher/courses/${courseId}/resources/${resourceId}`, body)
}

export function publishTeacherCourseResource(courseId: number, resourceId: number, published: boolean) {
  return http.patch<CourseResourceItem>(`/teacher/courses/${courseId}/resources/${resourceId}/publish`, { published })
}

export function deleteTeacherCourseResource(courseId: number, resourceId: number) {
  return http.delete<{ ok: boolean }>(`/teacher/courses/${courseId}/resources/${resourceId}`)
}

export function listStudentCourseResources(
  courseId: number,
  query: { type?: string; chapterKey?: string; search?: string },
) {
  const params = new URLSearchParams()
  if (query.type) params.set('type', query.type)
  if (query.chapterKey) params.set('chapterKey', query.chapterKey)
  if (query.search) params.set('search', query.search)
  const q = params.toString()
  return http.get<CourseResourceListResult>(`/student/courses/${courseId}/resources${q ? `?${q}` : ''}`)
}

export function getCourseResourcePreview(courseId: number, resourceId: number) {
  return http.get<ResourcePreviewResult>(`/courses/${courseId}/resources/${resourceId}/preview`)
}

export function getCourseResourceDownload(courseId: number, resourceId: number) {
  return http.get<ResourceDownloadResult>(`/courses/${courseId}/resources/${resourceId}/download`)
}

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function scopeLabel(item: CourseResourceItem, content: CourseContent | null, locale = getAppLocale()): string {
  if (item.scopeType === 'course') return translateApp(locale, 'courseResources.scopeWholeCourse')
  const chapter = content?.chapters.find((ch) => ch.key === item.chapterKey)
  if (item.scopeType === 'chapter') return chapter?.title ?? item.chapterKey
  const activity = chapter?.activities.find((a) => a.key === item.sectionKey)
  return activity?.title ?? `${item.chapterKey} / ${item.sectionKey}`
}

export function chapterOptions(content: CourseContent | null): CourseChapter[] {
  return content?.chapters ?? []
}
