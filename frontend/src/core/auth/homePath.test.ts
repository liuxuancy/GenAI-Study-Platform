import { describe, expect, it } from 'vitest'

import { homePathFor, studentHomePath, teacherHomePath } from '@/core/auth/homePath'

describe('homePathFor', () => {
  it('学生进学生首页', () => {
    expect(homePathFor('ROLE_STUDENT')).toBe(studentHomePath)
  })

  it('讲师进讲师首页，管理员进现有首页', () => {
    expect(homePathFor('ROLE_TEACHER')).toBe(teacherHomePath)
    expect(homePathFor('ROLE_ADMIN')).toBe('/')
  })
})
