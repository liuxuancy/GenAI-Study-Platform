import { describe, expect, it } from 'vitest'

import { bubbleLinks, fitScale, mapBounds, parentBubbleId, placeBubbles, type BubbleLink, type PlacedBubble } from './bubbleLayout'
import type { BubbleNode } from './bubbleTree'

function node(children: BubbleNode[] = []): BubbleNode {
  return { label: 'Node', note: '', children }
}

function segment(link: BubbleLink) {
  const radians = link.angle * Math.PI / 180
  return { x1: link.x, y1: link.y, x2: link.x + Math.cos(radians) * link.length, y2: link.y + Math.sin(radians) * link.length }
}

function distanceToSegment(bubble: PlacedBubble, link: BubbleLink) {
  const { x1, y1, x2, y2 } = segment(link)
  const dx = x2 - x1
  const dy = y2 - y1
  const fraction = Math.max(0, Math.min(1, ((bubble.x - x1) * dx + (bubble.y - y1) * dy) / (dx * dx + dy * dy)))
  return Math.hypot(bubble.x - (x1 + fraction * dx), bubble.y - (y1 + fraction * dy))
}

function intersect(a: BubbleLink, b: BubbleLink) {
  const first = segment(a)
  const second = segment(b)
  const cross = (x: number, y: number, u: number, v: number) => x * v - y * u
  const ax = first.x2 - first.x1
  const ay = first.y2 - first.y1
  const bx = second.x2 - second.x1
  const by = second.y2 - second.y1
  const dx = second.x1 - first.x1
  const dy = second.y1 - first.y1
  const denominator = cross(ax, ay, bx, by)
  if (Math.abs(denominator) < 1e-8) {
    if (Math.abs(cross(dx, dy, ax, ay)) >= 1e-6) return false
    const length = ax * ax + ay * ay
    const t1 = (dx * ax + dy * ay) / length
    const t2 = ((second.x2 - first.x1) * ax + (second.y2 - first.y1) * ay) / length
    return Math.min(1, Math.max(t1, t2)) - Math.max(0, Math.min(t1, t2)) > 1e-8
  }
  const t = cross(dx, dy, bx, by) / denominator
  const u = cross(dx, dy, ax, ay) / denominator
  return t > 1e-8 && t < 1 - 1e-8 && u > 1e-8 && u < 1 - 1e-8
}

