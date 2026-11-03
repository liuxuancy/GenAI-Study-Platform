import { describe, expect, it } from 'vitest'

import {
  roleForAudience,
  roleMatchesAudience,
} from '@/core/auth/loginAudience'
import { audienceMismatchMessage } from '@/core/auth/loginAudienceMessages'

describe('loginAudience', () => {
  it('maps audience to role', () => {
    expect(roleForAudience('student')).toBe('ROLE_STUDENT')
    expect(roleForAudience('teacher')).toBe('ROLE_TEACHER')
    expect(roleForAudience('admin')).toBe('ROLE_ADMIN')
  })

  it('matches only the selected audience role', () => {
    expect(roleMatchesAudience('ROLE_STUDENT', 'student')).toBe(true)
    expect(roleMatchesAudience('ROLE_TEACHER', 'student')).toBe(false)
    expect(roleMatchesAudience('ROLE_ADMIN', 'teacher')).toBe(false)
  })

  it('includes audience label in mismatch message', () => {
    expect(audienceMismatchMessage('teacher', 'zh')).toContain('教师')
  })
})
