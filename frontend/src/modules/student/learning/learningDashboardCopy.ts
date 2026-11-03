import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'

type Translate = (key: string, params?: Record<string, string | number>) => string

const TASK_TYPE_KEYS: Record<string, string> = {
  学习: 'student.dashboard.taskTypeStudy',
  测验: 'student.dashboard.taskTypeQuiz',
  复习: 'student.dashboard.taskTypeReview',
  Study: 'student.dashboard.taskTypeStudy',
  Quiz: 'student.dashboard.taskTypeQuiz',
  Review: 'student.dashboard.taskTypeReview',
}

const ACTION_TITLE_BACKEND: Record<string, string> = {
  继续当前课程: 'student.dashboard.actionContinueTitle',
  完成待办测验: 'student.dashboard.actionQuizTitle',
  探索课程库: 'student.dashboard.actionExploreTitle',
}

function t(key: string, params?: Record<string, string | number>, locale = getAppLocale()) {
  return translateApp(locale, key, params)
}

export function localizeLearningTaskType(type: string, translate: Translate = t) {
  const key = TASK_TYPE_KEYS[type.trim()]
  return key ? translate(key) : type
}

export function localizeLearningTaskTitle(title: string, translate: Translate = t) {
  const trimmed = title.trim()
  const patterns: { re: RegExp; key: string }[] = [
    { re: /^继续\s+(.+)$/, key: 'student.dashboard.taskTitleContinue' },
    { re: /^开始\s+(.+)$/, key: 'student.dashboard.taskTitleStart' },
    { re: /^完成\s+(.+)$/, key: 'student.dashboard.taskTitleComplete' },
    { re: /^复习\s+(.+)$/, key: 'student.dashboard.taskTitleReviewItem' },
    { re: /^Continue\s+(.+)$/i, key: 'student.dashboard.taskTitleContinue' },
    { re: /^Start\s+(.+)$/i, key: 'student.dashboard.taskTitleStart' },
    { re: /^Complete\s+(.+)$/i, key: 'student.dashboard.taskTitleComplete' },
    { re: /^Review\s+(.+)$/i, key: 'student.dashboard.taskTitleReviewItem' },
  ]
  for (const { re, key } of patterns) {
    const match = re.exec(trimmed)
    if (match?.[1]) {
      return translate(key, { name: match[1].trim() })
    }
  }
  return title
}

function localizeRulePart(part: string, translate: Translate) {
  const text = part.trim().replace(/。$/, '')
  if (!text) {
    return part
  }
  if (text === '你还没有可学习的课程。前往「我的课程」创建一门课，或等待教师发布到班级') {
    return translate('student.dashboard.ruleNoCourses')
  }
  if (text === '从「我的课程」选择一门课开始学习，进度与测验将在此汇总') {
    return translate('student.dashboard.rulePickCourse')
  }
  let match = /^建议优先继续「(.+)」（当前进度 (\d+)%）$/.exec(text)
  if (match) {
    return translate('student.dashboard.ruleContinue', { title: match[1], percent: match[2] })
  }
  match = /^「(.+)」已完成，可以在测验中心巩固或探索其他课程$/.exec(text)
  if (match) {
    return translate('student.dashboard.ruleCourseDone', { title: match[1] })
  }
  match = /^测验中心有 (\d+) 项待完成，可与当前章节配合练习$/.exec(text)
  if (match) {
    return translate('student.dashboard.rulePendingQuiz', { count: match[1] })
  }
  return part
}

export function localizeRuleSuggestion(raw: string, translate: Translate = t) {
  const trimmed = raw.trim()
  if (!trimmed) {
    return raw
  }
  const endsWithPeriod = trimmed.endsWith('。')
  const body = endsWithPeriod ? trimmed.slice(0, -1) : trimmed
  const parts = body.split('；').map((segment) => localizeRulePart(segment, translate))
  const joiner = translate('student.dashboard.ruleJoin')
  const joined = parts.join(joiner)
  if (getAppLocale() === 'zh') {
    return endsWithPeriod ? `${joined}。` : joined
  }
  return endsWithPeriod ? `${joined}.` : joined
}

export function localizeRecommendedAction(
  action: { title: string; detail: string },
  translate: Translate = t,
) {
  const titleKey = ACTION_TITLE_BACKEND[action.title.trim()]
  if (!titleKey) {
    return action
  }
  if (titleKey === 'student.dashboard.actionContinueTitle') {
    const match = /进入「(.+)」，完成下一章节内容。?$/.exec(action.detail.trim())
    return {
      title: translate(titleKey),
      detail: translate('student.dashboard.actionContinueDetail', { course: match?.[1] ?? action.detail }),
    }
  }
  if (titleKey === 'student.dashboard.actionQuizTitle') {
    return {
      title: translate(titleKey),
      detail: translate('student.dashboard.actionQuizDetail'),
    }
  }
  return {
    title: translate(titleKey),
    detail: translate('student.dashboard.actionExploreDetail'),
  }
}
