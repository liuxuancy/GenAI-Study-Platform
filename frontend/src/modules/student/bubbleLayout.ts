// 子节点位于父节点周围的圆弧上，按实际气泡和线段避让。

import type { BubbleNode } from '@/modules/student/bubbleTree'

const bubbleSize = [210, 122, 114, 78]
const branchHalfAngle = Math.PI * 2 / 9
const wideBranchHalfAngle = Math.PI / 3
const spacing = 24
const minScale = 0.8
const maxScale = 1
const linkLengthFactor = 2

interface Segment {
  x1: number
  y1: number
  x2: number
  y2: number
}

interface Branch {
  bubbles: PlacedBubble[]
  links: Segment[]
  connection: Segment
}

function pointSegmentDistance(x: number, y: number, line: Segment): number {
  const dx = line.x2 - line.x1
  const dy = line.y2 - line.y1
  const lengthSquared = dx * dx + dy * dy
  const t = lengthSquared === 0 ? 0 : Math.max(0, Math.min(1, ((x - line.x1) * dx + (y - line.y1) * dy) / lengthSquared))
  return Math.hypot(x - line.x1 - t * dx, y - line.y1 - t * dy)
}

function linesIntersect(a: Segment, b: Segment): boolean {
  const cross = (x: number, y: number, u: number, v: number) => x * v - y * u
  const ax = a.x2 - a.x1
  const ay = a.y2 - a.y1
  const bx = b.x2 - b.x1
  const by = b.y2 - b.y1
  const denominator = cross(ax, ay, bx, by)
  const dx = b.x1 - a.x1
  const dy = b.y1 - a.y1
  if (Math.abs(denominator) < 1e-8) {
    if (Math.abs(cross(dx, dy, ax, ay)) >= 1e-8) return false
    const lengthSquared = ax * ax + ay * ay
    if (lengthSquared === 0) return pointSegmentDistance(a.x1, a.y1, b) < 1e-8
    const t1 = (dx * ax + dy * ay) / lengthSquared
    const t2 = ((b.x2 - a.x1) * ax + (b.y2 - a.y1) * ay) / lengthSquared
    return Math.min(1, Math.max(t1, t2)) >= Math.max(0, Math.min(t1, t2))
  }
  const t = cross(dx, dy, bx, by) / denominator
  const u = cross(dx, dy, ax, ay) / denominator
  return t >= 0 && t <= 1 && u >= 0 && u <= 1
}

function clearOfBubble(line: Segment, bubble: PlacedBubble): boolean {
  return pointSegmentDistance(bubble.x, bubble.y, line) >= bubble.size / 2 + 2
}

function branchesFit(parent: PlacedBubble, branches: Branch[]): boolean {
  for (let i = 0; i < branches.length; i++) {
    const branch = branches[i]
    for (const bubble of branch.bubbles) {
      if (Math.hypot(bubble.x - parent.x, bubble.y - parent.y) < (bubble.size + parent.size) / 2 + 8) return false
    }
    if (branch.links.some((line) => !clearOfBubble(line, parent))) return false
    if (branch.bubbles.slice(1).some((bubble) => !clearOfBubble(branch.connection, bubble))) return false
    if (branch.links.some((line) => linesIntersect(line, branch.connection))) return false
    for (const other of branches.slice(0, i)) {
      const branchLines = [branch.connection, ...branch.links]
      const otherLines = [other.connection, ...other.links]
      for (const bubble of branch.bubbles) {
        if (other.bubbles.some((item) => Math.hypot(item.x - bubble.x, item.y - bubble.y) < (item.size + bubble.size) / 2 + 8)) return false
        if (otherLines.some((line) => !clearOfBubble(line, bubble))) return false
      }
      if (other.bubbles.some((bubble) => branchLines.some((line) => !clearOfBubble(line, bubble)))) return false
      if (branchLines.some((line) => otherLines.some((otherLine) => linesIntersect(line, otherLine)))) return false
    }
  }
  return true
}

export interface PlacedBubble {
  id: string
  depth: number
  node: BubbleNode
  x: number
  y: number
  size: number
}

export interface BubbleLink {
  id: string
  x: number
  y: number
  length: number
  angle: number
}

