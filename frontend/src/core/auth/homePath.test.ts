import { describe, expect, it } from 'vitest'

import { homePathFor, studentHomePath } from '@/core/auth/homePath'

describe('homePathFor', () => {
  it('学生进学生首页', () => {
    expect(homePathFor('ROLE_STUDENT')).toBe(studentHomePath)
  })

  it('讲师和管理员进现有首页', () => {
    expect(homePathFor('ROLE_TEACHER')).toBe('/')
    expect(homePathFor('ROLE_ADMIN')).toBe('/')
  })
})
