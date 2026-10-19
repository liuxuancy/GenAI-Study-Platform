import { describe, expect, it } from 'vitest'

import permissionCases from '../../../docs/permission-cases.json'
import type { CurrentUser } from '@/core/api/account'
import { hasPermission, isValidPermissionCode, matchPermission } from '@/core/permission'

describe('matchPermission', () => {
  it.each(permissionCases.cases)('$granted 对 $required → $match（$note）', (testCase) => {
    expect(matchPermission(testCase.granted, testCase.required)).toBe(testCase.match)
  })
})

describe('isValidPermissionCode', () => {
  it('合法和非法的格式', () => {
    expect(isValidPermissionCode('teacher:course:view')).toBe(true)
    expect(isValidPermissionCode('teacher:*')).toBe(true)
    expect(isValidPermissionCode('teacher:*:view')).toBe(false)
    expect(isValidPermissionCode('teacher::view')).toBe(false)
    expect(isValidPermissionCode('')).toBe(false)
  })
})

describe('hasPermission', () => {
  const student: CurrentUser = {
    id: 3,
    username: 'student1',
    nickname: '',
    email: '',
    role: 'ROLE_STUDENT',
    roleName: '学生',
    isAdmin: false,
    permissions: ['student:*'],
  }
  const admin: CurrentUser = { ...student, id: 1, role: 'ROLE_ADMIN', isAdmin: true, permissions: [] }

  it('未登录一律没有权限', () => {
    expect(hasPermission(null, 'student:course:view')).toBe(false)
  })

  it('按授予的权限码匹配', () => {
    expect(hasPermission(student, 'student:course:view')).toBe(true)
    expect(hasPermission(student, 'teacher:course:view')).toBe(false)
  })

  it('管理员全部放行', () => {
    expect(hasPermission(admin, 'teacher:course:update')).toBe(true)
  })
})
