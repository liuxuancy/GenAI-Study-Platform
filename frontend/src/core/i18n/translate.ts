import type { AppLocale, MessageTree } from '@/core/i18n/types'

export function resolveMessage(tree: MessageTree, key: string): string | undefined {
  const parts = key.split('.')
  let node: string | MessageTree | undefined = tree
  for (const part of parts) {
    if (!node || typeof node === 'string') {
      return undefined
    }
    node = node[part] as string | MessageTree | undefined
  }
  return typeof node === 'string' ? node : undefined
}

export function formatMessage(template: string, params?: Record<string, string | number>) {
  if (!params) {
    return template
  }
  return template.replace(/\{\{(\w+)\}\}/g, (_, name: string) => {
    const value = params[name]
    return value === undefined ? '' : String(value)
  })
}

export function translate(
  tree: MessageTree,
  key: string,
  params?: Record<string, string | number>,
  fallbackTree?: MessageTree,
) {
  const primary = resolveMessage(tree, key)
  if (primary !== undefined) {
    return formatMessage(primary, params)
  }
  if (fallbackTree) {
    const fallback = resolveMessage(fallbackTree, key)
    if (fallback !== undefined) {
      return formatMessage(fallback, params)
    }
  }
  return key
}

export function applyDocumentLocale(locale: AppLocale) {
  if (typeof document === 'undefined') {
    return
  }
  document.documentElement.lang = locale === 'zh' ? 'zh-CN' : 'en'
}
