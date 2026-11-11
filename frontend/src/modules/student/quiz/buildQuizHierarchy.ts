import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'
import type { StudentCourse, StudentQuestion } from '@/modules/student/api'
import type { BubbleNode } from '@/modules/student/bubbleTree'
import { courseSourceLabel, courseSourceOf } from '@/modules/student/learning/courseSource'
import { mapStudentQuestionsToViews } from '@/modules/student/quiz/mapStudentQuiz'
import type {
  ChapterQuizGroup,
  CourseQuizGroup,
  QuizCreationFilter,
  QuizStatusFilter,
  SectionQuizGroup,
  StudentQuizView,
} from '@/modules/student/quiz/quizTypes'
import { UNLINKED_CHAPTER_KEY } from '@/modules/student/quiz/quizTypes'

export function buildCourseQuizGroups(
  courses: StudentCourse[],
  filteredViews: StudentQuizView[],
  courseTrees: Map<number, BubbleNode | null>,
  options: { courseId: string },
): CourseQuizGroup[] {
  const viewsByCourse = new Map<string, StudentQuizView[]>()
  for (const view of filteredViews) {
    const list = viewsByCourse.get(view.courseId) ?? []
    list.push(view)
    viewsByCourse.set(view.courseId, list)
  }

  const locale = getAppLocale()
  const unlinkedTitle = translateApp(locale, 'student.quiz.unlinkedNode')
  const groups: CourseQuizGroup[] = []
  const scopedCourses = options.courseId === 'all'
    ? courses
    : courses.filter((course) => String(course.id) === options.courseId)
  for (const course of scopedCourses) {
    const courseViews = viewsByCourse.get(String(course.id)) ?? []
    if (options.courseId === 'all' && courseViews.length === 0) {
      continue
    }
    const tree = courseTrees.get(course.id) ?? null
    groups.push({
      courseId: String(course.id),
      courseTitle: course.title,
      courseSourceLabel: courseSourceLabel(courseSourceOf(course)),
      creatorLabel: course.owned
        ? translateApp(getAppLocale(), 'student.courseSource.personal')
        : courseSourceLabel(courseSourceOf(course)),
      quizTotal: courseViews.length,
      completedCount: courseViews.filter((view) => view.attempt.status === 'completed').length,
      chapters: buildChapterGroups(courseViews, tree, unlinkedTitle),
    })
  }

  const listedIds = new Set(scopedCourses.map((course) => String(course.id)))
  for (const [courseId, courseViews] of viewsByCourse) {
    if (options.courseId !== 'all' && courseId !== options.courseId) {
      continue
    }
    if (listedIds.has(courseId) || courseViews.length === 0) {
      continue
    }
    groups.push({
      courseId,
      courseTitle: courseViews[0]?.courseName || translateApp(locale, 'student.quiz.unnamedCourse'),
      courseSourceLabel: courseViews[0]?.courseSourceLabel || translateApp(locale, 'student.quiz.fallbackCourseLabel'),
      creatorLabel: translateApp(locale, 'student.quiz.creatorCourseQuiz'),
      quizTotal: courseViews.length,
      completedCount: courseViews.filter((view) => view.attempt.status === 'completed').length,
      chapters: buildChapterGroups(courseViews, null, unlinkedTitle),
    })
  }

  return groups.sort((left, right) => left.courseTitle.localeCompare(right.courseTitle))
}

