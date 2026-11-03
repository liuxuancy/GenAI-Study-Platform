import { useI18n } from '@/core/i18n/useI18n'
import type { AppLocale } from '@/core/i18n/types'

import styles from '@/core/i18n/LanguageSwitcher.module.css'

interface LanguageSwitcherProps {
  compact?: boolean
  className?: string
}

function cx(...parts: Array<string | false | undefined>) {
  return parts.filter(Boolean).join(' ')
}

export function LanguageSwitcher({ compact, className }: LanguageSwitcherProps) {
  const { locale, setLocale, t } = useI18n()

  function select(next: AppLocale) {
    if (next !== locale) {
      setLocale(next)
    }
  }

  return (
    <div
      className={cx(styles.switcher, compact && styles.compact, className)}
      role="group"
      aria-label={t('language.switchAria')}
    >
      <button
        type="button"
        className={cx(styles.btn, locale === 'zh' && styles.btnActive)}
        aria-pressed={locale === 'zh'}
        onClick={() => select('zh')}
      >
        {t('language.zh')}
      </button>
      <button
        type="button"
        className={cx(styles.btn, locale === 'en' && styles.btnActive)}
        aria-pressed={locale === 'en'}
        onClick={() => select('en')}
      >
        {t('language.en')}
      </button>
    </div>
  )
}
