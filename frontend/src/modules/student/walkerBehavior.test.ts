import { describe, expect, it, vi } from 'vitest'
import { advanceWalker, canShowWalkerDialogue, registerWalkerClick, scheduleDialogueExpiry } from './walkerBehavior'

describe('walker behavior', () => {
  it('walks, rests, then resumes without moving while resting', () => {
    const duration = (phase: 'walking' | 'resting') => phase === 'resting' ? 2000 : 15000
    expect(advanceWalker('walking', 15000, 14999, duration)).toEqual({
      phase: 'walking', remaining: 1, walkingTime: 14999, beganRest: false,
    })
    expect(advanceWalker('walking', 1, 1, duration)).toEqual({
      phase: 'resting', remaining: 2000, walkingTime: 1, beganRest: true,
    })
    expect(advanceWalker('resting', 2000, 2000, duration)).toEqual({
      phase: 'walking', remaining: 15000, walkingTime: 0, beganRest: false,
    })
  })

  it('requires three clicks within two seconds and resets after a trigger', () => {
    let clicks: number[] = []
    for (const time of [0, 1000]) {
      const result = registerWalkerClick(clicks, time)
      expect(result.triggered).toBe(false)
      clicks = result.clicks
    }
    expect(registerWalkerClick(clicks, 2000)).toEqual({ clicks: [], triggered: true })
    expect(registerWalkerClick(clicks, 2001)).toEqual({ clicks: [1000, 2001], triggered: false })
  })

  it('prioritizes touch dialogue and does not display missing dialogue', () => {
    expect(canShowWalkerDialogue('touch', 'rest', true)).toBe(false)
    expect(canShowWalkerDialogue('rest', 'touch', true)).toBe(true)
    expect(canShowWalkerDialogue(null, 'rest', false)).toBe(false)
  })

  it('expires a bubble after exactly two seconds and permits restarting the timer', () => {
    vi.useFakeTimers()
    try {
      const expire = vi.fn()
      const first = scheduleDialogueExpiry(expire)
      vi.advanceTimersByTime(1500)
      clearTimeout(first)
      scheduleDialogueExpiry(expire)
      vi.advanceTimersByTime(1999)
      expect(expire).not.toHaveBeenCalled()
      vi.advanceTimersByTime(1)
      expect(expire).toHaveBeenCalledTimes(1)
    } finally {
      vi.useRealTimers()
    }
  })
})
