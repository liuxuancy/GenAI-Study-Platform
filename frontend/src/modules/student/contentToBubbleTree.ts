import type { CourseActivity, CourseContent } from '@/modules/teacher/api'

import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'
import type { BubbleNode } from '@/modules/student/bubbleTree'

function activityNote(activity: CourseActivity): string {
  if (activity.type === 'rich_text') {
    return String(activity.payload.markdown ?? '')
  }
  if (activity.type === 'coding') {
    return String(activity.payload.promptMarkdown ?? '')
  }
  if (activity.type === 'mock_interview') {
    return String(activity.payload.scenarioMarkdown ?? '')
  }
  return String(activity.payload.instructionsMarkdown ?? '')
}

export function contentToBubbleTree(
  content: CourseContent,
  chapterQuestionIds: ReadonlyMap<string, number> = new Map(),
  locale = getAppLocale(),
): BubbleNode {
  const chapters = [...content.chapters].sort((left, right) => left.order - right.order)
  return {
    label: content.title,
    note: content.description?.trim() || translateApp(locale, 'student.bubbleMap.courseNoteDefault'),
    kind: 'course',
    children: chapters.map((chapter) => {
      const activities = [...chapter.activities].sort((left, right) => left.order - right.order)
      return {
        key: chapter.key,
        label: chapter.title,
        note: chapter.summary?.trim() || translateApp(locale, 'student.bubbleMap.chapterSummaryPending'),
        kind: 'chapter',
        questionId: chapterQuestionIds.get(chapter.key),
        children: activities.map((activity) => ({
          key: activity.key,
          label: activity.title,
          note: activityNote(activity).trim() || translateApp(locale, 'student.bubbleMap.sectionContentPending'),
          kind: 'activity',
          activityType: activity.type,
        })),
      }
    }),
  }
}
