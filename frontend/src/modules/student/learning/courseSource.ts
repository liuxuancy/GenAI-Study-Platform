import type { StudentCourse } from '@/modules/student/api'

export type CourseSource = 'teacher' | 'personal' | 'shared'

export type CourseSourceFilter = 'all' | CourseSource

export function courseSourceOf(course: StudentCourse): CourseSource {
  if (course.owned) {
    return 'personal'
  }
  if (course.visibility === 'assigned') {
    return 'teacher'
  }
  return 'shared'
}

export const courseSourceLabels: Record<CourseSource, string> = {
  teacher: '教师发布',
  personal: '我创建的',
  shared: '同学共享',
}

export const courseSourceDescriptions: Record<CourseSource, string> = {
  teacher: '由教师创建并发布的课程',
  personal: '由你创建和管理的课程',
  shared: '来自同学分享的学习资源',
}

export function visibilityText(visibility: string) {
  if (visibility === 'class') {
    return '同班可见'
  }
  if (visibility === 'assigned') {
    return '教师课程'
  }
  return '仅自己'
}

export type ProgressFilter = 'all' | 'not_started' | 'in_progress' | 'completed'

export function matchesProgressFilter(course: StudentCourse, filter: ProgressFilter) {
  if (filter === 'all') {
    return true
  }
  if (filter === 'not_started') {
    return course.percent === 0
  }
  if (filter === 'in_progress') {
    return course.percent > 0 && course.percent < 100
  }
  return course.percent === 100
}

export function formatStatCount(value: number) {
  return String(value).padStart(2, '0')
}

export function pickContinueCourse(courses: StudentCourse[]) {
  if (courses.length === 0) {
    return null
  }
  return (
    courses.find((course) => course.percent > 0 && course.percent < 100) ??
    courses.find((course) => course.percent < 100) ??
    courses[0]
  )
}

export function pickRecentCourses(courses: StudentCourse[], limit = 3) {
  return [...courses]
    .sort((left, right) => {
      const leftActive = left.percent > 0 && left.percent < 100 ? 1 : 0
      const rightActive = right.percent > 0 && right.percent < 100 ? 1 : 0
      if (leftActive !== rightActive) {
        return rightActive - leftActive
      }
      return right.percent - left.percent
    })
    .slice(0, limit)
}
