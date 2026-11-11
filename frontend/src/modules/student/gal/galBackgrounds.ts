import type { GalScriptItem } from '@/modules/student/gal/galTypes'

export type GalBackgroundKey =
  | 'abstract-nodes'
  | 'campus-path'
  | 'cave-hub'
  | 'chapter-clear'
  | 'classroom-soft'
  | 'computer-lab'
  | 'library'
  | 'night-lamp'
  | 'quiz-focus'
  | 'study-desk'

const GAL_BACKGROUND_KEYS: GalBackgroundKey[] = [
  'abstract-nodes',
  'campus-path',
  'cave-hub',
  'chapter-clear',
  'classroom-soft',
  'computer-lab',
  'library',
  'night-lamp',
  'quiz-focus',
  'study-desk',
]

/** Dialogue backgrounds cycled every few lines (default start: cave-hub). */
export const GAL_DIALOGUE_BG_CYCLE: GalBackgroundKey[] = [
  'cave-hub',
  'classroom-soft',
  'study-desk',
  'library',
  'campus-path',
  'computer-lab',
  'abstract-nodes',
  'night-lamp',
]

const DIALOGUE_LINES_PER_BG = 2

export function isGalBackgroundKey(value: string): value is GalBackgroundKey {
  return (GAL_BACKGROUND_KEYS as string[]).includes(value)
}

export function galBackgroundUrl(key: GalBackgroundKey): string {
  return `/gal/bg/${key}.png`
}

export function resolveGalBackground({
  script,
  currentIndex,
  finished,
  currentItem,
}: {
  script: GalScriptItem[]
  currentIndex: number
  finished: boolean
  currentItem: GalScriptItem | undefined
}): GalBackgroundKey {
  if (finished) {
    return 'chapter-clear'
  }
  if (!currentItem) {
    return 'cave-hub'
  }
  if (currentItem.kind === 'choice') {
    return 'quiz-focus'
  }
  if (currentItem.kind === 'dialogue' && currentItem.background && isGalBackgroundKey(currentItem.background)) {
    return currentItem.background
  }
  let dialogueOrdinal = 0
  for (let i = 0; i <= currentIndex; i += 1) {
    if (script[i]?.kind === 'dialogue') {
      dialogueOrdinal += 1
    }
  }
  if (dialogueOrdinal <= 0) {
    return 'cave-hub'
  }
  const slot = Math.floor((dialogueOrdinal - 1) / DIALOGUE_LINES_PER_BG) % GAL_DIALOGUE_BG_CYCLE.length
  return GAL_DIALOGUE_BG_CYCLE[slot] ?? 'cave-hub'
}
