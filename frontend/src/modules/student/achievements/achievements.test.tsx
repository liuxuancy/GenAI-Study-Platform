import { renderToStaticMarkup } from 'react-dom/server'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { AchievementsPage } from './AchievementsPage'
import { scheduleNotificationRemoval } from './achievementTypes'

const state = vi.hoisted(() => ({ unlocked: [] as { key: string; unlockedAt: string }[], loading: false }))
vi.mock('./AchievementProvider', () => ({ useAchievements: () => state }))
vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('achievements', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    state.unlocked = []
  })

  it('only renders generic placeholders for locked achievements', () => {
    state.unlocked = [{ key: 'affection_50', unlockedAt: '2026-01-01' }]
    const html = renderToStaticMarkup(<AchievementsPage />)
    expect(html).toContain('student.achievements.affection_50.title')
    expect(html).toContain('student.achievements.affection_50.description')
    expect(html).not.toContain('student.achievements.affection_100')
    expect(html).not.toContain('student.achievements.affection_five_100')
    expect(html).not.toContain('student.achievements.first_pixel_avatar')
    expect(html.match(/student\.achievements\.locked/g)).toHaveLength(3)
  })

  it('displays the first companion achievement once unlocked', () => {
    state.unlocked = [{ key: 'first_pixel_avatar', unlockedAt: '2026-01-01' }]
    const html = renderToStaticMarkup(<AchievementsPage />)
    expect(html).toContain('student.achievements.first_pixel_avatar.title')
    expect(html).toContain('student.achievements.first_pixel_avatar.description')
    expect(html.match(/student\.achievements\.locked/g)).toHaveLength(3)
  })

  it('removes each notification after exactly three seconds', () => {
    vi.useFakeTimers()
    vi.stubGlobal('window', globalThis)
    const first = vi.fn()
    const second = vi.fn()
    scheduleNotificationRemoval(first)
    scheduleNotificationRemoval(second)
    vi.advanceTimersByTime(2999)
    expect(first).not.toHaveBeenCalled()
    expect(second).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(first).toHaveBeenCalledOnce()
    expect(second).toHaveBeenCalledOnce()
  })
})