function buildChapterGroups(views: StudentQuizView[], tree: BubbleNode | null, unlinkedTitle: string): ChapterQuizGroup[] {
  if (views.length === 0) {
    return []
  }

  const byChapter = new Map<string, StudentQuizView[]>()
  for (const view of views) {
    const key = view.chapterKey?.trim() || UNLINKED_CHAPTER_KEY
    const list = byChapter.get(key) ?? []
    list.push(view)
    byChapter.set(key, list)
  }

  const ordered: { key: string; title: string }[] = []
  for (const chapter of tree?.children ?? []) {
    if (chapter.key && byChapter.has(chapter.key)) {
      ordered.push({ key: chapter.key, title: chapter.label })
    }
  }
  for (const [key, chapterViews] of byChapter) {
    if (key === UNLINKED_CHAPTER_KEY || ordered.some((item) => item.key === key)) {
      continue
    }
    ordered.push({
      key,
      title: chapterViews[0]?.chapterMissing
        ? translateApp(getAppLocale(), 'student.quiz.deletedChapter', { chapter: chapterViews[0].chapterName?.trim() || key })
        : chapterViews[0]?.chapterName?.trim() || key,
    })
  }
  if (byChapter.has(UNLINKED_CHAPTER_KEY)) {
    ordered.push({ key: UNLINKED_CHAPTER_KEY, title: unlinkedTitle })
  }

  return ordered.map(({ key, title }) => {
    const chapterViews = byChapter.get(key) ?? []
    const chapterNode = tree?.children?.find((node) => node.key === key)
    const sections = buildSectionGroups(chapterViews, chapterNode)
    const sectionQuizIds = new Set(sections.flatMap((section) => section.quizzes.map((quiz) => quiz.id)))
    const chapterQuizzes = chapterViews.filter((view) => !sectionQuizIds.has(view.id))
    return {
      chapterKey: key,
      chapterTitle: title,
      sections,
      chapterQuizzes,
    }
  })
}

function buildSectionGroups(views: StudentQuizView[], chapterNode: BubbleNode | undefined): SectionQuizGroup[] {
  const sectionQuizzes = views.filter((view) => view.targetType === 'section' && view.sectionKey)
  if (sectionQuizzes.length === 0) {
    return []
  }
  const bySection = new Map<string, StudentQuizView[]>()
  for (const view of sectionQuizzes) {
    const key = view.sectionKey ?? ''
    const list = bySection.get(key) ?? []
    list.push(view)
    bySection.set(key, list)
  }
  const sections: SectionQuizGroup[] = []
  for (const activity of chapterNode?.children ?? []) {
    if (!activity.key) {
      continue
    }
    const quizzes = bySection.get(activity.key)
    if (!quizzes?.length) {
      continue
    }
    sections.push({
      sectionKey: activity.key,
      sectionTitle: activity.label,
      quizzes,
    })
  }
  return sections
}

export function filterQuizViews(
  views: StudentQuizView[],
  options: {
    status: QuizStatusFilter
    courseId: string
    chapterKey: string
    creationMethod: QuizCreationFilter
  },
): StudentQuizView[] {
  return views.filter((view) => {
    if (options.status !== 'all' && view.attempt.status !== options.status) {
      return false
    }
    if (options.courseId !== 'all' && view.courseId !== options.courseId) {
      return false
    }
    if (options.chapterKey !== 'all') {
      const key = view.chapterKey?.trim() || UNLINKED_CHAPTER_KEY
      if (key !== options.chapterKey) {
        return false
      }
    }
    if (options.creationMethod !== 'all' && view.creationMethod !== options.creationMethod) {
      return false
    }
    return true
  })
}

export function chapterFilterOptions(views: StudentQuizView[], courseId: string) {
  const locale = getAppLocale()
  const scoped = courseId === 'all' ? views : views.filter((view) => view.courseId === courseId)
  const unlinkedTitle = translateApp(locale, 'student.quiz.unlinkedNode')
  const options: { value: string; label: string }[] = [
    { value: 'all', label: translateApp(locale, 'student.quiz.chapterFilterAll') },
  ]
  const seen = new Set<string>()
  for (const view of scoped) {
    const key = view.chapterKey?.trim() || UNLINKED_CHAPTER_KEY
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    options.push({
      value: key,
      label: key === UNLINKED_CHAPTER_KEY ? unlinkedTitle : view.chapterMissing
        ? translateApp(locale, 'student.quiz.deletedChapter', { chapter: view.chapterName?.trim() || key })
        : view.chapterName?.trim() || key,
    })
  }
  return options
}

export function allViewsFromQuestions(courses: StudentCourse[], questions: StudentQuestion[]) {
  return mapStudentQuestionsToViews(questions, courses)
}
