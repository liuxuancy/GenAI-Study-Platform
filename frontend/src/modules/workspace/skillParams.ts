import type { WorkspaceSkill, WorkspaceSkillParam } from '@/modules/workspace/api'

const SELECT_DEFAULTS: Record<string, string> = {
  depth: '标准',
  difficulty: '中等',
  questionType: '混合',
}

function defaultNumberValue(param: WorkspaceSkillParam): string {
  const min = param.min ?? 1
  const max = param.max ?? min
  const preferred = Math.min(3, max)
  return String(Math.max(min, preferred))
}

export function buildSkillDefaults(skill: WorkspaceSkill): Record<string, string> {
  const values: Record<string, string> = {}
  for (const param of skill.params) {
    if (param.type === 'number') {
      values[param.key] = defaultNumberValue(param)
    } else if (param.type === 'select') {
      const preset = SELECT_DEFAULTS[param.key]
      if (preset && param.options?.includes(preset)) {
        values[param.key] = preset
      }
    }
  }
  return values
}

export function normalizeSkillParams(
  skill: WorkspaceSkill,
  raw: Record<string, string>,
): { params: Record<string, string> } | { error: string } {
  const params: Record<string, string> = {}
  for (const param of skill.params) {
    let value = (raw[param.key] ?? '').trim()
    if (param.required && !value) {
      return { error: `请填写${param.label}` }
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
          return { error: `${param.label}需在 ${min}–${maxLabel} 之间的整数` }
        }
        value = String(number)
      }
    }
    if (param.type === 'select' && value && param.options?.length && !param.options.includes(value)) {
      return { error: `${param.label}无效` }
    }
    params[param.key] = value
  }
  return { params }
}
