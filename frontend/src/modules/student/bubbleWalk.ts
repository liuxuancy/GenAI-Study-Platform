import { parentBubbleId, type BubbleLink, type PlacedBubble } from '@/modules/student/bubbleLayout'

export interface Point {
  x: number
  y: number
}

export interface WalkApproach {
  center: Point
  radius: number
  angle: number
  sweep: number
  length: number
  targetDistance: number
}

export interface WalkPlacement {
  point: Point
  distance: number
  approach: WalkApproach | null
}

export type WalkSegment =
  | { kind: 'line'; from: Point; to: Point; length: number; linkId: string }
  | { kind: 'arc'; center: Point; radius: number; startAngle: number; sweep: number; length: number }

export interface WalkRoute {
  start: Point
  segments: WalkSegment[]
  length: number
}

export function bubbleWalkRoute(bubbles: readonly PlacedBubble[], links: readonly BubbleLink[]): WalkRoute {
  const root = bubbles.find((bubble) => bubble.id === 'r')
  if (!root) return { start: { x: 0, y: 0 }, segments: [], length: 0 }

  const children = new Map<string, BubbleLink[]>()
  for (const link of links) {
    const parentId = parentBubbleId(link.id)
    if (!parentId) continue
    const siblings = children.get(parentId) ?? []
    siblings.push(link)
    children.set(parentId, siblings)
  }
  const byId = new Map(bubbles.map((bubble) => [bubble.id, bubble]))
  const first = children.get(root.id)?.[0]
  const start = first ? { x: first.x, y: first.y } : { x: root.x, y: root.y - root.size / 2 }
  const segments: WalkSegment[] = []
  let current = start

  function moveAround(bubble: PlacedBubble, destination: Point) {
    const startAngle = Math.atan2(current.y - bubble.y, current.x - bubble.x)
    const endAngle = Math.atan2(destination.y - bubble.y, destination.x - bubble.x)
    const sweep = Math.atan2(Math.sin(endAngle - startAngle), Math.cos(endAngle - startAngle))
    const radius = bubble.size / 2
    const length = Math.abs(sweep) * radius
    if (length > 1e-8) {
      segments.push({ kind: 'arc', center: { x: bubble.x, y: bubble.y }, radius, startAngle, sweep, length })
    }
    current = destination
  }

  function walk(bubble: PlacedBubble, returnPoint: Point) {
    for (const link of children.get(bubble.id) ?? []) {
      const child = byId.get(link.id)
      if (!child) continue
      const from = { x: link.x, y: link.y }
      const angle = link.angle * Math.PI / 180
      const to = { x: from.x + Math.cos(angle) * link.length, y: from.y + Math.sin(angle) * link.length }
      moveAround(bubble, from)
      segments.push({ kind: 'line', from, to, length: link.length, linkId: link.id })
      current = to
      walk(child, to)
      segments.push({ kind: 'line', from: to, to: from, length: link.length, linkId: link.id })
      current = from
    }
    moveAround(bubble, returnPoint)
  }

  walk(root, start)
  return { start, segments, length: segments.reduce((sum, segment) => sum + segment.length, 0) }
}

export function pointOnWalkRoute(route: WalkRoute, distance: number): Point {
  if (route.length <= 0) return route.start
  let remaining = ((distance % route.length) + route.length) % route.length
  for (const segment of route.segments) {
    if (remaining <= segment.length) {
      return segmentPoint(segment, remaining / segment.length)
    }
    remaining -= segment.length
  }
  return route.start
}

const fullTurn = Math.PI * 2
const geometryTolerance = 1e-6

function positiveAngle(angle: number) {
  return ((angle % fullTurn) + fullTurn) % fullTurn
}

function pointOnCircle(center: Point, radius: number, angle: number): Point {
  return { x: center.x + Math.cos(angle) * radius, y: center.y + Math.sin(angle) * radius }
}

function segmentPoint(segment: WalkSegment, fraction: number): Point {
  if (segment.kind === 'line') {
    return {
      x: segment.from.x + (segment.to.x - segment.from.x) * fraction,
      y: segment.from.y + (segment.to.y - segment.from.y) * fraction,
    }
  }
  return pointOnCircle(segment.center, segment.radius, segment.startAngle + segment.sweep * fraction)
}

function arcFraction(segment: Extract<WalkSegment, { kind: 'arc' }>, angle: number): number | null {
  const delta = segment.sweep > 0
    ? positiveAngle(angle - segment.startAngle)
    : positiveAngle(segment.startAngle - angle)
  return delta <= Math.abs(segment.sweep) + geometryTolerance ? Math.min(1, delta / Math.abs(segment.sweep)) : null
}