export function placeBubbles(root: BubbleNode, openIds: ReadonlySet<string>, scale: number, linkFactor = linkLengthFactor): PlacedBubble[] {
  function walk(node: BubbleNode, id: string, depth: number): PlacedBubble[] {
    const size = bubbleSize[Math.min(depth, bubbleSize.length - 1)]
    const parent: PlacedBubble = { id, node, depth, x: 0, y: 0, size }
    const children = openIds.has(id) ? node.children ?? [] : []
    if (!children.length) return [parent]

    const subtrees = children.map((child, index) => walk(child, `${id}.${index}`, depth + 1))
    const count = subtrees.length
    const halfAngle = count >= 3 ? wideBranchHalfAngle : branchHalfAngle
    const gap = count <= 1 ? Math.PI : depth === 0 ? 2 * Math.PI / count : 2 * halfAngle / (count - 1)
    const maxRadius = Math.max(...subtrees.flatMap((subtree) => subtree.map((item) => Math.hypot(item.x, item.y) + item.size / 2 + spacing)))
    const upperBound = Math.max(size + maxRadius + spacing, (maxRadius + spacing) / Math.sin(Math.min(gap / 2, Math.PI / 2)) + size)
    let distance = (size + subtrees[0][0].size) / 2 + spacing
    let branches: Branch[]
    // Keep siblings on the same arc, expanding it only when their actual geometry collides.
    while (true) {
      branches = subtrees.map((subtree, index) => {
        const angle = count === 1 ? 0 : depth === 0 ? index * gap : -halfAngle + index * gap
        const x = Math.cos(angle) * distance
        const y = Math.sin(angle) * distance
        const rotation = depth === 0 ? angle : 0
        const cosine = Math.cos(rotation)
        const sine = Math.sin(rotation)
        const bubbles = subtree.map((item) => ({
          ...item,
          x: x + item.x * cosine - item.y * sine,
          y: y + item.x * sine + item.y * cosine,
        }))
        const links = bubbleLinks(bubbles).map((link) => {
          const radians = link.angle * Math.PI / 180
          return { x1: link.x, y1: link.y, x2: link.x + Math.cos(radians) * link.length, y2: link.y + Math.sin(radians) * link.length }
        })
        return {
          bubbles,
          links,
          connection: {
            x1: Math.cos(angle) * size / 2,
            y1: Math.sin(angle) * size / 2,
            x2: x - Math.cos(angle) * bubbles[0].size / 2,
            y2: y - Math.sin(angle) * bubbles[0].size / 2,
          },
        }
      })
      if (branchesFit(parent, branches)) break
      if (distance >= upperBound) {
        throw new Error(`Unable to place bubble branches without collisions: ${id}`)
      }
      distance = Math.min(upperBound, Math.max(distance + 4, distance * 1.05))
    }
    return [parent, ...branches.flatMap((branch) => branch.bubbles)]
  }
  const items = walk(root, 'r', 0)
  const childrenByParent = new Map<string, PlacedBubble[]>()
  for (const item of items) {
    const parentId = parentBubbleId(item.id)
    if (!parentId) continue
    const children = childrenByParent.get(parentId) ?? []
    children.push(item)
    childrenByParent.set(parentId, children)
  }
  // Double the existing edge-to-edge gaps, then expand further if descendants collide.
  function spaceBranch(parent: PlacedBubble): PlacedBubble[] {
    const children = childrenByParent.get(parent.id) ?? []
    if (!children.length) return [parent]
    const subtrees = children.map(spaceBranch)
    const initialDistance = Math.hypot(children[0].x - parent.x, children[0].y - parent.y)
    const radii = (parent.size + children[0].size) / 2
    let distance = radii + (initialDistance - radii) * linkFactor
    const subtreeExtent = Math.max(...subtrees.flatMap((subtree, index) => subtree.map(
      (item) => Math.hypot(item.x - children[index].x, item.y - children[index].y) + item.size / 2 + spacing,
    )))
    const depth = parent.id.split('.').length - 1
    const halfAngle = children.length >= 3 ? wideBranchHalfAngle : branchHalfAngle
    const gap = children.length <= 1 ? Math.PI : depth === 0 ? 2 * Math.PI / children.length : 2 * halfAngle / (children.length - 1)
    const upperBound = Math.max(distance, parent.size + subtreeExtent + spacing,
      (subtreeExtent + spacing) / Math.sin(Math.min(gap / 2, Math.PI / 2)) + parent.size)
    let branches: Branch[]
    while (true) {
      branches = subtrees.map((subtree, index) => {
        const child = children[index]
        const dx = (child.x - parent.x) / initialDistance
        const dy = (child.y - parent.y) / initialDistance
        const shift = distance - initialDistance
        const bubbles = subtree.map((item) => ({ ...item, x: item.x + dx * shift, y: item.y + dy * shift }))
        const links = bubbleLinks(bubbles).map((link) => {
          const angle = link.angle * Math.PI / 180
          return { x1: link.x, y1: link.y, x2: link.x + Math.cos(angle) * link.length, y2: link.y + Math.sin(angle) * link.length }
        })
        return {
          bubbles,
          links,
          connection: {
            x1: parent.x + dx * parent.size / 2,
            y1: parent.y + dy * parent.size / 2,
            x2: parent.x + dx * (distance - child.size / 2),
            y2: parent.y + dy * (distance - child.size / 2),
          },
        }
      })
      if (branchesFit(parent, branches)) break
      if (distance >= upperBound) throw new Error(`Unable to space bubble branches without collisions: ${parent.id}`)
      distance = Math.min(upperBound, Math.max(distance + 4, distance * 1.05))
    }
    return [parent, ...branches.flatMap((branch) => branch.bubbles)]
  }
  const spacedItems = spaceBranch(items[0])
  const minX = Math.min(...spacedItems.map((item) => item.x - item.size / 2))
  const maxX = Math.max(...spacedItems.map((item) => item.x + item.size / 2))
  const minY = Math.min(...spacedItems.map((item) => item.y - item.size / 2))
  const maxY = Math.max(...spacedItems.map((item) => item.y + item.size / 2))
  return spacedItems.map((item) => ({
    ...item,
    x: (item.x - (minX + maxX) / 2) * scale,
    y: (item.y - (minY + maxY) / 2) * scale,
    size: item.size * scale,
  }))
}

