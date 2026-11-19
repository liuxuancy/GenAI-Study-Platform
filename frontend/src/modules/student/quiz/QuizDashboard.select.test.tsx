// @vitest-environment jsdom
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { afterEach, expect, it, vi } from 'vitest'

import { QuizDashboard } from './QuizDashboard'

const mocks = vi.hoisted(() => ({ create: vi.fn(), questions: vi.fn(), course: vi.fn() }))
vi.mock('@/modules/student/api', () => ({
  getStudentCourses: () => Promise.resolve([{ id: 7, title: 'Course A', description: '', percent: 0, owned: true, visibility: 'private' }]),
  getStudentQuestions: mocks.questions,
  getStudentCourse: mocks.course,
  generateStudentQuiz: mocks.create,
}))
vi.mock('@/core/ai/useAiProviderSelect', () => ({
  useAiProviderSelect: () => ({
    provider: 'deepseek', options: [{ value: 'deepseek', label: 'DeepSeek' }], setProvider: vi.fn(),
    loading: false, optionsAvailable: true, error: '',
  }),
}))
vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/core/i18n/store', () => ({ getAppLocale: () => 'en' }))
vi.mock('@/core/i18n/translateApp', () => ({ translateApp: (_locale: string, key: string, params?: { chapter: string }) => params ? `${params.chapter} (deleted)` : key }))
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
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

it('generates and saves a quiz with the selected settings', async () => {
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
  mocks.create.mockResolvedValue({ id: 99 })
  mocks.questions.mockResolvedValue([])
  mocks.course.mockResolvedValue({ tree: { label: 'Course A', children: [{ key: 'ch-1', label: 'Chapter A' }] } })
  await act(async () => root.render(<QuizDashboard />))
  const openButton = Array.from(host.querySelectorAll('button')).find((button) => button.textContent === 'student.quiz.create.button')!
  act(() => openButton.click())
  for (const [id, value] of [['quiz-create-count', '5'], ['quiz-create-difficulty', 'hard'], ['quiz-create-course', '7']]) {
    const select = host.querySelector<HTMLSelectElement>(`#${id}`)!
    await act(async () => {
      select.value = value
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
  }
  const chapter = host.querySelector<HTMLSelectElement>('#quiz-create-chapter')!
  act(() => {
    chapter.value = 'ch-1'
    chapter.dispatchEvent(new Event('change', { bubbles: true }))
  })
  const form = host.querySelector('dialog form')!
  await act(async () => form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
  expect(mocks.create).toHaveBeenCalledWith(7, { chapterKey: 'ch-1', count: 5, difficulty: 'hard', provider: 'deepseek' })
  expect(host.querySelector('dialog ol')).toBeNull()
  act(() => root.unmount())
})

it('explains why quiz generation is disabled before a course is selected', async () => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', {
    configurable: true, value(this: HTMLDialogElement) { this.open = true },
  })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', {
    configurable: true, value(this: HTMLDialogElement) { this.open = false },
  })
  mocks.questions.mockResolvedValue([])
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  await act(async () => root.render(<QuizDashboard />))
  act(() => Array.from(host.querySelectorAll('button')).find((button) => button.textContent === 'student.quiz.create.button')!.click())

  expect(host.textContent).toContain('student.quiz.create.courseRequiredHint')
  expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(true)
  expect(mocks.create).not.toHaveBeenCalled()
  act(() => root.unmount())
})

