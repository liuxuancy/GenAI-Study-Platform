import type { RoleCode } from '@/core/api/account'

import { translateApp } from '@/core/i18n/translateApp'
import { getAppLocale } from '@/core/i18n/store'

export function localizedRoleName(role: RoleCode, locale = getAppLocale()) {
  return translateApp(locale, `roles.${role}`)
}
