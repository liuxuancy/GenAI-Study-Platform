import { describe, expect, it } from 'vitest'

import { adminHomePath, homePathFor, studentHomePath, teacherHomePath } from '@/core/auth/homePath'

describe('homePathFor', () => {
  it('学生进学生首页', () => {
    expect(homePathFor('ROLE_STUDENT')).toBe(studentHomePath)
  })

  it('讲师进讲师首页，管理员进用户管理', () => {
    expect(homePathFor('ROLE_TEACHER')).toBe(teacherHomePath)
    expect(homePathFor('ROLE_ADMIN')).toBe(adminHomePath)
  })
})