it.each(['grouped', 'flat'] as const)('scrolls to the created quiz in %s view after closing the dialog', async (mode) => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', {
    configurable: true, value(this: HTMLDialogElement) { this.open = true },
  })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', {
    configurable: true, value(this: HTMLDialogElement) { this.open = false },
  })
  const scroll = vi.fn(function (this: HTMLElement) {
    expect(host.querySelector('dialog')!.open).toBe(false)
    expect(this.textContent).toContain('New quiz')
    expect(this.closest('[hidden]')).toBeNull()
  })
  const originalScroll = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'scrollIntoView')
  const originalMatchMedia = Object.getOwnPropertyDescriptor(window, 'matchMedia')
  Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', { configurable: true, value: scroll })
  Object.defineProperty(window, 'matchMedia', { configurable: true, value: () => ({ matches: mode === 'flat' }) })
  const base = { courseId: 7, courseTitle: 'Course A', difficulty: 'easy', answered: 0, questionCount: 2, creatorId: 1, creatorName: 'Student', creatorRole: 'ROLE_STUDENT', isSubmit: false }
  const oldQuiz = { ...base, id: 1, title: 'Old quiz', chapterKey: 'ch-1', chapterName: 'Chapter A', creationMethod: 'manual' }
  const newQuiz = mode === 'grouped'
    ? { ...base, id: 99, title: 'New quiz', chapterKey: 'ch-1', chapterName: 'Chapter A', creationMethod: 'ai' }
    : { ...base, id: 99, title: 'New quiz', creationMethod: 'ai' }
  mocks.questions.mockResolvedValueOnce([oldQuiz]).mockResolvedValue([oldQuiz, newQuiz])
  const detail = { tree: { label: 'Course A', children: [{ key: 'ch-1', label: 'Chapter A' }] } }
  let resolveRefreshedTree!: (value: typeof detail) => void
  const refreshedTree = new Promise<typeof detail>((resolve) => { resolveRefreshedTree = resolve })
  mocks.course.mockResolvedValueOnce(detail).mockResolvedValueOnce(detail)
    .mockImplementationOnce(() => refreshedTree).mockResolvedValue(detail)
  mocks.create.mockResolvedValue({ id: 99 })
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  try {
    await act(async () => root.render(<QuizDashboard />))
    if (mode === 'grouped') {
      const chapter = Array.from(host.querySelectorAll<HTMLButtonElement>('button[aria-expanded]')).find((button) => button.textContent?.includes('Chapter A'))!
      act(() => chapter.click())
      const course = host.querySelector<HTMLButtonElement>('h2 button[aria-expanded]')!
      act(() => course.click())
      expect(chapter.getAttribute('aria-expanded')).toBe('false')
      expect(course.getAttribute('aria-expanded')).toBe('false')
    } else {
      const select = host.querySelector<HTMLSelectElement>('select[aria-label="student.quiz.viewModeAria"]')!
      act(() => {
        select.value = 'flat'
        select.dispatchEvent(new Event('change', { bubbles: true }))
      })
    }
    const creationFilter = host.querySelector<HTMLSelectElement>('select[aria-label="student.quiz.creationFilterAria"]')!
    act(() => {
      creationFilter.value = 'manual'
      creationFilter.dispatchEvent(new Event('change', { bubbles: true }))
    })
    act(() => Array.from(host.querySelectorAll('button')).find((button) => button.textContent === 'student.quiz.create.button')!.click())
    const courseSelect = host.querySelector<HTMLSelectElement>('#quiz-create-course')!
    await act(async () => {
      courseSelect.value = '7'
      courseSelect.dispatchEvent(new Event('change', { bubbles: true }))
    })
    await act(async () => host.querySelector('dialog form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
    if (mode === 'grouped') expect(scroll).not.toHaveBeenCalled()
    await act(async () => resolveRefreshedTree(detail))
    expect(scroll).toHaveBeenCalledTimes(1)
    expect(scroll).toHaveBeenCalledWith({ behavior: mode === 'flat' ? 'instant' : 'smooth', block: 'center' })
    expect(creationFilter.value).toBe('all')
    expect(host.querySelector<HTMLSelectElement>('select[aria-label="student.quiz.viewModeAria"]')!.value).toBe(mode)
    if (mode === 'grouped') {
      expect(host.querySelector<HTMLButtonElement>('h2 button[aria-expanded]')!.getAttribute('aria-expanded')).toBe('true')
      expect(Array.from(host.querySelectorAll<HTMLButtonElement>('button[aria-expanded]')).find((button) => button.textContent?.includes('Chapter A'))!.getAttribute('aria-expanded')).toBe('true')
    }
  } finally {
    act(() => root.unmount())
    if (originalScroll) Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', originalScroll)
    else Reflect.deleteProperty(HTMLElement.prototype, 'scrollIntoView')
    if (originalMatchMedia) Object.defineProperty(window, 'matchMedia', originalMatchMedia)
    else Reflect.deleteProperty(window, 'matchMedia')
  }
})

