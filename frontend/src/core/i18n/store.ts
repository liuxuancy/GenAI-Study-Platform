import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { applyDocumentLocale } from '@/core/i18n/translate'
import type { AppLocale } from '@/core/i18n/types'

const storageKey = 'caveclass.locale'

function readInitialLocale(): AppLocale {
  if (typeof window === 'undefined') {
    return 'zh'
  }
  const stored = window.localStorage.getItem(storageKey)
  if (stored === 'en' || stored === 'zh') {
    return stored
  }
  return 'zh'
}

interface LocaleState {
  locale: AppLocale
  setLocale: (locale: AppLocale) => void
  toggleLocale: () => void
}

export const useLocaleStore = create<LocaleState>()(
  persist(
    (set, get) => ({
      locale: readInitialLocale(),
      setLocale: (locale) => {
        applyDocumentLocale(locale)
        set({ locale })
      },
      toggleLocale: () => {
        const next: AppLocale = get().locale === 'zh' ? 'en' : 'zh'
        get().setLocale(next)
      },
    }),
    {
      name: storageKey,
      partialize: (state) => ({ locale: state.locale }),
      onRehydrateStorage: () => (state) => {
        if (state) {
          applyDocumentLocale(state.locale)
        }
      },
    },
  ),
)

if (typeof window !== 'undefined') {
  applyDocumentLocale(readInitialLocale())
}

export function getAppLocale(): AppLocale {
  return useLocaleStore.getState().locale
}
