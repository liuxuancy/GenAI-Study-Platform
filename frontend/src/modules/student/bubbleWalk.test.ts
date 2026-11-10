import { describe, expect, it } from 'vitest'
import { bubbleLinks, placeBubbles } from '@/modules/student/bubbleLayout'
import type { BubbleNode } from '@/modules/student/bubbleTree'
import { bubbleWalkRoute, nearestWalkPlacement, pointOnWalkRoute, remapWalkPlacement, type WalkRoute } from '@/modules/student/bubbleWalk'

function node(children: BubbleNode[] = []): BubbleNode {
  return { label: 'Node', note: '', children }
}

function close(a: { x: number; y: number }, b: { x: number; y: number }) {
  expect(a.x).toBeCloseTo(b.x, 6)
  expect(a.y).toBeCloseTo(b.y, 6)
}

describe('bubble walking route', () => {
  it('rests on the root edge when there are no links', () => {
    const bubbles = placeBubbles(node(), new Set(), 1)
    const route = bubbleWalkRoute(bubbles, [])
    expect(route.segments).toEqual([])
    close(pointOnWalkRoute(route, 100), { x: bubbles[0].x, y: bubbles[0].y - bubbles[0].size / 2 })
  })

  it('walks out and back along a single line at a distance-based speed', () => {
    const bubbles = placeBubbles(node([node()]), new Set(['r']), 1)
    const links = bubbleLinks(bubbles)
    const route = bubbleWalkRoute(bubbles, links)
    expect(route.segments).toHaveLength(2)
    expect(route.length).toBeCloseTo(links[0].length * 2)
    const start = route.start
    const end = pointOnWalkRoute(route, links[0].length)
    close(pointOnWalkRoute(route, links[0].length / 2), {
      x: (start.x + end.x) / 2,
      y: (start.y + end.y) / 2,
    })
    close(pointOnWalkRoute(route, route.length + links[0].length / 2), pointOnWalkRoute(route, links[0].length / 2))
  })

  it('follows every expanded link in both directions and joins them along bubble edges', () => {
    const tree = node([node([node(), node()]), node(), node([node()])])
    for (const scale of [1, 0.52]) {
      const bubbles = placeBubbles(tree, new Set(['r', 'r.0', 'r.2']), scale)
      const links = bubbleLinks(bubbles)
      const route = bubbleWalkRoute(bubbles, links)
      let distance = 0
      for (const segment of route.segments) {
        const start = pointOnWalkRoute(route, distance)
        const end = pointOnWalkRoute(route, distance + segment.length)
        if (segment.kind === 'line') {
          close(start, segment.from)
          close(end, segment.to)
          close(pointOnWalkRoute(route, distance + segment.length / 2), {
            x: (segment.from.x + segment.to.x) / 2,
            y: (segment.from.y + segment.to.y) / 2,
          })
        } else {
          expect(Math.hypot(start.x - segment.center.x, start.y - segment.center.y)).toBeCloseTo(segment.radius, 6)
          expect(Math.hypot(end.x - segment.center.x, end.y - segment.center.y)).toBeCloseTo(segment.radius, 6)
          const middle = pointOnWalkRoute(route, distance + segment.length / 2)
          expect(Math.hypot(middle.x - segment.center.x, middle.y - segment.center.y)).toBeCloseTo(segment.radius, 6)
        }
        distance += segment.length
      }
      for (const link of links) {
        const paths = route.segments.filter((segment) => segment.kind === 'line' && segment.linkId === link.id)
        expect(paths).toHaveLength(2)
        if (paths[0].kind !== 'line' || paths[1].kind !== 'line') throw new Error('Expected line segments')
        close(paths[0].from, paths[1].to)
        close(paths[0].to, paths[1].from)
      }
      close(pointOnWalkRoute(route, route.length), route.start)
    }
  })

  it('rebuilds the route when branches are collapsed', () => {
    const tree = node([node([node()])])
    const expanded = placeBubbles(tree, new Set(['r', 'r.0']), 1)
    const collapsed = placeBubbles(tree, new Set(['r']), 1)
    expect(bubbleWalkRoute(expanded, bubbleLinks(expanded)).segments.filter((s) => s.kind === 'line')).toHaveLength(4)
    expect(bubbleWalkRoute(collapsed, bubbleLinks(collapsed)).segments.filter((s) => s.kind === 'line')).toHaveLength(2)
  })

  it('projects onto a line and clamps beyond its endpoints', () => {
    const bubbles = placeBubbles(node([node()]), new Set(['r']), 1)
    const route = bubbleWalkRoute(bubbles, bubbleLinks(bubbles))
    const line = route.segments[0]
    if (line.kind !== 'line') throw new Error('Expected line')
    const middle = { x: (line.from.x + line.to.x) / 2, y: (line.from.y + line.to.y) / 2 }
    const placement = nearestWalkPlacement(route, bubbles, { x: middle.x, y: middle.y + 3 }, 1)
    close(placement.point, middle)
    close(pointOnWalkRoute(route, placement.distance), placement.point)
    expect(placement.approach).toBeNull()
    const endpoint = nearestWalkPlacement(route, [], { x: line.from.x - 100, y: line.from.y }, 1)
    close(endpoint.point, line.from)
  })

  it('keeps the root edge angle when expanding changes the canvas origin', () => {
    const tree = node([node()])
    const oldBubbles = placeBubbles(tree, new Set(), 1)
    const oldRoute = bubbleWalkRoute(oldBubbles, [])
    const newBubbles = placeBubbles(tree, new Set(['r']), 1)
    const newRoute = bubbleWalkRoute(newBubbles, bubbleLinks(newBubbles))
    const placement = remapWalkPlacement(oldRoute, oldBubbles, newRoute, newBubbles, oldRoute.start, 1, 0, false)
    close(placement.point, { x: newBubbles[0].x, y: newBubbles[0].y - newBubbles[0].size / 2 })
    expect(placement.approach).not.toBeNull()
  })

  it('preserves the link fraction and return direction after expanding or resizing', () => {
    const tree = node([node([node()]), node()])
    const oldBubbles = placeBubbles(tree, new Set(['r']), 1)
    const oldRoute = bubbleWalkRoute(oldBubbles, bubbleLinks(oldBubbles))
    const newBubbles = placeBubbles(tree, new Set(['r', 'r.0']), 0.7)
    const newRoute = bubbleWalkRoute(newBubbles, bubbleLinks(newBubbles))
    const oldSegments = oldRoute.segments.filter((segment) => segment.kind === 'line' && segment.linkId === 'r.0')
    const returning = oldSegments[1]
    if (!returning || returning.kind !== 'line') throw new Error('Expected returning line')
    const offset = oldRoute.segments.slice(0, oldRoute.segments.indexOf(returning)).reduce((sum, segment) => sum + segment.length, 0)
    const oldDistance = offset + returning.length * 0.3
    const point = pointOnWalkRoute(oldRoute, oldDistance)
    const placement = remapWalkPlacement(oldRoute, oldBubbles, newRoute, newBubbles, point, 1, oldDistance, false)
    const newLines = newRoute.segments.filter((segment) => segment.kind === 'line' && segment.linkId === 'r.0')
    const newReturn = newLines[1]
    if (!newReturn || newReturn.kind !== 'line') throw new Error('Expected returning line')
    close(placement.point, {
      x: newReturn.from.x + (newReturn.to.x - newReturn.from.x) * 0.3,
      y: newReturn.from.y + (newReturn.to.y - newReturn.from.y) * 0.3,
    })
    close(pointOnWalkRoute(newRoute, placement.distance), placement.point)
    const next = pointOnWalkRoute(newRoute, placement.distance + 1)
    expect(Math.hypot(next.x - newReturn.to.x, next.y - newReturn.to.y))
      .toBeLessThan(Math.hypot(placement.point.x - newReturn.to.x, placement.point.y - newReturn.to.y))
  })

  it('keeps the surviving bubble attachment when its line is collapsed', () => {
    const tree = node([node([node()]), node()])
    const oldBubbles = placeBubbles(tree, new Set(['r', 'r.0']), 1)
    const oldRoute = bubbleWalkRoute(oldBubbles, bubbleLinks(oldBubbles))
    const newBubbles = placeBubbles(tree, new Set(['r']), 1)
    const newRoute = bubbleWalkRoute(newBubbles, bubbleLinks(newBubbles))
    const removed = oldRoute.segments.find((segment) => segment.kind === 'line' && segment.linkId === 'r.0.0')
    if (!removed || removed.kind !== 'line') throw new Error('Expected removed line')
    const oldParent = oldBubbles.find((bubble) => bubble.id === 'r.0')!
    const newParent = newBubbles.find((bubble) => bubble.id === 'r.0')!
    const angle = Math.atan2(removed.from.y - oldParent.y, removed.from.x - oldParent.x)
    const placement = remapWalkPlacement(oldRoute, oldBubbles, newRoute, newBubbles, removed.from, 1, 0, false)
    close(placement.point, {
      x: newParent.x + Math.cos(angle) * newParent.size / 2,
      y: newParent.y + Math.sin(angle) * newParent.size / 2,
    })
  })

  it('projects onto the full bubble edge and approaches an attachment in either direction', () => {
    const bubbles = placeBubbles(node([node()]), new Set(['r']), 1)
    const route = bubbleWalkRoute(bubbles, bubbleLinks(bubbles))
    const child = bubbles[1]
    const target = { x: child.x + child.size / 2, y: child.y }
    for (const direction of [1, -1] as const) {
      const placement = nearestWalkPlacement(route, bubbles, target, direction)
      close(placement.point, target)
      expect(placement.approach).not.toBeNull()
      const arc = placement.approach!
      expect(Math.sign(arc.sweep)).toBe(direction)
      close({
        x: arc.center.x + Math.cos(arc.angle + arc.sweep) * arc.radius,
        y: arc.center.y + Math.sin(arc.angle + arc.sweep) * arc.radius,
      }, pointOnWalkRoute(route, arc.targetDistance))
    }
  })

  it('uses an existing arc directly and handles wraparound and the circle center', () => {
    const bubbles = placeBubbles(node([node(), node(), node()]), new Set(['r']), 1)
    const route = bubbleWalkRoute(bubbles, bubbleLinks(bubbles))
    const arcIndex = route.segments.findIndex((segment) => segment.kind === 'arc')
    const arc = route.segments[arcIndex]
    if (!arc || arc.kind !== 'arc') throw new Error('Expected arc')
    const angle = arc.startAngle + arc.sweep / 2
    const target = { x: arc.center.x + arc.radius * Math.cos(angle), y: arc.center.y + arc.radius * Math.sin(angle) }
    const placement = nearestWalkPlacement(route, bubbles, target, -1)
    close(placement.point, target)
    expect(placement.approach).toBeNull()
    close(pointOnWalkRoute(route, placement.distance), target)
    const root = bubbles[0]
    const center = nearestWalkPlacement(route, bubbles, { x: root.x, y: root.y }, 1)
    expect(Number.isFinite(center.point.x)).toBe(true)
    expect(Number.isFinite(center.point.y)).toBe(true)
  })

  it('projects onto an arc that crosses the negative pi boundary', () => {
    const root = { id: 'r', depth: 0, node: node(), x: 0, y: 0, size: 200 }
    const startAngle = 3 * Math.PI / 4
    const sweep = Math.PI / 2
    const wrappedRoute: WalkRoute = {
      start: { x: 100 * Math.cos(startAngle), y: 100 * Math.sin(startAngle) },
      segments: [{ kind: 'arc', center: { x: 0, y: 0 }, radius: 100, startAngle, sweep, length: 100 * sweep }],
      length: 100 * sweep,
    }
    const angle = -Math.PI + 0.1
    const target = { x: 100 * Math.cos(angle), y: 100 * Math.sin(angle) }
    const placement = nearestWalkPlacement(wrappedRoute, [root], target, 1)
    expect(placement.approach).toBeNull()
    close(pointOnWalkRoute(wrappedRoute, placement.distance), target)
  })

  it('returns to the root start when there is no expanded line', () => {
    const bubbles = placeBubbles(node(), new Set(), 1)
    const route: WalkRoute = bubbleWalkRoute(bubbles, [])
    expect(nearestWalkPlacement(route, bubbles, { x: 10000, y: -10000 }, 1)).toEqual({
      point: route.start, distance: 0, approach: null,
    })
  })
})
