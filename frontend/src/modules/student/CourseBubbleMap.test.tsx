// @vitest-environment jsdom
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { CourseBubbleMap } from './CourseBubbleMap'
import { fitScale, mapBounds, placeBubbles } from './bubbleLayout'
import type { BubbleNode } from './bubbleTree'

vi.mock('@/core/i18n/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('CourseBubbleMap sizing', () => {
  it('updates the visible bubbles when expanding, resizing, and collapsing', () => {
    vi.useFakeTimers()
    Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
    let width = 900
    let height = 700
    let resize: ResizeObserverCallback = () => {}
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(() => width)
    vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockImplementation(() => height)
    vi.stubGlobal('ResizeObserver', class {
      constructor(callback: ResizeObserverCallback) { resize = callback }
      observe() {}
      disconnect() {}
    })
    vi.stubGlobal('requestAnimationFrame', () => 1)

    const tree: BubbleNode = {
      label: 'Root', note: '', children: Array.from({ length: 4 }, (_, i) => ({
        label: `Child ${i}`, note: '', children: [],
      })),
    }
    const host = document.createElement('div')
    document.body.appendChild(host)
    const root = createRoot(host)
    act(() => root.render(<CourseBubbleMap tree={tree} />))
    const rootButton = host.querySelector<HTMLButtonElement>('[aria-expanded]')
    if (!rootButton) throw new Error('Root bubble missing')
    expect(parseFloat(rootButton.style.width)).toBeCloseTo(210)

    act(() => rootButton.click())
    const bounds = mapBounds(placeBubbles(tree, new Set(['r']), 1))
    expect(host.querySelectorAll('[aria-label]').length).toBe(5)
    expect(parseFloat(rootButton.style.width)).toBeCloseTo(210 * fitScale(bounds, width, height))
    expect(rootButton.getAttribute('aria-expanded')).toBe('true')

    width = 500
    height = 400
    act(() => resize([], {} as ResizeObserver))
    expect(parseFloat(rootButton.style.width)).toBeCloseTo(210 * fitScale(bounds, width, height))

    act(() => rootButton.click())
    expect(host.querySelectorAll('[aria-label]').length).toBe(1)
    expect(parseFloat(rootButton.style.width)).toBeCloseTo(210 * fitScale({ width: 210, height: 210 }, width, height))
    act(() => root.unmount())
    host.remove()
  })
})
