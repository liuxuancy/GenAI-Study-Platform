export interface StudentNavItem {
  to: string
  label: string
  end?: boolean
  icon: 'dashboard' | 'book' | 'clipboard' | 'user' | 'spark'
}

export interface StudentNavSection {
  title: string
  items: StudentNavItem[]
}

export const studentNavSections: StudentNavSection[] = [
  {
    title: 'LEARNING',
    items: [
      { to: '/student', label: '学习中心', end: true, icon: 'dashboard' },
      { to: '/student/courses', label: '我的课程', icon: 'book' },
      { to: '/student/ai', label: 'AI 学习工作台', icon: 'spark' },
      { to: '/student/quizzes', label: '我的测验', icon: 'clipboard' },
    ],
  },
  {
    title: 'PERSONAL',
    items: [{ to: '/student/account', label: '我的账号', icon: 'user' }],
  },
]

export const studentPageTitles: Record<string, string> = {
  '/student': '学习中心',
  '/student/courses': '我的课程',
  '/student/ai': 'AI 学习工作台',
  '/student/quizzes': '我的测验',
  '/student/account': '我的账号',
}

export function studentPageTitle(pathname: string) {
  if (pathname.startsWith('/student/questions/')) {
    return '测验答题'
  }
  if (pathname.startsWith('/student/courses/')) {
    return '课程学习'
  }
  return studentPageTitles[pathname] ?? '学习中心'
}

export function isStudentNavActive(pathname: string, item: StudentNavItem) {
  if (item.to === '/student/quizzes') {
    return pathname === item.to || pathname.startsWith('/student/questions/')
  }
  if (item.to === '/student/courses') {
    return pathname === item.to || pathname === `${item.to}/`
  }
  if (item.end) {
    return pathname === item.to || pathname === `${item.to}/`
  }
  return pathname === item.to || pathname.startsWith(`${item.to}/`)
}
