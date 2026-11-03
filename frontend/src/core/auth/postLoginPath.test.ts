import { describe, expect, it } from 'vitest'

import { pathAfterLogin } from '@/core/auth/postLoginPath'

describe('pathAfterLogin', () => {
  it('sends each role to its workspace', () => {
    expect(pathAfterLogin('ROLE_STUDENT')).toBe('/student')
    expect(pathAfterLogin('ROLE_TEACHER')).toBe('/teacher')
    expect(pathAfterLogin('ROLE_ADMIN')).toBe('/admin')
  })

  it('returns admin deep link only for admin routes', () => {
    expect(pathAfterLogin('ROLE_ADMIN', '/admin/system')).toBe('/admin/system')
    expect(pathAfterLogin('ROLE_ADMIN', '/teacher')).toBe('/admin')
    expect(pathAfterLogin('ROLE_TEACHER', '/student')).toBe('/teacher')
  })
})
