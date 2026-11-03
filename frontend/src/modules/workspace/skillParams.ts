import type { WorkspaceSkill, WorkspaceSkillParam } from '@/modules/workspace/api'
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'
import { localizedSkillParamLabel } from '@/modules/workspace/workspaceSkillI18n'

const SELECT_DEFAULT_INDEX: Record<string, number> = {
  depth: 1,
  difficulty: 1,
  questionType: 2,
}

function defaultNumberValue(param: WorkspaceSkillParam): string {
  const min = param.min ?? 1
  const max = param.max ?? min
  const preferred = Math.min(3, max)
  return String(Math.max(min, preferred))
}

export function buildSkillDefaults(skill: WorkspaceSkill, _locale = getAppLocale()): Record<string, string> {
  const values: Record<string, string> = {}
  for (const param of skill.params) {
    if (param.type === 'number') {
      values[param.key] = defaultNumberValue(param)
    } else if (param.type === 'select') {
      const index = SELECT_DEFAULT_INDEX[param.key]
      const option = index != null ? param.options?.[index] : undefined
      if (option) {
        values[param.key] = option
      }
    }
  }
  return values
}

export function normalizeSkillParams(
  skill: WorkspaceSkill,
  raw: Record<string, string>,
  locale = getAppLocale(),
): { params: Record<string, string> } | { error: string } {
  const params: Record<string, string> = {}
  for (const param of skill.params) {
    let value = (raw[param.key] ?? '').trim()
    const label = localizedSkillParamLabel(locale, skill.id, param.key, param.label)
    if (param.required && !value) {
      return { error: translateApp(locale, 'workspace.skillParamRequired', { label }) }
    }
    if (param.type === 'number') {
      if (!value) {
        value = defaultNumberValue(param)
      } else {
        const number = Number(value)
        const min = param.min ?? 1
        const max = param.max
        if (!Number.isFinite(number) || !Number.isInteger(number) || number < min || (max != null && number > max)) {
          const maxLabel = max != null ? String(max) : '∞'
          return {
            error: translateApp(locale, 'workspace.skillParamRange', {
              label,
              min,
              max: maxLabel,
            }),
          }
        }
        value = String(number)
      }
    }
    if (param.type === 'select' && value && param.options?.length && !param.options.includes(value)) {
      return { error: translateApp(locale, 'workspace.skillParamInvalid', { label }) }
    }
    params[param.key] = value
  }
  return { params }
}
