// 气泡图的展开布局：子节点沿外圈均分，半径随同层间距撑开。

import type { BubbleNode } from '@/modules/student/bubbleTree'

const bubbleSize = [210, 122, 98, 78]
const bubbleRing = [0, 188, 312, 416]

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

interface LayoutItem {
  id: string
  depth: number
  angle: number
  node: BubbleNode
}

export function placeBubbles(root: BubbleNode, openIds: ReadonlySet<string>, scale: number): PlacedBubble[] {
  const items = layoutItems(root, openIds)
  const radii = ringRadii(items)
  return items.map((item) => ({
    id: item.id,
    depth: item.depth,
    node: item.node,
    x: Math.cos(item.angle) * radii[item.depth] * scale,
    y: Math.sin(item.angle) * radii[item.depth] * scale,
    size: bubbleSize[item.depth] * scale,
  }))
}

export function bubbleLinks(bubbles: PlacedBubble[]): BubbleLink[] {
  const byId = new Map(bubbles.map((bubble) => [bubble.id, bubble]))
  const links: BubbleLink[] = []
  for (const bubble of bubbles) {
    const parent = byId.get(parentBubbleId(bubble.id) ?? '')
    if (!parent) {
      continue
    }
    const dx = bubble.x - parent.x
    const dy = bubble.y - parent.y
    links.push({
      id: bubble.id,
      x: parent.x,
      y: parent.y,
      length: Math.hypot(dx, dy),
      angle: (Math.atan2(dy, dx) * 180) / Math.PI,
    })
  }
  return links
}

export function mapExtent(root: BubbleNode, openIds: ReadonlySet<string>): number {
  const items = layoutItems(root, openIds)
  const radii = ringRadii(items)
  let extent = bubbleSize[0] / 2
  for (const item of items) {
    extent = Math.max(extent, radii[item.depth] + bubbleSize[item.depth] / 2)
  }
  return extent
}

export function fitScale(extent: number, width: number, height: number): number {
  const fit = Math.min(width - 36, height - 56) / (2 * extent)
  if (!Number.isFinite(fit)) {
    return 1
  }
  return Math.max(0.52, Math.min(1, fit))
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

function layoutItems(root: BubbleNode, openIds: ReadonlySet<string>): LayoutItem[] {
  const items: LayoutItem[] = []
  walkNode(root, 'r', 0, -Math.PI / 2, Math.PI * 1.5, openIds, items)
  return items
}

function walkNode(
  node: BubbleNode,
  id: string,
  depth: number,
  startAngle: number,
  endAngle: number,
  openIds: ReadonlySet<string>,
  items: LayoutItem[],
) {
  items.push({ id, depth, angle: (startAngle + endAngle) / 2, node })
  if (!openIds.has(id) || !node.children?.length) {
    return
  }
  const width = (endAngle - startAngle) / node.children.length
  for (let index = 0; index < node.children.length; index += 1) {
    const child = node.children[index]
    walkNode(child, `${id}.${index}`, depth + 1, startAngle + index * width, startAngle + (index + 1) * width, openIds, items)
  }
}

function ringRadii(items: LayoutItem[]): number[] {
  const angles: number[][] = [[], [], [], []]
  for (const item of items) {
    angles[item.depth]?.push(item.angle)
  }
  const radii = [...bubbleRing]
  for (let depth = 1; depth < bubbleSize.length; depth += 1) {
    radii[depth] = Math.max(radii[depth], radii[depth - 1] + (bubbleSize[depth - 1] + bubbleSize[depth]) / 2 + 18)
    const sorted = [...(angles[depth] ?? [])].sort((left, right) => left - right)
    if (sorted.length > 1) {
      let gap = sorted[0] + Math.PI * 2 - sorted[sorted.length - 1]
      for (let index = 1; index < sorted.length; index += 1) {
        gap = Math.min(gap, sorted[index] - sorted[index - 1])
      }
      radii[depth] = Math.max(radii[depth], Math.min(820, (bubbleSize[depth] + 16) / gap))
    }
  }
  return radii
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
