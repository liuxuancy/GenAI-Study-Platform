import { describe, expect, it, vi } from 'vitest'

import { displayQuizTitle } from './mapStudentQuiz'

vi.mock('@/core/i18n/store', () => ({ getAppLocale: () => 'en' }))
vi.mock('@/core/i18n/translateApp', () => ({
  translateApp: (_locale: string, key: string, params?: { chapter: string }) =>
    params ? `${params.chapter} quiz` : key,
}))

describe('displayQuizTitle', () => {
  it('removes generated student quiz metadata', () => {
    expect(displayQuizTitle('Course · medium · 2026-11-10 10:30:25')).toBe('Course')
  })

  it('uses the chapter name for generated quizzes, falling back to the chapter key', () => {
    const title = 'Quiz · chapter-1 · 42 · 2026-11-10 10:30:25'
    expect(displayQuizTitle(title, 'Introduction')).toBe('Introduction quiz')
    expect(displayQuizTitle(title)).toBe('chapter-1')
  })

  it('uses the same names as the quiz list for imported and manual quizzes', () => {
    expect(displayQuizTitle('Excel 导入 · 42 · 1234')).toBe('student.quiz.titleExcelImport')
    expect(displayQuizTitle('教师题库 · 42 · 1234')).toBe('student.quiz.titleManualQuiz')
  })

  it('preserves ordinary titles, including intentional dates', () => {
    expect(displayQuizTitle('Final exam 2026-11-10')).toBe('Final exam 2026-11-10')
  })
})
