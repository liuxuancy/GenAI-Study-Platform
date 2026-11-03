import type { CourseContent } from '@/modules/teacher/api'

type Translate = (key: string, params?: Record<string, string | number>) => string

export function buildTeacherResourceScopeOptions(
  content: CourseContent | null,
  t: Translate,
): { value: string; label: string }[] {
  const chapters = content?.chapters ?? []
  const options: { value: string; label: string }[] = [
    { value: 'course', label: t('courseResources.teacherPanel.scopeCourse') },
  ]
  chapters.forEach((ch, index) => {
    options.push({
      value: `chapter:${ch.key}`,
      label: t('courseResources.teacherPanel.chapterLabel', { index: index + 1, title: ch.title }),
    })
    ch.activities.forEach((act) => {
      options.push({
        value: `section:${ch.key}:${act.key}`,
        label: `${ch.title} · ${act.title}`,
      })
    })
  })
  return options
}
