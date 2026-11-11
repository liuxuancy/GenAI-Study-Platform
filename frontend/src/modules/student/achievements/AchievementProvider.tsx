import { Alert } from 'antd'
import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { http } from '@/core/http'
import { useI18n } from '@/core/i18n/useI18n'
import { scheduleNotificationRemoval, type Achievement, type AchievementSync } from './achievementTypes'
import styles from './achievements.module.css'

interface AchievementState {
  unlocked: Achievement[]
  loading: boolean
}

const AchievementContext = createContext<AchievementState>({ unlocked: [], loading: true })

export function useAchievements() {
  return useContext(AchievementContext)
}

export function AchievementProvider() {
  const currentUser = useAuthStore((state) => state.currentUser)
  const { t } = useI18n()
  const [unlocked, setUnlocked] = useState<Achievement[]>([])
  const [notifications, setNotifications] = useState<Achievement[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const active = useRef(false)
  const running = useRef(false)
  const pending = useRef(false)
  const timers = useRef<number[]>([])
  const userId = currentUser?.role === 'ROLE_STUDENT' ? currentUser.id : undefined

  const sync = useCallback(async () => {
    if (!userId) return
    if (running.current) {
      pending.current = true
      return
    }
    running.current = true
    try {
      do {
        pending.current = false
        const result = await http.post<AchievementSync>('/student/achievements/sync')
        if (!active.current) return
        setUnlocked(result.unlocked)
        setError('')
        if (result.notify.length) {
          setNotifications((previous) => [...previous, ...result.notify])
          for (const achievement of result.notify) {
            const timer = scheduleNotificationRemoval(() => {
              setNotifications((previous) => previous.filter((item) => item.key !== achievement.key))
            })
            timers.current.push(timer)
          }
        }
      } while (pending.current && active.current)
    } catch (reason) {
      if (active.current) setError(reason instanceof Error ? reason.message : 'Unknown error')
    } finally {
      running.current = false
      if (active.current) setLoading(false)
    }
  }, [userId])

  useEffect(() => {
    active.current = true
    if (userId) void sync()
    const refresh = () => void sync()
    window.addEventListener('student-achievements-changed', refresh)
    window.addEventListener('focus', refresh)
    return () => {
      active.current = false
      window.removeEventListener('student-achievements-changed', refresh)
      window.removeEventListener('focus', refresh)
      timers.current.forEach(window.clearTimeout)
      timers.current = []
    }
  }, [sync, userId])

  return (
    <AchievementContext.Provider value={{ unlocked, loading }}>
      <Outlet />
      {userId ? (
        <div className={styles.notifications} aria-live="polite">
          {error ? (
            <Alert type="error" showIcon message={t('student.achievements.loadFailed')} description={error}
              action={<button type="button" onClick={() => void sync()}>{t('student.achievements.retry')}</button>} />
          ) : null}
          {notifications.map((achievement) => (
            <div className={styles.notification} role="status" key={achievement.key}>
              <span aria-hidden="true">🏆</span>
              <div>
                <strong>{t('student.achievements.unlocked')}</strong>
                <div>{t(`student.achievements.${achievement.key}.title`)}</div>
              </div>
            </div>
          ))}
        </div>
      ) : null}
    </AchievementContext.Provider>
  )
}
