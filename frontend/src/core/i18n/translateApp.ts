import { enMessages, zhMessages } from '@/core/i18n/messages'
import { translate } from '@/core/i18n/translate'
import type { AppLocale } from '@/core/i18n/types'

const catalogs: Record<AppLocale, typeof zhMessages> = {
  zh: zhMessages,
  en: enMessages,
}

export function translateApp(
  locale: AppLocale,
  key: string,
  params?: Record<string, string | number>,
) {
  const tree = catalogs[locale]
  return translate(tree, key, params, enMessages)
}
