import { describe, expect, it } from 'vitest'

import { completedBubbleIds } from '@/modules/student/courseTreeNav'
import type { BubbleNode } from '@/modules/student/bubbleTree'

const tree: BubbleNode = {
  label: 'Course', note: '', children: [
    { label: 'First', note: '', children: [
      { label: 'One', note: '' },
      { label: 'Two', note: '' },
    ] },
    { label: 'Second', note: '', children: [{ label: 'Three', note: '' }] },
  ],
}

describe('completedBubbleIds', () => {
  it('marks individual completed sections but not their incomplete ancestors', () => {
    expect([...completedBubbleIds(tree, new Set(['r.0.0']))]).toEqual(['r.0.0'])
  })

  it('marks a chapter only after all of its children are complete', () => {
    expect([...completedBubbleIds(tree, new Set(['r.0.0', 'r.0.1']))]).toEqual(['r.0.0', 'r.0.1', 'r.0'])
  })

  it('propagates completion to the root when every branch is complete', () => {
    expect([...completedBubbleIds(tree, new Set(['r.0.0', 'r.0.1', 'r.1.0']))])
      .toEqual(['r.0.0', 'r.0.1', 'r.0', 'r.1.0', 'r.1', 'r'])
  })

  it('does not complete empty chapters or the course by vacuous truth', () => {
    const emptyChapter: BubbleNode = { label: 'Course', note: '', children: [{ label: 'Empty', note: '', children: [] }] }
    expect([...completedBubbleIds(emptyChapter, new Set())]).toEqual([])
    expect([...completedBubbleIds({ label: 'Empty course', note: '', children: [] }, new Set())]).toEqual([])
  })
})