it('keeps the dialog open on generation failure and prevents duplicate submissions', async () => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', {
    configurable: true, value(this: HTMLDialogElement) { this.open = true },
  })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', {
    configurable: true, value(this: HTMLDialogElement) { this.open = false },
  })
  mocks.questions.mockResolvedValue([])
  mocks.course.mockResolvedValue({ tree: { label: 'Course A', children: [] } })
  let rejectRequest!: (error: Error) => void
  mocks.create.mockImplementation(() => new Promise((_resolve, reject) => { rejectRequest = reject }))
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  await act(async () => root.render(<QuizDashboard />))
  act(() => Array.from(host.querySelectorAll('button')).find((button) => button.textContent === 'student.quiz.create.button')!.click())
  const select = host.querySelector<HTMLSelectElement>('#quiz-create-course')!
  await act(async () => {
    select.value = '7'
    select.dispatchEvent(new Event('change', { bubbles: true }))
  })
  const form = host.querySelector('dialog form')!
  const dialog = host.querySelector('dialog')!
  const cancel = Array.from(dialog.querySelectorAll('button')).find((button) => button.textContent === 'student.quiz.create.cancel')!
  expect(cancel.disabled).toBe(false)
  act(() => {
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
  })
  expect(mocks.create).toHaveBeenCalledTimes(1)
  expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(true)
  expect(cancel.disabled).toBe(true)
  act(() => cancel.click())
  expect(dialog.open).toBe(true)
  const cancelEvent = new Event('cancel', { bubbles: false, cancelable: true })
  act(() => dialog.dispatchEvent(cancelEvent))
  expect(cancelEvent.defaultPrevented).toBe(true)
  await act(async () => rejectRequest(new Error('AI generation failed')))
  expect(dialog.open).toBe(true)
  expect(host.textContent).toContain('AI generation failed')
  expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(false)
  expect(cancel.disabled).toBe(false)
  act(() => cancel.click())
  expect(dialog.open).toBe(false)
  act(() => root.unmount())
})

it('toggles each course independently and preserves chapter state', async () => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  mocks.questions.mockResolvedValue([
    { id: 1, title: 'Quiz A', courseId: 7, courseTitle: 'Course A', difficulty: 'easy', answered: 0, questionCount: 2, creatorId: 1, creatorName: 'Student', creatorRole: 'ROLE_STUDENT', isSubmit: false },
    { id: 2, title: 'Quiz B', courseId: 8, courseTitle: 'Course B', difficulty: 'easy', answered: 0, questionCount: 2, creatorId: 1, creatorName: 'Student', creatorRole: 'ROLE_STUDENT', isSubmit: false },
  ])
  mocks.course.mockResolvedValue({ tree: null })
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
  mocks.course.mockResolvedValue({ tree: null })
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

it('groups chapter quizzes separately from course quizzes and retains deleted chapters', async () => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  const base = { courseId: 7, courseTitle: 'Course A', difficulty: 'easy', answered: 0, questionCount: 2, creatorId: 1, creatorName: 'Student', creatorRole: 'ROLE_STUDENT', isSubmit: false, creationMethod: 'ai' }
  mocks.questions.mockResolvedValue([
    { ...base, id: 1, title: 'Chapter quiz', chapterKey: 'ch-1', chapterName: 'Chapter A' },
    { ...base, id: 2, title: 'Course quiz' },
    { ...base, id: 3, title: 'Old quiz', chapterKey: 'old', chapterName: 'Old chapter', chapterMissing: true },
  ])
  mocks.course.mockResolvedValue({ tree: { label: 'Course A', children: [{ key: 'ch-1', label: 'Chapter A' }] } })
  const host = document.createElement('div')
  document.body.appendChild(host)
  const root = createRoot(host)
  await act(async () => root.render(<QuizDashboard />))
  const panels = Array.from(host.querySelectorAll<HTMLButtonElement>('button[aria-expanded]'))
  const chapter = panels.find((button) => button.textContent?.includes('Chapter A'))!
  const unlinked = panels.find((button) => button.textContent?.includes('student.quiz.unlinkedNode'))!
  const deleted = panels.find((button) => button.textContent?.includes('Old chapter (deleted)'))!
  expect(chapter.parentElement?.textContent).toContain('Chapter quiz')
  expect(unlinked.parentElement?.textContent).toContain('Course quiz')
  expect(deleted.parentElement?.textContent).toContain('Old quiz')
  act(() => root.unmount())
})
