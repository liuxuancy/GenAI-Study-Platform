// @vitest-environment jsdom
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { CourseCreationProvider, useCourseCreation } from './CourseCreationProvider'

const mocks = vi.hoisted(() => ({ create: vi.fn(), navigate: vi.fn(), error: vi.fn(), providers: vi.fn(), pathname: '/student', t: (key: string) => key }))
vi.mock('@/modules/student/api', () => ({ createStudentCourse: mocks.create }))
vi.mock('@/core/api/ai', () => ({ getAiProviderOptions: mocks.providers }))
vi.mock('@/core/auth/store', () => {
  const store = Object.assign(
    (selector: (state: { currentUser: { id: number; major: string } }) => unknown) => selector({ currentUser: { id: 1, major: '' } }),
    { getState: () => ({ currentUser: { id: 1 } }) },
  )
  return { useAuthStore: store }
})
vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: mocks.t }) }))
vi.mock('react-router-dom', () => ({
  useNavigate: () => mocks.navigate,
  useLocation: () => ({ pathname: mocks.pathname }),
}))
vi.mock('antd', () => ({ message: { error: mocks.error }, Select: () => null, Alert: () => null, Space: () => null, Typography: { Text: () => null } }))

let host: HTMLDivElement
let root: Root

function Page() {
  const { openDialog, revision, pendingTitle } = useCourseCreation()
  return <><button type="button" onClick={openDialog}>open</button><span data-testid="revision">{revision}</span><span data-testid="pending">{pendingTitle}</span></>
}

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
  mocks.navigate.mockReset()
  mocks.error.mockReset()
  mocks.providers.mockReset().mockResolvedValue({ default: 'deepseek', providers: { deepseek: true, teamily: true, mock: true } })
  const storage = new Map<string, string>()
  Object.defineProperty(window, 'localStorage', { configurable: true, value: {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value),
  } })
  mocks.pathname = '/student'
})

afterEach(() => {
  act(() => root.unmount())
  host.remove()
})

async function render() {
  await act(async () => root.render(<CourseCreationProvider><Page /></CourseCreationProvider>))
}

function open() {
  act(() => host.querySelector<HTMLButtonElement>('button')!.click())
}

function enter(index: number, value: string) {
  const input = host.querySelectorAll('dialog input')[index]
  const setValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!
  act(() => {
    setValue.call(input, value)
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
}

function submit() {
  act(() => host.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })))
}

function close() {
  act(() => host.querySelector<HTMLButtonElement>('dialog button[type="button"]')!.click())
}

function deferred() {
  let resolve!: (value: { id: number }) => void
  let reject!: (reason: Error) => void
  const promise = new Promise((ok, fail) => { resolve = ok; reject = fail })
  mocks.create.mockReturnValue(promise)
  return { resolve, reject }
}

