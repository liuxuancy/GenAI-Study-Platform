export type WalkerPhase = 'walking' | 'resting'
export type DialogueKind = 'touch' | 'rest'

export function scheduleDialogueExpiry(onExpire: () => void) {
  return setTimeout(onExpire, 2000)
}

export function advanceWalker(phase: WalkerPhase, remaining: number, elapsed: number, nextDuration: (phase: WalkerPhase) => number) {
  const consumed = Math.min(remaining, elapsed)
  const walkingTime = phase === 'walking' ? consumed : 0
  if (elapsed < remaining) return { phase, remaining: remaining - elapsed, walkingTime, beganRest: false }
  const next: WalkerPhase = phase === 'walking' ? 'resting' : 'walking'
  return { phase: next, remaining: nextDuration(next), walkingTime, beganRest: next === 'resting' }
}

export function registerWalkerClick(previous: readonly number[], now: number) {
  const clicks = [...previous.filter((time) => now - time <= 2000), now]
  return clicks.length >= 3 ? { clicks: [], triggered: true } : { clicks, triggered: false }
}

export function canShowWalkerDialogue(current: DialogueKind | null, incoming: DialogueKind, available: boolean) {
  return available && !(current === 'touch' && incoming === 'rest')
}
