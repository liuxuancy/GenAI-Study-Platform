import type { MessageTree } from '@/core/i18n/types'

export const enNav: MessageTree = {
  nav: {
    common: {
      openMenu: 'Open navigation menu',
      closeMenu: 'Close navigation menu',
    },
    student: {
      sectionLearning: 'Learning',
      sectionPersonal: 'Personal',
      dashboard: 'Dashboard',
      myCourses: 'My courses',
      aiWorkspace: 'AI workspace',
      quizzes: 'Quizzes',
      account: 'Account',
      achievements: 'Achievements',
      quizAttempt: 'Quiz attempt',
      courseStudy: 'Course study',
      brandSubtitle: 'Student',
      sidebarAria: 'Student navigation',
    },
    teacher: {
      sectionTeaching: 'Teaching',
      sectionPersonal: 'Personal',
      courses: 'Course hub',
      learningOverview: 'Learning overview',
      aiWorkspace: 'AI workspace',
      account: 'Account',
      brandSubtitle: 'Teacher',
      sidebarAria: 'Teacher navigation',
      defaultRole: 'Instructor',
    },
    admin: {
      users: 'Users',
      classes: 'Classes',
      courses: 'Courses',
      system: 'System',
      account: 'Account',
    },
  },
}