describe('background course creation', () => {
  it('sends the selected model and keeps it locked while the request runs', async () => {
    const request = deferred()
    await render()
    open()
    enter(0, 'Test course')
    const select = host.querySelectorAll<HTMLSelectElement>('dialog select')[1]
    act(() => {
      select.value = 'teamily'
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
    submit()
    expect(mocks.create).toHaveBeenCalledWith({ title: 'Test course', description: '', provider: 'teamily' })
    expect(select.disabled).toBe(true)
    close()
    open()
    expect(select.value).toBe('teamily')
    expect(select.disabled).toBe(true)
    await act(async () => request.reject(new Error('failed')))
    expect(select.disabled).toBe(false)
    expect(Array.from(select.options, (option) => option.value)).toEqual(['deepseek', 'teamily'])
    act(() => {
      select.value = 'deepseek'
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
    mocks.create.mockResolvedValue({ id: 2 })
    await act(async () => submit())
    expect(mocks.create).toHaveBeenLastCalledWith({ title: 'Test course', description: '', provider: 'deepseek' })
  })

  it('hides mock and blocks creation when no real model is available', async () => {
    mocks.providers.mockResolvedValue({ default: 'deepseek', providers: { deepseek: false, teamily: false, mock: true } })
    window.localStorage.setItem('caveclass.ai.courseAvatarProvider', 'mock')
    await render()
    open()
    enter(0, 'Test course')
    const select = host.querySelectorAll<HTMLSelectElement>('dialog select')[1]
    expect(Array.from(select.options, (option) => option.value)).toEqual(['deepseek', 'teamily'])
    expect(select.disabled).toBe(true)
    expect(host.textContent).toContain('ai.providerNotConfigured')
    expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(true)
    await act(async () => submit())
    expect(mocks.create).not.toHaveBeenCalled()
  })

  it('ignores a previously saved mock choice when a real model is available', async () => {
    window.localStorage.setItem('caveclass.ai.courseAvatarProvider', 'mock')
    mocks.providers.mockResolvedValue({ default: 'teamily', providers: { deepseek: false, teamily: true, mock: true } })
    mocks.create.mockResolvedValue({ id: 1 })
    await render()
    open()
    enter(0, 'Test course')
    const select = host.querySelectorAll<HTMLSelectElement>('dialog select')[1]
    expect(select.value).toBe('teamily')
    expect(Array.from(select.options, (option) => option.value)).toEqual(['deepseek', 'teamily'])
    await act(async () => submit())
    expect(mocks.create).toHaveBeenCalledWith({ title: 'Test course', description: '', provider: 'teamily' })
  })

  it('ignores a mock default returned by an older server', async () => {
    mocks.providers.mockResolvedValue({ default: 'mock', providers: { deepseek: false, teamily: true, mock: true } })
    await render()
    open()
    enter(0, 'Test course')
    const select = host.querySelectorAll<HTMLSelectElement>('dialog select')[1]
    expect(select.value).toBe('teamily')
    expect(Array.from(select.options, (option) => option.value)).toEqual(['deepseek', 'teamily'])
    mocks.create.mockResolvedValue({ id: 1 })
    await act(async () => submit())
    expect(mocks.create).toHaveBeenCalledWith({ title: 'Test course', description: '', provider: 'teamily' })
  })

  it('preserves a previously selected real model for course creation', async () => {
    window.localStorage.setItem('caveclass.ai.courseAvatarProvider', 'teamily')
    await render()
    open()
    expect(host.querySelectorAll<HTMLSelectElement>('dialog select')[1].value).toBe('teamily')
  })

  it('blocks submission while options load or no model is configured', async () => {
    let resolve!: (value: unknown) => void
    mocks.providers.mockReturnValue(new Promise((done) => { resolve = done }))
    await render()
    open()
    enter(0, 'Test course')
    expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(true)
    expect(host.textContent).toContain('common.loading')
    submit()
    expect(mocks.create).not.toHaveBeenCalled()
    await act(async () => resolve({ default: 'deepseek', providers: { deepseek: false, teamily: false, mock: false } }))
    expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(true)
    expect(host.textContent).toContain('ai.providerNotConfigured')
    submit()
    expect(mocks.create).not.toHaveBeenCalled()
  })

  it('shows provider loading errors and blocks submission', async () => {
    mocks.providers.mockRejectedValue(new Error('provider request failed'))
    await render()
    open()
    expect(host.textContent).toContain('provider request failed')
    expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(true)
  })

  it('continues after closing, retains the form and silently refreshes on success', async () => {
    const request = deferred()
    await render()
    open()
    enter(0, 'Test course')
    enter(1, 'Description')
    submit()
    expect(mocks.create).toHaveBeenCalledWith({ title: 'Test course', description: 'Description', provider: 'deepseek' })
    close()
    open()
    expect(host.querySelectorAll<HTMLInputElement>('dialog input')[0].value).toBe('Test course')
    expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(true)
    submit()
    expect(mocks.create).toHaveBeenCalledOnce()
    close()
    await act(async () => request.resolve({ id: 1 }))
    expect(mocks.navigate).not.toHaveBeenCalled()
    expect(host.querySelector('[data-testid="revision"]')!.textContent).toBe('1')
    expect(host.querySelector('[data-testid="pending"]')!.textContent).toBe('')
    open()
    expect(host.querySelectorAll<HTMLInputElement>('dialog input')[0].value).toBe('')
  })

  it('navigates when the reopened dialog is still open at completion', async () => {
    const request = deferred()
    await render()
    open()
    enter(0, 'Test course')
    submit()
    close()
    open()
    await act(async () => request.resolve({ id: 42 }))
    expect(mocks.navigate).toHaveBeenCalledWith('/student/courses/42')
    expect(host.querySelector('dialog')!.open).toBe(false)
  })

  it('notifies about a closed-dialog failure and keeps the inputs available for retry', async () => {
    const request = deferred()
    await render()
    open()
    enter(0, 'Test course')
    enter(1, 'Description')
    submit()
    close()
    await act(async () => request.reject(new Error('model failed')))
    expect(mocks.error).toHaveBeenCalledWith('student.createCourse.failed: model failed')
    open()
    expect(host.querySelectorAll<HTMLInputElement>('dialog input')[0].value).toBe('Test course')
    expect(host.querySelectorAll<HTMLInputElement>('dialog input')[1].value).toBe('Description')
    expect(host.textContent).toContain('model failed')
    expect(host.querySelector<HTMLButtonElement>('dialog button[type="submit"]')!.disabled).toBe(false)
    mocks.create.mockResolvedValue({ id: 2 })
    await act(async () => submit())
    expect(mocks.create).toHaveBeenCalledTimes(2)
    expect(mocks.navigate).toHaveBeenCalledWith('/student/courses/2')
  })

  it('closes on route changes while keeping the request running', async () => {
    const request = deferred()
    await render()
    open()
    enter(0, 'Test course')
    submit()
    mocks.pathname = '/student/courses'
    await render()
    expect(host.querySelector('dialog')!.open).toBe(false)
    await act(async () => request.resolve({ id: 3 }))
    expect(mocks.navigate).not.toHaveBeenCalled()
    expect(host.querySelector('[data-testid="revision"]')!.textContent).toBe('1')
  })

  it('treats Escape as closing without cancelling the request', async () => {
    const request = deferred()
    await render()
    open()
    enter(0, 'Test course')
    submit()
    act(() => host.querySelector('dialog')!.dispatchEvent(new Event('cancel', { cancelable: true })))
    expect(host.querySelector('dialog')!.open).toBe(false)
    await act(async () => request.resolve({ id: 4 }))
    expect(mocks.navigate).not.toHaveBeenCalled()
  })
})
