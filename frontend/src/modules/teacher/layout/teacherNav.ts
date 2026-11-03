export interface TeacherNavItem {
  to: string
  label: string
  end?: boolean
  icon: 'courses' | 'spark' | 'user'
}

export interface TeacherNavSection {
  title: string
  items: TeacherNavItem[]
}

export const teacherNavSections: TeacherNavSection[] = [
  {
    title: 'TEACHING',
    items: [
      { to: '/teacher', label: '课程中心', end: true, icon: 'courses' },
      { to: '/teacher/ai', label: 'AI 工作台', icon: 'spark' },
    ],
  },
  {
    title: 'PERSONAL',
    items: [{ to: '/teacher/account', label: '我的账号', icon: 'user' }],
  },
]

export const teacherPageTitles: Record<string, string> = {
  '/teacher': '课程中心',
  '/teacher/ai': 'AI 工作台',
  '/teacher/account': '我的账号',
}

export function teacherPageTitle(pathname: string) {
  return teacherPageTitles[pathname] ?? '课程中心'
}

export function isTeacherNavActive(pathname: string, item: TeacherNavItem) {
  if (item.end) {
    return pathname === item.to || pathname === `${item.to}/`
  }
  return pathname === item.to || pathname.startsWith(`${item.to}/`)
}
