export interface StudentNavItem {
  to: string
  labelKey: string
  end?: boolean
  icon: 'dashboard' | 'book' | 'clipboard' | 'user' | 'spark' | 'trophy'
}

export interface StudentNavSection {
  titleKey: string
  items: StudentNavItem[]
}

export const studentNavSections: StudentNavSection[] = [
  {
    titleKey: 'nav.student.sectionLearning',
    items: [
      { to: '/student', labelKey: 'nav.student.dashboard', end: true, icon: 'dashboard' },
      { to: '/student/courses', labelKey: 'nav.student.myCourses', icon: 'book' },
      { to: '/student/ai', labelKey: 'nav.student.aiWorkspace', icon: 'spark' },
      { to: '/student/quizzes', labelKey: 'nav.student.quizzes', icon: 'clipboard' },
    ],
  },
  {
    titleKey: 'nav.student.sectionPersonal',
    items: [
      { to: '/student/achievements', labelKey: 'nav.student.achievements', icon: 'trophy' },
      { to: '/student/account', labelKey: 'nav.student.account', icon: 'user' },
    ],
  },
]

const studentPageTitleKeys: Record<string, string> = {
  '/student': 'nav.student.dashboard',
  '/student/courses': 'nav.student.myCourses',
  '/student/ai': 'nav.student.aiWorkspace',
  '/student/quizzes': 'nav.student.quizzes',
  '/student/account': 'nav.student.account',
  '/student/achievements': 'nav.student.achievements',
}

export function studentPageTitle(pathname: string, t: (key: string) => string) {
  if (pathname.startsWith('/student/questions/')) {
    return t('nav.student.quizAttempt')
  }
  if (pathname.startsWith('/student/courses/')) {
    return t('nav.student.courseStudy')
  }
  const key = studentPageTitleKeys[pathname] ?? 'nav.student.dashboard'
  return t(key)
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