export function bubbleLinks(bubbles: PlacedBubble[]): BubbleLink[] {
  const byId = new Map(bubbles.map((bubble) => [bubble.id, bubble]))
  const links: BubbleLink[] = []
  for (const bubble of bubbles) {
    const parent = byId.get(parentBubbleId(bubble.id) ?? '')
    if (!parent) continue
    const dx = bubble.x - parent.x
    const dy = bubble.y - parent.y
    const distance = Math.hypot(dx, dy)
    const ux = dx / distance
    const uy = dy / distance
    links.push({
      id: bubble.id,
      x: parent.x + ux * parent.size / 2,
      y: parent.y + uy * parent.size / 2,
      length: distance - (parent.size + bubble.size) / 2,
      angle: Math.atan2(dy, dx) * 180 / Math.PI,
    })
  }
  return links
}

export function mapBounds(bubbles: PlacedBubble[]): { width: number; height: number } {
  return {
    width: Math.max(...bubbles.map((bubble) => 2 * (Math.abs(bubble.x) + bubble.size / 2))),
    height: Math.max(...bubbles.map((bubble) => 2 * (Math.abs(bubble.y) + bubble.size / 2))),
  }
}

export function fitScale(bounds: { width: number; height: number }, width: number, height: number): number {
  const fit = Math.min((width - 36) / bounds.width, (height - 56) / bounds.height)
  if (!Number.isFinite(fit)) {
    return 1
  }
  return Math.max(minScale, Math.min(maxScale, fit))
}

export function findBubble(root: BubbleNode, id: string): BubbleNode | null {
  const indexes = id.split('.').slice(1)
  let node = root
  for (const part of indexes) {
    const child = node.children?.[Number(part)]
    if (!child) {
      return null
    }
    node = child
  }
  return node
}

export function ancestorIds(id: string): string[] {
  const parts = id.split('.')
  const ancestors: string[] = []
  for (let length = 1; length < parts.length; length += 1) {
    ancestors.push(parts.slice(0, length).join('.'))
  }
  return ancestors
}

// stepOpenLayer 未展开时只打开直接子气泡；已展开时只收回下面最深的一层。
export function stepOpenLayer(openIds: ReadonlySet<string>, id: string): Set<string> {
  if (!openIds.has(id)) {
    const next = new Set(openIds)
    next.add(id)
    return next
  }
  const depth = bubbleDepth(id)
  let maxDepth = depth
  const prefix = `${id}.`
  for (const openId of openIds) {
    if (openId === id || openId.startsWith(prefix)) {
      maxDepth = Math.max(maxDepth, bubbleDepth(openId))
    }
  }
  const next = new Set<string>()
  for (const openId of openIds) {
    const underNode = openId === id || openId.startsWith(prefix)
    if (underNode && bubbleDepth(openId) === maxDepth) {
      continue
    }
    next.add(openId)
  }
  return next
}

export function closeDeepestLayer(openIds: ReadonlySet<string>): Set<string> {
  let maxDepth = -1
  for (const id of openIds) {
    maxDepth = Math.max(maxDepth, id.split('.').length - 1)
  }
  const next = new Set<string>()
  for (const id of openIds) {
    if (id.split('.').length - 1 === maxDepth) {
      continue
    }
    next.add(id)
  }
  return next
}

function bubbleDepth(id: string) {
  return id.split('.').length - 1
}

export function parentBubbleId(id: string): string | null {
  const dot = id.lastIndexOf('.')
  if (dot < 0) {
    return null
  }
  return id.slice(0, dot)
}
