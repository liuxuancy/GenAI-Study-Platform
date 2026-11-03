import type { MessageTree } from '@/core/i18n/types'

export const enCommon: MessageTree = {
  language: {
    zh: '中文',
    en: 'EN',
    switchAria: 'Switch language',
  },
  roles: {
    ROLE_STUDENT: 'Student',
    ROLE_TEACHER: 'Teacher',
    ROLE_ADMIN: 'Administrator',
  },
  common: {
    signOut: 'Sign out',
    loading: 'Loading…',
    save: 'Save',
    cancel: 'Cancel',
    delete: 'Delete',
    retry: 'Retry',
    confirm: 'Confirm',
    brandName: 'CaveClass',
    permissionDenied: 'You do not have permission for this action.',
    dash: '—',
  },
  auth: {
    loadAccountFailed: 'Failed to load your account',
  },
  notFound: {
    title: 'Page not found',
    backHome: 'Back to home',
  },
  http: {
    networkError: 'Network error: is the backend running?',
  },
  layout: {
    admin: {
      users: 'Users',
      classes: 'Classes',
      courses: 'Courses',
      system: 'System',
      account: 'Account',
    },
    teacher: {
      courses: 'Courses',
      account: 'Account',
    },
    student: {
      courses: 'Courses',
      account: 'Account',
    },
    home: 'Home',
  },
}
