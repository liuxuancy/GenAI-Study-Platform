import { useEffect, useRef } from 'react'

import { postCourseLearningHeartbeat } from '@/modules/student/api'

/** 课程页可见时累计学习时长并上报（服务端单次 1–120 秒）。 */
export function useCourseLearningHeartbeat(courseId: string | undefined, enabled: boolean) {
  const accumulatedRef = useRef(0)
  const tickAtRef = useRef(Date.now())

  useEffect(() => {
    if (!enabled || !courseId) {
      return
    }
    const activeCourseId = courseId

    accumulatedRef.current = 0
    tickAtRef.current = Date.now()

    function flush(forceAll = false) {
      let seconds = accumulatedRef.current
      if (!forceAll && seconds < 15) {
        return
      }
      seconds = Math.min(Math.max(seconds, 1), 120)
      accumulatedRef.current = 0
      void postCourseLearningHeartbeat(activeCourseId, seconds).catch(() => {})
    }

    function captureElapsed() {
      const elapsed = Math.floor((Date.now() - tickAtRef.current) / 1000)
      tickAtRef.current = Date.now()
      if (elapsed > 0) {
        accumulatedRef.current += elapsed
      }
    }

    function onHidden() {
      captureElapsed()
      flush(true)
    }

    function onVisible() {
      tickAtRef.current = Date.now()
    }

    const interval = window.setInterval(() => {
      if (document.visibilityState !== 'visible') {
        return
      }
      captureElapsed()
      if (accumulatedRef.current >= 60) {
        flush(false)
      }
    }, 20_000)

    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') {
        onHidden()
      } else {
        onVisible()
      }
    })
    window.addEventListener('pagehide', onHidden)

    return () => {
      window.clearInterval(interval)
      window.removeEventListener('pagehide', onHidden)
      captureElapsed()
      flush(true)
    }
  }, [courseId, enabled])
}
