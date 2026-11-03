import type { StudentCourse } from '@/modules/student/api'
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'

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

export function courseSourceLabel(source: CourseSource, locale = getAppLocale()) {
  return translateApp(locale, `student.courseSource.${source}`)
}

export function courseSourceDescription(source: CourseSource, locale = getAppLocale()) {
  return translateApp(locale, `student.courseSource.${source}Desc`)
}

/** @deprecated 在组件内请调用 courseSourceLabel */
export const courseSourceLabels: Record<CourseSource, string> = {
  get teacher() {
    return courseSourceLabel('teacher')
  },
  get personal() {
    return courseSourceLabel('personal')
  },
  get shared() {
    return courseSourceLabel('shared')
  },
}

/** @deprecated 在组件内请调用 courseSourceDescription */
export const courseSourceDescriptions: Record<CourseSource, string> = {
  get teacher() {
    return courseSourceDescription('teacher')
  },
  get personal() {
    return courseSourceDescription('personal')
  },
  get shared() {
    return courseSourceDescription('shared')
  },
}

export function visibilityText(visibility: string, locale = getAppLocale()) {
  if (visibility === 'class') {
    return translateApp(locale, 'student.visibility.class')
  }
  if (visibility === 'assigned') {
    return translateApp(locale, 'student.visibility.assigned')
  }
  return translateApp(locale, 'student.visibility.private')
}

export type ProgressFilter = 'all' | 'not_started' | 'in_progress' | 'completed'

export function progressFilterLabel(filter: ProgressFilter, locale = getAppLocale()) {
  return translateApp(locale, `student.progressFilter.${filter}`)
}

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
