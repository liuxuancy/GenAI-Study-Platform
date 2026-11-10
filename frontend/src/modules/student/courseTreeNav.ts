import type { BubbleNode } from '@/modules/student/bubbleTree'

export interface ChapterRef {
  id: string
  index: number
  node: BubbleNode
}

export interface ActivityRef {
  id: string
  chapterId: string
  chapterKey: string
  chapterTitle: string
  node: BubbleNode
}

export function listChapters(tree: BubbleNode): ChapterRef[] {
  return (tree.children ?? []).map((node, index) => ({
    id: `r.${index}`,
    index,
    node,
  }))
}

export function listChapterActivities(chapterId: string, chapter: BubbleNode): ActivityRef[] {
  return (chapter.children ?? []).map((node, index) => ({
    id: `${chapterId}.${index}`,
    chapterId,
    chapterKey: chapter.key ?? '',
    chapterTitle: chapter.label,
    node,
  }))
}

export function flattenActivities(tree: BubbleNode): ActivityRef[] {
  const items: ActivityRef[] = []
  for (const chapter of listChapters(tree)) {
    items.push(...listChapterActivities(chapter.id, chapter.node))
  }
  return items
}

export type ChapterProgress = 'completed' | 'learning' | 'pending'

export function chapterProgress(
  chapterId: string,
  chapter: BubbleNode,
  doneIds: ReadonlySet<string>,
  activeChapterId: string,
): ChapterProgress {
  const activities = listChapterActivities(chapterId, chapter)
  if (activities.length === 0) {
    return chapterId === activeChapterId ? 'learning' : 'pending'
  }
  const doneCount = activities.filter((item) => doneIds.has(item.id)).length
  if (doneCount === activities.length) {
    return 'completed'
  }
  if (doneCount > 0 || chapterId === activeChapterId) {
    return 'learning'
  }
  return 'pending'
}

export function isActivityDone(id: string, doneIds: ReadonlySet<string>) {
  return doneIds.has(id)
}

export function completedBubbleIds(tree: BubbleNode, doneIds: ReadonlySet<string>): Set<string> {
  const completed = new Set<string>()
  function visit(node: BubbleNode, id: string): boolean {
    const children = node.children ?? []
    const childStates = children.map((child, index) => visit(child, `${id}.${index}`))
    const done = children.length > 0 ? childStates.every(Boolean) : doneIds.has(id)
    if (done) {
      completed.add(id)
    }
    return done
  }
  visit(tree, 'r')
  return completed
}

export function leafPercent(tree: BubbleNode, doneIds: ReadonlySet<string>) {
  const total = countLeaves(tree)
  if (total <= 0) {
    return 0
  }
  const percent = Math.floor((countDoneLeaves(tree, 'r', doneIds) * 100) / total)
  return percent > 100 ? 100 : percent
}

function countLeaves(node: BubbleNode): number {
  if (!node.children?.length) {
    return 1
  }
  return node.children.reduce((sum, child) => sum + countLeaves(child), 0)
}

function countDoneLeaves(node: BubbleNode, id: string, doneIds: ReadonlySet<string>): number {
  if (!node.children?.length) {
    return doneIds.has(id) ? 1 : 0
  }
  let total = 0
  for (let index = 0; index < node.children.length; index += 1) {
    total += countDoneLeaves(node.children[index], `${id}.${index}`, doneIds)
  }
  return total
}
