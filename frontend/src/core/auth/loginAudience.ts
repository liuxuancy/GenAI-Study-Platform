import type { RoleCode } from '@/core/api/account'
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'
import { localizedRoleName } from '@/core/i18n/roleName'

export type LoginAudience = 'student' | 'teacher' | 'admin'

export function roleForAudience(audience: LoginAudience): RoleCode {
  if (audience === 'student') {
    return 'ROLE_STUDENT'
  }
  if (audience === 'teacher') {
    return 'ROLE_TEACHER'
  }
  return 'ROLE_ADMIN'
}

export function roleMatchesAudience(role: RoleCode, audience: LoginAudience) {
  return role === roleForAudience(audience)
}

export function audienceMismatchMessage(audience: LoginAudience) {
  const locale = getAppLocale()
  const role = roleForAudience(audience)
  const label = localizedRoleName(role, locale)
  return translateApp(locale, 'login.error.audienceMismatch', { role: label })
}
