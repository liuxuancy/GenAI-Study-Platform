import type { MessageTree } from '@/core/i18n/types'

export const zhNav: MessageTree = {
  nav: {
    common: {
      openMenu: '打开导航菜单',
      closeMenu: '关闭导航菜单',
    },
    student: {
      sectionLearning: '学习',
      sectionPersonal: '个人',
      dashboard: '学习仪表盘',
      myCourses: '我的课程',
      aiWorkspace: 'AI 工作台',
      quizzes: '测验中心',
      account: '我的账号',
      quizAttempt: '测验作答',
      courseStudy: '课程学习',
      brandSubtitle: '学生',
      sidebarAria: '学生导航',
    },
    teacher: {
      sectionTeaching: '教学',
      sectionPersonal: '个人',
      courses: '课程中心',
      learningOverview: '学情概览',
      aiWorkspace: 'AI 工作台',
      account: '我的账号',
      brandSubtitle: '教师',
      sidebarAria: '教师导航',
      defaultRole: '讲师',
    },
    admin: {
      users: '用户',
      classes: '班级',
      courses: '课程',
      system: '系统',
      account: '我的账号',
    },
  },
}
