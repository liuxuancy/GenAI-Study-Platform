import type { ActivityRef } from '@/modules/student/courseTreeNav'
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'

export function isLastActivityInChapter(activityId: string, activities: readonly ActivityRef[]) {
  const current = activities.find((item) => item.id === activityId)
  if (!current) {
    return false
  }
  const inChapter = activities.filter((item) => item.chapterId === current.chapterId)
  return inChapter.length > 0 && inChapter[inChapter.length - 1]?.id === activityId
}

export type LessonPrimaryAction = 'complete-next' | 'next' | 'complete-chapter' | 'next-chapter' | 'finished'

export function lessonPrimaryAction(
  activityId: string,
  activities: readonly ActivityRef[],
  done: boolean,
  locale = getAppLocale(),
) {
  const last = isLastActivityInChapter(activityId, activities)
  const hasNextChapter = firstActivityOfNextChapter(activityId, activities) !== null
  if (!last && !done) {
    return {
      label: translateApp(locale, 'student.lesson.completeNext'),
      action: 'complete-next' as const,
      disabled: false,
    }
  }
  if (!last && done) {
    return { label: translateApp(locale, 'student.lesson.next'), action: 'next' as const, disabled: false }
  }
  if (!done) {
    return {
      label: translateApp(locale, 'student.lesson.completeChapter'),
      action: 'complete-chapter' as const,
      disabled: false,
    }
  }
  if (hasNextChapter) {
    return {
      label: translateApp(locale, 'student.lesson.nextChapter'),
      action: 'next-chapter' as const,
      disabled: false,
    }
  }
  return { label: translateApp(locale, 'student.lesson.finished'), action: 'finished' as const, disabled: true }
}

export function shouldAdvanceAfterComplete(activityId: string, activities: readonly ActivityRef[]) {
  return nextActivityInSameChapter(activityId, activities) !== null
}

export function nextActivityInSameChapter(activityId: string, activities: readonly ActivityRef[]) {
  const current = activities.find((item) => item.id === activityId)
  if (!current) {
    return null
  }
  const inChapter = activities.filter((item) => item.chapterId === current.chapterId)
  const index = inChapter.findIndex((item) => item.id === activityId)
  if (index < 0 || index >= inChapter.length - 1) {
    return null
  }
  return inChapter[index + 1]?.id ?? null
}

export function firstActivityOfNextChapter(activityId: string, activities: readonly ActivityRef[]) {
  const index = activities.findIndex((item) => item.id === activityId)
  if (index < 0) {
    return null
  }
  const chapterId = activities[index]?.chapterId
  for (let cursor = index + 1; cursor < activities.length; cursor += 1) {
    const candidate = activities[cursor]
    if (candidate && candidate.chapterId !== chapterId) {
      return candidate.id
    }
  }
  return null
}

export function doneIdsAfterComplete(doneIds: ReadonlySet<string>, activityId: string) {
  if (doneIds.has(activityId)) {
    return { next: new Set(doneIds), changed: false }
  }
  const next = new Set(doneIds)
  next.add(activityId)
  return { next, changed: true }
}

/** 将一章内所有小节标记为已完成。 */
export function markChapterActivitiesDone(
  chapterId: string,
  activities: readonly ActivityRef[],
  doneIds: ReadonlySet<string>,
) {
  const next = new Set(doneIds)
  let changed = false
  for (const activity of activities) {
    if (activity.chapterId !== chapterId || next.has(activity.id)) {
      continue
    }
    next.add(activity.id)
    changed = true
  }
  return { next, changed }
}

export function mergeProgressPercent(serverPercent: number, computedPercent: number) {
  return Math.max(clampPercent(serverPercent), clampPercent(computedPercent))
}

export function progressStorageKey(userId: number, courseId: string) {
  return `caveclass.lesson-done.${userId}.${courseId}`
}

export function readDoneIds(raw: string | null, validIds: ReadonlySet<string>) {
  if (!raw) {
    return []
  }
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) {
      return []
    }
    return parsed.filter((item): item is string => typeof item === 'string' && validIds.has(item))
  } catch {
    return []
  }
}

function clampPercent(value: number) {
  if (!Number.isFinite(value) || value < 0) {
    return 0
  }
  if (value > 100) {
    return 100
  }
  return Math.floor(value)
}
