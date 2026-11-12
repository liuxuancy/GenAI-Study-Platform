// @vitest-environment jsdom
import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { BubbleWalker } from './BubbleWalker'
import { bubbleLinks, placeBubbles } from './bubbleLayout'
import { bubbleWalkRoute, nearestWalkPlacement, type Point } from './bubbleWalk'
import type { BubbleNode } from './bubbleTree'

vi.mock('@/core/i18n/useI18n', () => ({
  useI18n: () => ({ locale: 'en', t: () => 'Course character' }),
}))

const sprite = { palette: ['#000000'], pixels: ['0'] }
const dialogue = {
  zh: { touch: '不要碰我', rest: '休息一下' },
  en: { touch: 'Please stop', rest: 'I need a rest' },
}
const tree: BubbleNode = { label: 'Root', note: '', children: [{ label: 'Child', note: '', children: [] }] }
const bubbles = placeBubbles(tree, new Set(['r']), 1)
const route = bubbleWalkRoute(bubbles, bubbleLinks(bubbles))

let host: HTMLDivElement
let root: Root
let nextFrame: FrameRequestCallback | undefined
let reduced: boolean
let captured: number | null

function frame(time: number) {
  const callback = nextFrame
  nextFrame = undefined
  if (!callback) throw new Error('No animation frame scheduled')
  act(() => callback(time))
}

function pointer(button: HTMLButtonElement, type: string, x: number, y: number, pointerType = 'mouse') {
  const event = new MouseEvent(type, { bubbles: true, clientX: x, clientY: y, button: 0 })
  Object.defineProperties(event, {
    pointerId: { value: 1 }, isPrimary: { value: true }, pointerType: { value: pointerType },
  })
  act(() => button.dispatchEvent(event))
}

function click(button: HTMLButtonElement) {
  act(() => button.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 1 })))
}

function position(): Point {
  const transform = (host.querySelector('button')!.parentElement as HTMLElement).style.transform
  const match = transform.match(/translate\(([-.\d]+)px(?:, ([-.\d]+)px)?\)/)
  if (!match) throw new Error(`Unexpected transform: ${transform}`)
  return { x: Number(match[1]), y: Number(match[2] ?? 0) }
}

function expectPoint(actual: Point, expected: Point) {
  expect(actual.x).toBeCloseTo(expected.x, 2)
  expect(actual.y).toBeCloseTo(expected.y, 2)
}

function render(currentRoute = route, currentBubbles = bubbles, showHearts = false) {
  act(() => root.render(<BubbleWalker sprite={sprite} route={currentRoute} bubbles={currentBubbles} dialogue={dialogue} showHearts={showHearts} />))
  const button = host.querySelector('button')
  if (!button) throw new Error('Walker button not rendered')
  return button
}

beforeEach(() => {
  vi.spyOn(Math, 'random').mockReturnValue(0)
  reduced = false
  captured = null
  nextFrame = undefined
  Object.defineProperty(globalThis, 'IS_REACT_ACT_ENVIRONMENT', { configurable: true, value: true })
  vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => { nextFrame = callback; return 1 })
  vi.stubGlobal('cancelAnimationFrame', () => { nextFrame = undefined })
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: () => ({ matches: reduced, addEventListener: () => {}, removeEventListener: () => {} }),
  })
  Object.defineProperties(HTMLElement.prototype, {
    clientWidth: { configurable: true, get: () => 800 },
    clientHeight: { configurable: true, get: () => 600 },
    setPointerCapture: { configurable: true, value: (id: number) => { captured = id } },
    hasPointerCapture: { configurable: true, value: (id: number) => captured === id },
    releasePointerCapture: { configurable: true, value: () => { captured = null } },
  })
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    left: 100, top: 50, width: 800, height: 600, right: 900, bottom: 650, x: 100, y: 50, toJSON: () => ({}),
  })
  host = document.createElement('div')
  document.body.appendChild(host)
  root = createRoot(host)
})