describe('course bubble tree layout', () => {
  it('keeps the original upper-layer size without enlarging sparse maps', () => {
    const root = node([node(), node(), node(), node()])
    const collapsed = mapBounds(placeBubbles(root, new Set(), 1))
    const expanded = mapBounds(placeBubbles(root, new Set(['r']), 1))
    const collapsedScale = fitScale(collapsed, 900, 700)
    const expandedScale = fitScale(expanded, 900, 700)
    expect(collapsedScale).toBe(1)
    expect(expandedScale).toBe(1)
    expect(expanded.width * expandedScale).toBeLessThanOrEqual(900 - 36)
    expect(expanded.height * expandedScale).toBeLessThanOrEqual(700 - 56)
  })

  it('preserves readable minimum sizing and allows scrolling for dense maps', () => {
    const root = node(Array.from({ length: 18 }, () => node([node(), node(), node()])))
    const open = new Set(['r', ...Array.from({ length: 18 }, (_, i) => `r.${i}`)])
    const bounds = mapBounds(placeBubbles(root, open, 1))
    const scale = fitScale(bounds, 800, 600)
    expect(scale).toBe(0.8)
    expect(bounds.width * scale > 800 - 36 || bounds.height * scale > 600 - 56).toBe(true)
    expect(fitScale({ width: 210, height: 210 }, 0, 0)).toBe(0.8)
  })

  it.each([3, 4])('keeps %i third-layer bubbles large and doubles their connecting lines', (count) => {
    const root = node([node(Array.from({ length: count }, () => node()))])
    const bubbles = placeBubbles(root, new Set(['r', 'r.0']), 1)
    expect(bubbles[0].size).toBe(210)
    expect(bubbles[1].size).toBe(122)
    expect(bubbles.filter((bubble) => bubble.depth === 2).map((bubble) => bubble.size)).toEqual(Array(count).fill(114))
    const original = bubbleLinks(placeBubbles(root, new Set(['r', 'r.0']), 1, 1))
    for (const link of bubbleLinks(bubbles)) {
      expect(link.length).toBeCloseTo(original.find((item) => item.id === link.id)!.length * 2)
    }
  })

  it.each([0, 1, 2, 3, 4, 5])('fits the six-chapter Design layout when chapter %i is opened', (index) => {
    const root = node([2, 3, 2, 3, 2, 3].map((count) =>
      node(Array.from({ length: count }, () => node()))))
    const bubbles = placeBubbles(root, new Set(['r', `r.${index}`]), 1)
    const links = bubbleLinks(bubbles)
    const original = bubbleLinks(placeBubbles(root, new Set(['r', `r.${index}`]), 1, 1))
    for (const link of links) expect(link.length).toBeCloseTo(original.find((item) => item.id === link.id)!.length * 2)
    const scale = fitScale(mapBounds(bubbles), 800 - 64, 600 - 64)
    expect(scale).toBeGreaterThanOrEqual(0.8)
    for (const bubble of bubbles) {
      for (const other of bubbles) {
        if (bubble.id !== other.id) {
          expect(Math.hypot(bubble.x - other.x, bubble.y - other.y)).toBeGreaterThan((bubble.size + other.size) / 2)
        }
      }
      for (const link of links) {
        expect(distanceToSegment(bubble, link) + 1e-7).toBeGreaterThanOrEqual(bubble.size / 2)
      }
    }
  })

  it('keeps Design branches and lines separate for every combination of expanded chapters', () => {
    const root = node([2, 3, 2, 3, 2, 3].map((count) =>
      node(Array.from({ length: count }, () => node()))))
    for (let mask = 0; mask < 64; mask++) {
      const open = new Set(['r', ...Array.from({ length: 6 }, (_, index) => `r.${index}`).filter((_, index) => mask & (1 << index))])
      const bubbles = placeBubbles(root, open, 1)
      const links = bubbleLinks(bubbles)
      for (let index = 0; index < bubbles.length; index++) {
        const bubble = bubbles[index]
        for (const other of bubbles.slice(index + 1)) {
          expect(Math.hypot(bubble.x - other.x, bubble.y - other.y), `overlap with mask ${mask}`).toBeGreaterThan((bubble.size + other.size) / 2)
        }
        for (const link of links) {
          expect(distanceToSegment(bubble, link) + 1e-7, `line crosses bubble with mask ${mask}`).toBeGreaterThanOrEqual(bubble.size / 2)
        }
      }
      for (let index = 0; index < links.length; index++) {
        for (const other of links.slice(index + 1)) {
          expect(intersect(links[index], other), `intersecting lines with mask ${mask}`).toBe(false)
        }
      }
    }
  })

  it('avoids collisions in varied expanded trees after doubling links', () => {
    let seed = 126
    const random = () => {
      seed = (seed * 1664525 + 1013904223) >>> 0
      return seed / 2 ** 32
    }
    function tree(depth: number): BubbleNode {
      const count = depth === 0 ? 2 + Math.floor(random() * 5) : depth < 3 ? Math.floor(random() * 4) : 0
      return node(Array.from({ length: count }, () => tree(depth + 1)))
    }
    for (let sample = 0; sample < 100; sample++) {
      const root = tree(0)
      const open = new Set<string>()
      function expand(current: BubbleNode, id: string) {
        open.add(id)
        current.children?.forEach((child, index) => expand(child, `${id}.${index}`))
      }
      expand(root, 'r')
      const bubbles = placeBubbles(root, open, 1)
      const links = bubbleLinks(bubbles)
      for (let index = 0; index < bubbles.length; index++) {
        const bubble = bubbles[index]
        for (const other of bubbles.slice(index + 1)) {
          expect(Math.hypot(bubble.x - other.x, bubble.y - other.y), `overlap in sample ${sample}`).toBeGreaterThan((bubble.size + other.size) / 2)
        }
        for (const link of links) {
          expect(distanceToSegment(bubble, link) + 1e-7, `line crosses bubble in sample ${sample}`).toBeGreaterThanOrEqual(bubble.size / 2)
        }
      }
      for (let index = 0; index < links.length; index++) {
        for (const other of links.slice(index + 1)) {
          expect(intersect(links[index], other), `intersecting lines in sample ${sample}`).toBe(false)
        }
      }
    }
  })

  it('keeps nodes and unrelated links apart for expanded, uneven, large trees', () => {
    const root = node(Array.from({ length: 18 }, (_, i) =>
      node(Array.from({ length: i % 4 + 1 }, (_, j) =>
        node(Array.from({ length: (i + j) % 4 }, () => node()))))))
    const open = new Set<string>()
    function expand(current: BubbleNode, id: string) {
      if (!current.children?.length) return
      open.add(id)
      current.children.forEach((child, i) => expand(child, `${id}.${i}`))
    }
    expand(root, 'r')
    for (const opened of [new Set<string>(), new Set(['r']), new Set(['r', 'r.0', 'r.1', 'r.1.0']), open]) {
      const bubbles = placeBubbles(root, opened, 0.52)
      const links = bubbleLinks(bubbles)
      for (let i = 0; i < bubbles.length; i++) {
        const a = bubbles[i]
        for (const b of bubbles.slice(i + 1)) {
          expect(Math.hypot(a.x - b.x, a.y - b.y)).toBeGreaterThan((a.size + b.size) / 2)
        }
        for (const link of links) {
          expect(distanceToSegment(a, link) + 1e-7).toBeGreaterThanOrEqual(a.size / 2)
        }
      }
      for (let i = 0; i < links.length; i++) {
        const a = segment(links[i])
        for (const link of links.slice(i + 1)) {
          const b = segment(link)
          const cross = (x1: number, y1: number, x2: number, y2: number) => x1 * y2 - y1 * x2
          const ax = a.x2 - a.x1
          const ay = a.y2 - a.y1
          const bx = b.x2 - b.x1
          const by = b.y2 - b.y1
          const denominator = cross(ax, ay, bx, by)
          if (Math.abs(denominator) < 1e-8) {
            if (Math.abs(cross(b.x1 - a.x1, b.y1 - a.y1, ax, ay)) < 1e-6) {
              const project = (x: number, y: number) => ((x - a.x1) * ax + (y - a.y1) * ay) / (ax * ax + ay * ay)
              const start = project(b.x1, b.y1)
              const end = project(b.x2, b.y2)
              expect(Math.min(1, Math.max(start, end)) - Math.max(0, Math.min(start, end))).toBeLessThanOrEqual(1e-8)
            }
          } else {
            const dx = b.x1 - a.x1
            const dy = b.y1 - a.y1
            const t = cross(dx, dy, bx, by) / denominator
            const u = cross(dx, dy, ax, ay) / denominator
            expect(t > 1e-8 && t < 1 - 1e-8 && u > 1e-8 && u < 1 - 1e-8).toBe(false)
          }
        }
      }
      for (const parent of bubbles) {
        const children = bubbles.filter((item) => parentBubbleId(item.id) === parent.id)
        if (children.length < 2) continue
        const distances = children.map((item) => Math.hypot(item.x - parent.x, item.y - parent.y))
        expect(Math.max(...distances) - Math.min(...distances)).toBeLessThan(1e-6)
      }
      expect(mapBounds(bubbles).width).toBeGreaterThan(0)
    }
  })

  it('distributes root children around a circle, not in a row', () => {
    const bubbles = placeBubbles(node([node(), node(), node(), node()]), new Set(['r']), 1)
    const root = bubbles.find((item) => item.id === 'r')
    if (!root) throw new Error('root missing')
    const children = bubbles.filter((item) => parentBubbleId(item.id) === 'r')
    expect(children.filter((item) => item.x < root.x - 1)).toHaveLength(1)
    expect(children.filter((item) => item.x > root.x + 1)).toHaveLength(1)
    expect(children.filter((item) => item.y < root.y - 1)).toHaveLength(1)
    expect(children.filter((item) => item.y > root.y + 1)).toHaveLength(1)
  })

  it.each([1, 2, 3, 4, 6])('places the first root branch to the right with %i children', (count) => {
    const bubbles = placeBubbles(node(Array.from({ length: count }, () => node())), new Set(['r']), 1)
    const root = bubbles[0]
    const first = bubbles.find((bubble) => bubble.id === 'r.0')!
    expect(first.x).toBeGreaterThan(root.x)
    expect(first.y).toBeCloseTo(root.y)
    if (count % 2 === 0) {
      const opposite = bubbles.find((bubble) => bubble.id === `r.${count / 2}`)!
      expect(opposite.x).toBeLessThan(root.x)
      expect(opposite.y).toBeCloseTo(root.y)
    }
  })

  it('extends two-child branches mostly horizontally away from the root', () => {
    const root = node([node([node(), node()]), node([node(), node()])])
    const bubbles = placeBubbles(root, new Set(['r', 'r.0', 'r.1']), 1)
    for (const [parentId, direction] of [['r.0', 1], ['r.1', -1]] as const) {
      const parent = bubbles.find((bubble) => bubble.id === parentId)!
      const children = bubbles.filter((bubble) => parentBubbleId(bubble.id) === parentId)
      expect(children).toHaveLength(2)
      for (const child of children) {
        expect((child.x - parent.x) * direction).toBeGreaterThan(Math.abs(child.y - parent.y))
      }
    }
  })
})
