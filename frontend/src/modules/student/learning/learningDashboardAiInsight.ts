const storagePrefix = 'caveclass.learningDashboard.aiInsight'

export type LearningDashboardAiInsight = {
  suggestion: string
  recommendedAction: { title: string; detail: string }
}

function storageKey(userId: number) {
  return `${storagePrefix}.${userId}`
}

export function readLearningDashboardAiInsight(userId: number | undefined): LearningDashboardAiInsight | null {
  if (userId == null) {
    return null
  }
  try {
    const raw = sessionStorage.getItem(storageKey(userId))
    if (!raw) {
      return null
    }
    const parsed = JSON.parse(raw) as LearningDashboardAiInsight
    if (typeof parsed.suggestion !== 'string' || !parsed.recommendedAction) {
      return null
    }
    const { title, detail } = parsed.recommendedAction
    if (typeof title !== 'string' || typeof detail !== 'string') {
      return null
    }
    if (!parsed.suggestion.trim()) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

export function writeLearningDashboardAiInsight(userId: number | undefined, insight: LearningDashboardAiInsight) {
  if (userId == null) {
    return
  }
  try {
    sessionStorage.setItem(storageKey(userId), JSON.stringify(insight))
  } catch {
    // ignore quota / private mode
  }
}
