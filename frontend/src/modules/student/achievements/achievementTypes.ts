export const achievementKeys = ['first_pixel_avatar', 'affection_50', 'affection_100', 'affection_five_100'] as const
export type AchievementKey = (typeof achievementKeys)[number]

export interface Achievement {
  key: AchievementKey
  unlockedAt: string
}

export interface AchievementSync {
  unlocked: Achievement[]
  notify: Achievement[]
}

export const achievementNotificationDuration = 3000

export function scheduleNotificationRemoval(remove: () => void) {
  return window.setTimeout(remove, achievementNotificationDuration)
}
