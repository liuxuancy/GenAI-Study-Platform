// @vitest-environment jsdom
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { AiGenerationMeta } from '@/core/api/ai'
import { GenerationMetaAlert } from './GenerationMetaAlert'

vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('antd', () => ({
  Alert: ({ type, description, closable, onClose }: {
    type: string
    description: string
    closable: boolean
    onClose: () => void
  }) => <div data-type={type}>{description}{closable && <button onClick={onClose}>close</button>}</div>,
}))

const success: AiGenerationMeta = {
  scene: 'course', templateId: 'v1', templateVersion: '1', modelRequested: '', modelUsed: '',
  fallbackUsed: false, latencyMs: 100, warnings: [],
}
let host: HTMLDivElement
let root: Root

beforeEach(() => {
  vi.useFakeTimers()
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
})

afterEach(() => {
  act(() => root.unmount())
  host.remove()
  vi.useRealTimers()
})

function render(meta: AiGenerationMeta) {
  act(() => root.render(<GenerationMetaAlert meta={meta} />))
}

describe('course generation notification', () => {
  it('disappears exactly two seconds after success', () => {
    render(success)
    expect(host.querySelector('[data-type="success"]')).not.toBeNull()
    act(() => vi.advanceTimersByTime(1999))
    expect(host.querySelector('[data-type="success"]')).not.toBeNull()
    act(() => vi.advanceTimersByTime(1))
    expect(host.textContent).toBe('')
  })

  it.each([
    { ...success, warnings: ['warning'] },
    { ...success, fallbackUsed: true },
  ])('keeps warning or fallback information until closed', (meta) => {
    render(meta)
    act(() => vi.advanceTimersByTime(10000))
    expect(host.querySelector('[data-type="warning"]')).not.toBeNull()
    act(() => host.querySelector('button')!.click())
    expect(host.textContent).toBe('')
  })

  it('restarts the timer for each new successful generation', () => {
    render(success)
    act(() => vi.advanceTimersByTime(1500))
    render({ ...success })
    act(() => vi.advanceTimersByTime(1999))
    expect(host.querySelector('[data-type="success"]')).not.toBeNull()
    act(() => vi.advanceTimersByTime(1))
    expect(host.textContent).toBe('')
  })
})
