// @vitest-environment jsdom
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { CourseCreationProvider } from './CourseCreationProvider'
import { MyCoursesPage } from './MyCoursesPage'

const mocks = vi.hoisted(() => ({ create: vi.fn(), list: vi.fn(), navigate: vi.fn(), error: vi.fn(), t: (key: string) => key }))
vi.mock('@/modules/student/api', () => ({ createStudentCourse: mocks.create, getStudentCourses: mocks.list }))
vi.mock('@/core/api/ai', () => ({ getAiProviderOptions: () => Promise.resolve({ default: 'deepseek', providers: { deepseek: true, teamily: false, mock: true } }) }))
vi.mock('@/core/auth/store', () => ({
  useAuthStore: Object.assign(
    (selector: (state: { currentUser: { id: number; major: string } }) => unknown) => selector({ currentUser: { id: 1, major: '' } }),
    { getState: () => ({ currentUser: { id: 1 } }) },
  ),
}))
vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: mocks.t }) }))
vi.mock('@/core/i18n/store', () => ({ getAppLocale: () => 'en' }))
vi.mock('@/core/i18n/translateApp', () => ({ translateApp: (_locale: string, key: string) => key }))
vi.mock('react-router-dom', () => ({
  useNavigate: () => mocks.navigate,
  useLocation: () => ({ pathname: '/student/courses' }),
  Link: ({ children, to, ...props }: { children: React.ReactNode; to: string }) => <a href={to} {...props}>{children}</a>,
}))
vi.mock('antd', () => ({ message: { error: mocks.error }, Select: () => null, Alert: () => null, Space: () => null, Typography: { Text: () => null } }))
vi.mock('@/modules/student/studentCourseVisuals', () => ({ CourseCover: () => <div /> }))

let host: HTMLDivElement
let root: Root

beforeEach(() => {
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', {
    configurable: true, value(this: HTMLDialogElement) { this.open = true },
  })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', {
    configurable: true, value(this: HTMLDialogElement) { this.open = false },
  })
  host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
  mocks.create.mockReset()
  mocks.list.mockReset()
  mocks.navigate.mockReset()
  const storage = new Map<string, string>()
  Object.defineProperty(window, 'localStorage', { configurable: true, value: {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value),
  } })
})

afterEach(() => {
  act(() => root.unmount())
  host.remove()
})

it('refreshes the course list after creation finishes with the dialog closed', async () => {
  const course = {
    id: 42, title: 'New course', description: '', courseType: 'structured',
    percent: 0, visibility: 'private', owned: true, chapterCount: 0, affection: 0, avatar: null,
  }
  let resolve!: (value: typeof course) => void
  mocks.create.mockReturnValue(new Promise((done) => { resolve = done }))
  mocks.list.mockResolvedValueOnce([]).mockResolvedValue([course])
  await act(async () => root.render(<CourseCreationProvider><MyCoursesPage /></CourseCreationProvider>))
  act(() => host.querySelector<HTMLButtonElement>('button')!.click())
  const input = host.querySelector<HTMLInputElement>('dialog input')!
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!
  act(() => {
    setter.call(input, 'New course')
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
  act(() => host.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
  act(() => host.querySelector<HTMLButtonElement>('dialog button[type="button"]')!.click())
  await act(async () => resolve(course))
  expect(mocks.navigate).not.toHaveBeenCalled()
  expect(mocks.list).toHaveBeenCalledTimes(2)
  expect(host.textContent).toContain('New course')
  expect(host.querySelector('a[href="/student/courses/42"]')).not.toBeNull()
})
