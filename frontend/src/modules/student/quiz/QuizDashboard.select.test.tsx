// @vitest-environment jsdom
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { afterEach, expect, it, vi } from 'vitest'

import { QuizDashboard } from './QuizDashboard'

const mocks = vi.hoisted(() => ({ create: vi.fn(), questions: vi.fn() }))
vi.mock('@/modules/student/api', () => ({
  getStudentCourses: () => Promise.resolve([{ id: 7, title: 'Course A', description: '', percent: 0, owned: true, visibility: 'private' }]),
  getStudentQuestions: mocks.questions,
  getStudentCourse: () => Promise.resolve({ tree: null }),
  createStudentQuestions: mocks.create,
}))
vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/core/i18n/store', () => ({ getAppLocale: () => 'en' }))
vi.mock('@/core/i18n/translateApp', () => ({ translateApp: (_locale: string, key: string) => key }))
vi.mock('react-router-dom', () => ({ Link: ({ children }: { children: React.ReactNode }) => <>{children}</> }))
vi.mock('antd', () => ({
  Select: ({ id, value, options, onChange, 'aria-label': label }: {
    id?: string
    value: string | number
    options: { value: string | number; label: React.ReactNode }[]
    onChange: (value: string | number) => void
    'aria-label'?: string
  }) => (
    <select id={id} value={value} aria-label={label} onChange={(event) => {
      const option = options.find((item) => String(item.value) === event.target.value)
      if (option) onChange(option.value)
    }}>
      {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
    </select>
  ),
}))

afterEach(() => {
  document.body.innerHTML = ''
})

it('preserves selected quiz settings when previewing and creating questions', async () => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', {
    configurable: true, value(this: HTMLDialogElement) { this.open = true },
  })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', {
    configurable: true, value(this: HTMLDialogElement) { this.open = false },
  })
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  mocks.create.mockResolvedValue({})
  mocks.questions.mockResolvedValue([])
  await act(async () => root.render(<QuizDashboard />))
  const openButton = Array.from(host.querySelectorAll('button')).find((button) => button.textContent === 'student.quiz.create.button')!
  act(() => openButton.click())
  for (const [id, value] of [['quiz-create-count', '5'], ['quiz-create-difficulty', 'hard'], ['quiz-create-course', '7']]) {
    const select = host.querySelector<HTMLSelectElement>(`#${id}`)!
    act(() => {
      select.value = value
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
  }
  const form = host.querySelector('dialog form')!
  act(() => form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
  expect(host.querySelectorAll('dialog ol li')).toHaveLength(5)
  await act(async () => form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
  expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ courseId: 7, difficulty: 'hard', questions: expect.any(Array) }))
  expect(mocks.create.mock.calls[0][0].questions).toHaveLength(5)
  act(() => root.unmount())
})

it('toggles each course independently and preserves chapter state', async () => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  mocks.questions.mockResolvedValue([
    { id: 1, title: 'Quiz A', courseId: 7, courseTitle: 'Course A', difficulty: 'easy', answered: 0, questionCount: 2, creatorId: 1, creatorName: 'Student', creatorRole: 'ROLE_STUDENT', isSubmit: false },
    { id: 2, title: 'Quiz B', courseId: 8, courseTitle: 'Course B', difficulty: 'easy', answered: 0, questionCount: 2, creatorId: 1, creatorName: 'Student', creatorRole: 'ROLE_STUDENT', isSubmit: false },
  ])
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  await act(async () => root.render(<QuizDashboard />))
  const courseA = Array.from(host.querySelectorAll<HTMLButtonElement>('h2 button')).find((button) => button.textContent?.includes('Course A'))!
  const courseB = Array.from(host.querySelectorAll<HTMLButtonElement>('h2 button')).find((button) => button.textContent?.includes('Course B'))!
  const groupA = courseA.closest('section')!
  const groupB = courseB.closest('section')!
  const bodyA = groupA.querySelector<HTMLDivElement>('h2 + div')!
  const bodyB = groupB.querySelector<HTMLDivElement>('h2 + div')!
  const chapter = bodyA.querySelector<HTMLButtonElement>('button[aria-expanded]')!
  expect(courseA.getAttribute('aria-expanded')).toBe('true')
  expect(courseA.getAttribute('aria-controls')).toBe(bodyA.id)
  expect(bodyA.hidden).toBe(false)
  act(() => chapter.click())
  expect(chapter.getAttribute('aria-expanded')).toBe('false')
  act(() => courseA.click())
  expect(courseA.getAttribute('aria-expanded')).toBe('false')
  expect(bodyA.hidden).toBe(true)
  expect(bodyB.hidden).toBe(false)
  expect(courseB.getAttribute('aria-expanded')).toBe('true')
  act(() => courseA.click())
  expect(bodyA.hidden).toBe(false)
  expect(chapter.getAttribute('aria-expanded')).toBe('false')
  act(() => chapter.click())
  expect(bodyA.textContent).toContain('Quiz A')
  act(() => root.unmount())
})

it('toggles the empty state of a selected course', async () => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  mocks.questions.mockResolvedValue([])
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  await act(async () => root.render(<QuizDashboard />))
  const filter = host.querySelector<HTMLSelectElement>('select[aria-label="student.quiz.courseFilterAria"]')!
  act(() => {
    filter.value = '7'
    filter.dispatchEvent(new Event('change', { bubbles: true }))
  })
  const header = host.querySelector<HTMLButtonElement>('h2 button')!
  const body = header.closest('section')!.querySelector<HTMLDivElement>('h2 + div')!
  expect(body.textContent).toContain('student.quiz.courseEmpty')
  act(() => header.click())
  expect(header.getAttribute('aria-expanded')).toBe('false')
  expect(body.hidden).toBe(true)
  act(() => header.click())
  expect(body.hidden).toBe(false)
  expect(body.textContent).toContain('student.quiz.courseEmpty')
  act(() => root.unmount())
})
