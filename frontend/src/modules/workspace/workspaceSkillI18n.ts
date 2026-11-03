import type { AppLocale } from '@/core/i18n/types'
import { translateApp } from '@/core/i18n/translateApp'

function skillMessageKey(skillId: string, suffix: string) {
  return `workspace.skills.${skillId}.${suffix}`
}

function pickTranslation(locale: AppLocale, key: string, fallback: string) {
  const value = translateApp(locale, key)
  return value === key ? fallback : value
}

export function localizedSkillName(locale: AppLocale, skillId: string, fallback: string) {
  return pickTranslation(locale, skillMessageKey(skillId, 'name'), fallback)
}

export function localizedSkillDescription(locale: AppLocale, skillId: string, fallback: string) {
  return pickTranslation(locale, skillMessageKey(skillId, 'description'), fallback)
}

export function localizedSkillParamLabel(
  locale: AppLocale,
  skillId: string,
  paramKey: string,
  fallback: string,
) {
  return pickTranslation(locale, skillMessageKey(skillId, `params.${paramKey}`), fallback)
}

export function localizedSkillSelectOption(
  locale: AppLocale,
  skillId: string,
  paramKey: string,
  optionIndex: number,
  fallback: string,
) {
  return pickTranslation(locale, skillMessageKey(skillId, `options.${paramKey}.${optionIndex}`), fallback)
}
