import type { LoginAudience } from '@/core/auth/loginAudience'
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'

function audienceRoleKey(audience: LoginAudience) {
  if (audience === 'student') {
    return 'roles.ROLE_STUDENT'
  }
  if (audience === 'teacher') {
    return 'roles.ROLE_TEACHER'
  }
  return 'roles.ROLE_ADMIN'
}

export function audienceMismatchMessage(audience: LoginAudience, locale = getAppLocale()) {
  const role = translateApp(locale, audienceRoleKey(audience))
  return translateApp(locale, 'login.error.audienceMismatch', { role })
}
