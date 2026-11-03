import type { MessageTree } from '@/core/i18n/types'

export const zhCommon: MessageTree = {
  language: {
    zh: '中文',
    en: 'EN',
    switchAria: '切换语言',
  },
  roles: {
    ROLE_STUDENT: '学生',
    ROLE_TEACHER: '教师',
    ROLE_ADMIN: '管理员',
  },
  common: {
    signOut: '退出登录',
    loading: '加载中…',
    save: '保存',
    cancel: '取消',
    delete: '删除',
    retry: '重试',
    confirm: '确认',
    brandName: 'CaveClass',
    permissionDenied: '无权执行此操作。',
    dash: '—',
  },
  auth: {
    loadAccountFailed: '无法加载账号信息',
  },
  notFound: {
    title: '页面不存在',
    backHome: '返回首页',
  },
  http: {
    networkError: '网络异常，请检查连接或确认后端已启动',
  },
  layout: {
    admin: {
      users: '用户',
      classes: '班级',
      courses: '课程',
      system: '系统',
      account: '我的账号',
    },
    teacher: {
      courses: '课程',
      account: '我的账号',
    },
    student: {
      courses: '课程',
      account: '我的账号',
    },
    home: '首页',
  },
}
