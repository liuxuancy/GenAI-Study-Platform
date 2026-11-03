import { describe, expect, it } from 'vitest'

import { aiPixelWidth, defaultAiRatio, expandedAiRatio, readAiRatio, separatorWidth } from '@/modules/student/learning/workspaceLayout'

describe('aiPixelWidth', () => {
  it('空间够时 AI 不窄于 300，正文不窄于 480', () => {
    const available = 900
    const width = aiPixelWidth(defaultAiRatio, available)
    expect(width).toBeGreaterThanOrEqual(300)
    expect(available - separatorWidth - width).toBeGreaterThanOrEqual(480)
  })

  it('展开比例高于默认，且不会挤掉正文最小宽度', () => {
    const available = 1400
    const expanded = aiPixelWidth(expandedAiRatio, available)
    expect(expanded).toBeGreaterThan(aiPixelWidth(defaultAiRatio, available))
    expect(available - separatorWidth - expanded).toBeGreaterThanOrEqual(480)
  })

  it('空间不足时两侧宽度之和不超过可用宽度', () => {
    const available = 420
    const width = aiPixelWidth(0.9, available)
    expect(width).toBeGreaterThan(0)
    expect(width).toBeLessThanOrEqual(available - separatorWidth)
  })
})

describe('readAiRatio', () => {
  it('非法存储回到默认比例', () => {
    expect(readAiRatio(null)).toBe(defaultAiRatio)
    expect(readAiRatio('not-json')).toBe(defaultAiRatio)
    expect(readAiRatio('{"aiRatio":4}')).toBe(defaultAiRatio)
    expect(readAiRatio('{"aiRatio":0.45}')).toBe(0.45)
  })
})
