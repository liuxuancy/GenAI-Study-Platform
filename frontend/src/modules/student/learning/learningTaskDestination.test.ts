import { describe, expect, it } from 'vitest'

import { learningTaskDestination } from './learningTaskDestination'

describe('learningTaskDestination', () => {
  it('opens the specific course for study tasks', () => {
    expect(learningTaskDestination('course-42')).toBe('/student/courses/42')
  })

  it('opens the specific question for quiz and review tasks', () => {
    expect(learningTaskDestination('quiz-17')).toBe('/student/questions/17')
    expect(learningTaskDestination('review-17')).toBe('/student/questions/17')
  })

  it('rejects invalid and unrecognized task identifiers', () => {
    for (const id of ['course-0', 'course--1', 'quiz-1/extra', 'review-abc', 'other-1', 'course-01']) {
      expect(learningTaskDestination(id)).toBeNull()
    }
  })
})
