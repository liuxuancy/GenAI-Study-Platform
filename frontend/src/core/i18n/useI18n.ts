import { useCallback } from 'react'

import { useLocaleStore } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'

export { translateApp } from '@/core/i18n/translateApp'

export function useI18n() {
  const locale = useLocaleStore((state) => state.locale)
  const setLocale = useLocaleStore((state) => state.setLocale)
  const toggleLocale = useLocaleStore((state) => state.toggleLocale)

  const t = useCallback(
    (key: string, params?: Record<string, string | number>) => translateApp(locale, key, params),
    [locale],
  )

  return { locale, setLocale, toggleLocale, t }
}