afterEach(() => {
  act(() => root.unmount())
  host.remove()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('BubbleWalker pointer interaction', () => {
  it('shows a clear heart decoration only at maximum affection', () => {
    reduced = true
    render()
    expect(host.querySelector('svg[viewBox="0 0 64 48"]')).toBeNull()
    render(route, bubbles, true)
    const hearts = host.querySelector('svg[viewBox="0 0 64 48"]')
    expect(hearts).not.toBeNull()
    expect(hearts?.getAttribute('aria-hidden')).toBe('true')
    expect(hearts?.querySelectorAll('path')).toHaveLength(2)
    render()
    expect(host.querySelector('svg[viewBox="0 0 64 48"]')).toBeNull()
  })

  it('stays on the same root edge when the first branch expands', () => {
    reduced = true
    const collapsed = placeBubbles(tree, new Set(), 1)
    const collapsedRoute = bubbleWalkRoute(collapsed, [])
    render(collapsedRoute, collapsed)
    expectPoint(position(), collapsedRoute.start)
    render(route, bubbles)
    expectPoint(position(), { x: bubbles[0].x, y: bubbles[0].y - bubbles[0].size / 2 })
    expect(nextFrame).toBeUndefined()
  })

  it('keeps its relative root-edge position when expansion also changes the scale', () => {
    reduced = true
    const collapsed = placeBubbles(tree, new Set(), 1.8)
    const expanded = placeBubbles(tree, new Set(['r']), 1.2)
    render(bubbleWalkRoute(collapsed, []), collapsed)
    render(bubbleWalkRoute(expanded, bubbleLinks(expanded)), expanded)
    expectPoint(position(), { x: expanded[0].x, y: expanded[0].y - expanded[0].size / 2 })
  })

  it('walks smoothly from the preserved root position after expansion', () => {
    const collapsed = placeBubbles(tree, new Set(), 1)
    render(bubbleWalkRoute(collapsed, []), collapsed)
    render(route, bubbles)
    const start = position()
    expectPoint(start, { x: bubbles[0].x, y: bubbles[0].y - bubbles[0].size / 2 })
    frame(0)
    frame(100)
    const next = position()
    expect(Math.hypot(next.x - start.x, next.y - start.y)).toBeGreaterThan(1)
    expect(Math.hypot(next.x - start.x, next.y - start.y)).toBeLessThanOrEqual(1.81)
    expect(Math.hypot(next.x - bubbles[0].x, next.y - bubbles[0].y)).toBeCloseTo(bubbles[0].size / 2, 1)
  })

  it('drags with touch, snaps to the nearest visible edge, and does not count the drag as a click', () => {
    reduced = true
    const button = render()
    const start = position()
    const target = { x: bubbles[1].x + bubbles[1].size / 2 + 20, y: bubbles[1].y }
    pointer(button, 'pointerdown', 500 + start.x, 350 + start.y, 'touch')
    pointer(button, 'pointermove', 500 + target.x, 350 + target.y, 'touch')
    expectPoint(position(), target)
    pointer(button, 'pointerup', 500 + target.x, 350 + target.y, 'touch')
    expectPoint(position(), nearestWalkPlacement(route, bubbles, target, 1).point)
    click(button)
    click(button)
    click(button)
    expect(host.querySelector('[role="status"]')).toBeNull()
    click(button)
    expect(host.querySelector('[role="status"]')?.textContent).toBe('Please stop')
    expect(nextFrame).toBeUndefined()
  })

  it('returns to the root when no lines are expanded', () => {
    const collapsed = placeBubbles(tree, new Set(), 1)
    const collapsedRoute = bubbleWalkRoute(collapsed, [])
    const button = render(collapsedRoute, collapsed)
    const start = position()
    pointer(button, 'pointerdown', 500 + start.x, 350 + start.y)
    pointer(button, 'pointermove', 700, 500)
    pointer(button, 'pointerup', 700, 500)
    expectPoint(position(), collapsedRoute.start)
  })

  it('pauses rest while held and resumes walking only after the remaining rest', () => {
    const button = render()
    frame(0)
    for (let time = 1000; time <= 15000; time += 1000) frame(time)
    expect(host.querySelector('[role="status"]')?.textContent).toBe('I need a rest')
    const start = position()
    pointer(button, 'pointerdown', 500 + start.x, 350 + start.y)
    pointer(button, 'pointermove', 500 + start.x + 30, 350 + start.y)
    frame(16000)
    frame(17000)
    const target = position()
    pointer(button, 'pointerup', 500 + start.x + 30, 350 + start.y)
    const snapped = nearestWalkPlacement(route, bubbles, target, 1).point
    expectPoint(position(), snapped)
    frame(18000)
    frame(19000)
    expectPoint(position(), snapped)
    frame(20000)
    expectPoint(position(), snapped)
    frame(21000)
    expect(position()).not.toEqual(snapped)
  })

  it('snaps on cancellation and keeps an ordinary click working', () => {
    reduced = true
    const button = render()
    const start = position()
    pointer(button, 'pointerdown', 500 + start.x, 350 + start.y)
    pointer(button, 'pointermove', 700, 500)
    const target = position()
    pointer(button, 'pointercancel', 700, 500)
    expectPoint(position(), nearestWalkPlacement(route, bubbles, target, 1).point)
    for (let index = 0; index < 3; index++) {
      pointer(button, 'pointerdown', 600, 400)
      pointer(button, 'pointerup', 600, 400)
      click(button)
    }
    expect(host.querySelector('[role="status"]')?.textContent).toBe('Please stop')
  })

  it('keeps the pointer offset with scaled and scrolled canvas coordinates', () => {
    reduced = true
    let left = 100
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(() => ({
      left, top: 50, width: 1600, height: 1200, right: left + 1600, bottom: 1250, x: left, y: 50, toJSON: () => ({}),
    }))
    const button = render()
    const start = position()
    const startX = left + (400 + start.x) * 2 + 8
    const startY = 50 + (300 + start.y) * 2 - 12
    pointer(button, 'pointerdown', startX, startY)
    left = 40
    pointer(button, 'pointermove', startX + 40, startY + 20)
    const target = { x: start.x + 50, y: start.y + 10 }
    expectPoint(position(), target)
    pointer(button, 'pointerup', startX + 40, startY + 20)
    expectPoint(position(), nearestWalkPlacement(route, bubbles, target, 1).point)
  })

  it('cancels on escape and uses the latest route if the map collapses during a drag', () => {
    reduced = true
    const button = render()
    const start = position()
    pointer(button, 'pointerdown', 500 + start.x, 350 + start.y)
    pointer(button, 'pointermove', 700, 500)
    const target = position()
    act(() => window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })))
    expectPoint(position(), nearestWalkPlacement(route, bubbles, target, 1).point)
    click(button)
    expect(host.querySelector('[role="status"]')).toBeNull()

    const next = position()
    pointer(button, 'pointerdown', 500 + next.x, 350 + next.y)
    pointer(button, 'pointermove', 700, 500)
    const collapsed = placeBubbles(tree, new Set(), 1)
    const collapsedRoute = bubbleWalkRoute(collapsed, [])
    render(collapsedRoute, collapsed)
    expectPoint(position(), collapsedRoute.start)
    expect(captured).toBeNull()
  })
})