export function remapWalkPlacement(
  previousRoute: WalkRoute,
  previousBubbles: readonly PlacedBubble[],
  route: WalkRoute,
  bubbles: readonly PlacedBubble[],
  point: Point,
  direction: 1 | -1,
  previousDistance: number,
  onApproach: boolean,
): WalkPlacement {
  if (route.length <= 0) return { point: route.start, distance: 0, approach: null }

  let closest = Infinity
  let mappedPoint: Point | null = null
  let mappedLine: { linkId: string; fraction: number; occurrence: number } | null = null
  const normalizedDistance = previousRoute.length > 0
    ? ((previousDistance % previousRoute.length) + previousRoute.length) % previousRoute.length
    : 0
  let offset = 0
  const occurrences = new Map<string, number>()
  for (const segment of previousRoute.segments) {
    if (segment.kind === 'line' && segment.length > 0) {
      const occurrence = occurrences.get(segment.linkId) ?? 0
      occurrences.set(segment.linkId, occurrence + 1)
      const dx = segment.to.x - segment.from.x
      const dy = segment.to.y - segment.from.y
      const lengthSquared = dx * dx + dy * dy
      const fraction = Math.max(0, Math.min(1, ((point.x - segment.from.x) * dx + (point.y - segment.from.y) * dy) / lengthSquared))
      const oldPoint = segmentPoint(segment, fraction)
      const distance = Math.hypot(point.x - oldPoint.x, point.y - oldPoint.y)
      const onCurrentSegment = !onApproach && normalizedDistance >= offset - geometryTolerance &&
        normalizedDistance <= offset + segment.length + geometryTolerance
      if (distance < closest - geometryTolerance || (onCurrentSegment && distance <= closest + geometryTolerance)) {
        closest = distance
        mappedPoint = null
        mappedLine = { linkId: segment.linkId, fraction, occurrence }
      }
    }
    offset += segment.length
  }
  for (const oldBubble of previousBubbles) {
    const radius = oldBubble.size / 2
    if (radius <= 0) continue
    const angle = Math.atan2(point.y - oldBubble.y, point.x - oldBubble.x)
    const oldPoint = pointOnCircle(oldBubble, radius, angle)
    const distance = Math.hypot(point.x - oldPoint.x, point.y - oldPoint.y)
    const newBubble = bubbles.find((bubble) => bubble.id === oldBubble.id)
    const lineRemoved = mappedLine !== null && !route.segments.some(
      (segment) => segment.kind === 'line' && segment.linkId === mappedLine?.linkId,
    )
    if (distance < closest - geometryTolerance || (newBubble && lineRemoved && distance <= closest + geometryTolerance)) {
      closest = distance
      mappedPoint = newBubble ? pointOnCircle(newBubble, newBubble.size / 2, angle) : null
      mappedLine = null
    }
  }
  if (mappedLine) {
    let index = 0
    let newOffset = 0
    for (const segment of route.segments) {
      if (segment.kind === 'line' && segment.linkId === mappedLine.linkId) {
        if (index === mappedLine.occurrence) {
          const distance = newOffset + segment.length * mappedLine.fraction
          return { point: segmentPoint(segment, mappedLine.fraction), distance, approach: null }
        }
        index++
      }
      newOffset += segment.length
    }
  }
  return nearestWalkPlacement(route, bubbles, mappedPoint ?? point, direction)
}

export function nearestWalkPlacement(
  route: WalkRoute,
  bubbles: readonly PlacedBubble[],
  target: Point,
  direction: 1 | -1,
): WalkPlacement {
  if (route.length <= 0) return { point: route.start, distance: 0, approach: null }

  let bestDistanceSquared = Infinity
  let best: WalkPlacement | null = null
  let offset = 0
  for (const segment of route.segments) {
    if (segment.kind === 'line' && segment.length > 0) {
      const dx = segment.to.x - segment.from.x
      const dy = segment.to.y - segment.from.y
      const lengthSquared = dx * dx + dy * dy
      if (lengthSquared > 0) {
        const fraction = Math.max(0, Math.min(1, ((target.x - segment.from.x) * dx + (target.y - segment.from.y) * dy) / lengthSquared))
        const point = segmentPoint(segment, fraction)
        const distanceSquared = (point.x - target.x) ** 2 + (point.y - target.y) ** 2
        if (distanceSquared < bestDistanceSquared) {
          bestDistanceSquared = distanceSquared
          best = { point, distance: offset + segment.length * fraction, approach: null }
        }
      }
    }
    offset += segment.length
  }

  for (const bubble of bubbles) {
    const radius = bubble.size / 2
    if (radius <= 0) continue
    const center = { x: bubble.x, y: bubble.y }
    const angle = Math.atan2(target.y - center.y, target.x - center.x)
    const point = pointOnCircle(center, radius, angle)
    const distanceSquared = (point.x - target.x) ** 2 + (point.y - target.y) ** 2
    if (distanceSquared >= bestDistanceSquared) continue

    let routeDistance = 0
    let smallestSweep = Infinity
    offset = 0
    for (const segment of route.segments) {
      if (segment.kind === 'line' && segment.linkId !== bubble.id && parentBubbleId(segment.linkId) !== bubble.id) {
        offset += segment.length
        continue
      }
      if (segment.kind === 'arc' &&
          (Math.hypot(segment.center.x - center.x, segment.center.y - center.y) >= geometryTolerance ||
            Math.abs(segment.radius - radius) >= geometryTolerance)) {
        offset += segment.length
        continue
      }
      if (segment.kind === 'arc') {
        const fraction = arcFraction(segment, angle)
        if (fraction !== null) {
          smallestSweep = 0
          routeDistance = offset + segment.length * fraction
          break
        }
      }
      for (const fraction of [0, 1]) {
        const endpoint = segmentPoint(segment, fraction)
        if (Math.abs(Math.hypot(endpoint.x - center.x, endpoint.y - center.y) - radius) > geometryTolerance) continue
        const endpointAngle = Math.atan2(endpoint.y - center.y, endpoint.x - center.x)
        const sweep = direction === 1 ? positiveAngle(endpointAngle - angle) : positiveAngle(angle - endpointAngle)
        if (sweep < smallestSweep) {
          smallestSweep = sweep
          routeDistance = offset + segment.length * fraction
        }
      }
      offset += segment.length
    }
    if (!Number.isFinite(smallestSweep)) continue
    bestDistanceSquared = distanceSquared
    best = {
      point,
      distance: routeDistance,
      approach: smallestSweep < geometryTolerance ? null : {
        center, radius, angle, sweep: direction * smallestSweep,
        length: radius * smallestSweep, targetDistance: routeDistance,
      },
    }
  }
  return best ?? { point: route.start, distance: 0, approach: null }
}
