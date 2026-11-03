import { describe, expect, it } from 'vitest'

import type { ActivityRef } from '@/modules/student/courseTreeNav'
import {
  doneIdsAfterComplete,
  firstActivityOfNextChapter,
  lessonPrimaryAction,
  mergeProgressPercent,
  readDoneIds,
  shouldAdvanceAfterComplete,
} from '@/modules/student/learning/lessonProgress'

function activity(id: string, chapterId: string): ActivityRef {
  return {
    id,
    chapterId,
    chapterKey: chapterId,
    chapterTitle: chapterId,
    node: { label: id, note: '' },
  }
}

const activities = [
  activity('r.0.0', 'r.0'),
  activity('r.0.1', 'r.0'),
  activity('r.1.0', 'r.1'),
]

describe('lessonPrimaryAction', () => {
  it('未完成的普通小节先保存再进入下一节', () => {
    expect(lessonPrimaryAction('r.0.0', activities, false).label).toBe('Mark done & next section →')
    expect(shouldAdvanceAfterComplete('r.0.0', activities)).toBe(true)
  })

  it('已完成的普通小节直接显示下一节', () => {
    expect(lessonPrimaryAction('r.0.0', activities, true)).toMatchObject({ label: 'Next section →', action: 'next' })
  })

  it('本章最后一节未完成时不自动跳到下一节', () => {
    expect(lessonPrimaryAction('r.0.1', activities, false).label).toBe('Finish chapter')
    expect(shouldAdvanceAfterComplete('r.0.1', activities)).toBe(false)
    expect(firstActivityOfNextChapter('r.0.1', activities)).toBe('r.1.0')
  })

  it('已完成且还有下一章时进入下一章，课程末尾则停住', () => {
    expect(lessonPrimaryAction('r.0.1', activities, true)).toMatchObject({ label: 'Next chapter', action: 'next-chapter' })
    expect(lessonPrimaryAction('r.1.0', activities, true)).toMatchObject({ label: 'All done', disabled: true })
  })
})

describe('doneIdsAfterComplete', () => {
  it('已完成的小节不会再次加入', () => {
    const done = new Set(['r.0.0'])
    const result = doneIdsAfterComplete(done, 'r.0.0')
    expect(result.changed).toBe(false)
    expect(result.next.size).toBe(1)
    expect(result.next.has('r.0.0')).toBe(true)
  })
})

describe('mergeProgressPercent', () => {
  it('保留服务端已经更高的进度', () => {
    expect(mergeProgressPercent(40, 5)).toBe(40)
    expect(mergeProgressPercent(40, 80)).toBe(80)
  })
})

describe('readDoneIds', () => {
  it('只恢复当前课程里仍然存在的小节', () => {
    expect(readDoneIds('["r.0.0","missing"]', new Set(['r.0.0', 'r.0.1']))).toEqual(['r.0.0'])
    expect(readDoneIds('not-json', new Set(['r.0.0']))).toEqual([])
  })
})
