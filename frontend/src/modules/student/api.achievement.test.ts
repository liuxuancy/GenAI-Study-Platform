// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { createStudentCourse } from './api'

const mocks = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/core/http', () => ({ http: { post: mocks.post } }))

afterEach(() => {
  mocks.post.mockReset()
})

it('requests achievement sync only after successful course creation', async () => {
  const listener = vi.fn()
  window.addEventListener('student-achievements-changed', listener)
  try {
    const input = { title: 'Course', description: '', provider: 'deepseek' as const }
    mocks.post.mockRejectedValueOnce(new Error('failed')).mockResolvedValueOnce({ id: 1 })
    await expect(createStudentCourse(input)).rejects.toThrow('failed')
    expect(listener).not.toHaveBeenCalled()
    await expect(createStudentCourse(input)).resolves.toEqual({ id: 1 })
    expect(listener).toHaveBeenCalledOnce()
  } finally {
    window.removeEventListener('student-achievements-changed', listener)
  }
})
