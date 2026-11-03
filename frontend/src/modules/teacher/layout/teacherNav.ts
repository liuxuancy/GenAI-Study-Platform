export interface TeacherNavItem {
  to: string
  labelKey: string
  end?: boolean
  icon: 'courses' | 'spark' | 'user' | 'learning'
}

export interface TeacherNavSection {
  titleKey: string
  items: TeacherNavItem[]
}

export const teacherNavSections: TeacherNavSection[] = [
  {
    titleKey: 'nav.teacher.sectionTeaching',
    items: [
      { to: '/teacher', labelKey: 'nav.teacher.courses', end: true, icon: 'courses' },
      { to: '/teacher/learning', labelKey: 'nav.teacher.learningOverview', icon: 'learning' },
      { to: '/teacher/ai', labelKey: 'nav.teacher.aiWorkspace', icon: 'spark' },
    ],
  },
  {
    titleKey: 'nav.teacher.sectionPersonal',
    items: [{ to: '/teacher/account', labelKey: 'nav.teacher.account', icon: 'user' }],
  },
]

const teacherPageTitleKeys: Record<string, string> = {
  '/teacher': 'nav.teacher.courses',
  '/teacher/learning': 'nav.teacher.learningOverview',
  '/teacher/ai': 'nav.teacher.aiWorkspace',
  '/teacher/account': 'nav.teacher.account',
}

export function teacherPageTitle(pathname: string, t: (key: string) => string) {
  const key = teacherPageTitleKeys[pathname] ?? 'nav.teacher.courses'
  return t(key)
}

export function isTeacherNavActive(pathname: string, item: TeacherNavItem) {
  if (item.end) {
    return pathname === item.to || pathname === `${item.to}/`
  }
  return pathname === item.to || pathname.startsWith(`${item.to}/`)
}
