import { describe, expect, it } from 'vitest'

import { CYCLE_COVER_KEYS, resolveCourseCoverKey } from '@/modules/student/studentCourseVisuals'

describe('resolveCourseCoverKey', () => {
  it('matches thematic titles before cycling', () => {
    expect(resolveCourseCoverKey('Intro to Java', '', 99)).toBe('java')
    expect(resolveCourseCoverKey('Git workshop', '', 1)).toBe('git')
  })

  it('cycles palette by course id when no keyword matches', () => {
    expect(resolveCourseCoverKey('My course A', '', 1)).toBe(CYCLE_COVER_KEYS[1 % CYCLE_COVER_KEYS.length])
    expect(resolveCourseCoverKey('My course B', '', 2)).toBe(CYCLE_COVER_KEYS[2 % CYCLE_COVER_KEYS.length])
    expect(resolveCourseCoverKey('Another', '', 3)).not.toBe(
      resolveCourseCoverKey('Another', '', 2),
    )
  })

  it('falls back to first palette entry without course id', () => {
    expect(resolveCourseCoverKey('Untitled')).toBe(CYCLE_COVER_KEYS[0])
  })
})
