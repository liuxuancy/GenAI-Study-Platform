export const defaultAiRatio = 0.3
export const expandedAiRatio = 0.45
export const minAiWidth = 300
export const minContentWidth = 480
export const separatorWidth = 16
export const syllabusWidth = 260
export const syllabusRailWidth = 44
export const layoutStorageKey = 'caveclass.workspace-layout'

export function clampAiRatio(ratio: number) {
  if (!Number.isFinite(ratio)) {
    return defaultAiRatio
  }
  return Math.min(0.7, Math.max(0.15, ratio))
}
export function readAiRatio(raw: string | null) {
  if (!raw) {
    return defaultAiRatio
  }
  try {
    const parsed = JSON.parse(raw) as { aiRatio?: unknown }
    if (typeof parsed.aiRatio !== 'number' || !Number.isFinite(parsed.aiRatio)) {
      return defaultAiRatio
    }
    if (parsed.aiRatio < 0.15 || parsed.aiRatio > 0.7) {
      return defaultAiRatio
    }
    return parsed.aiRatio
  } catch {
    return defaultAiRatio
  }
}

export function aiPixelWidth(ratio: number, available: number) {
  const usable = Math.max(0, available - separatorWidth)
  if (usable <= 0) {
    return 0
  }
  const safeRatio = Number.isFinite(ratio) ? ratio : defaultAiRatio
  const wanted = safeRatio * usable
  const fitsMins = usable >= minAiWidth + minContentWidth
  const contentMin = fitsMins ? minContentWidth : Math.round(usable * 0.55)
  const aiMin = fitsMins ? minAiWidth : Math.round(usable * 0.3)
  const maxAi = Math.max(aiMin, usable - contentMin)
  return Math.min(Math.round(Math.min(Math.max(wanted, aiMin), maxAi)), usable)
}

export function ratioFromAiWidth(width: number, available: number) {
  const usable = Math.max(0, available - separatorWidth)
  if (usable <= 0) {
    return defaultAiRatio
  }
  return aiPixelWidth(width / usable, available) / usable
}
