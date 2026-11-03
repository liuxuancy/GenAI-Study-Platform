import type { MessageTree } from '@/core/i18n/types'

export function mergeMessages(...parts: MessageTree[]): MessageTree {
  const result: MessageTree = {}
  for (const part of parts) {
    for (const key of Object.keys(part)) {
      const incoming = part[key]
      const existing = result[key]
      if (
        incoming &&
        typeof incoming === 'object' &&
        existing &&
        typeof existing === 'object' &&
        typeof existing !== 'string'
      ) {
        result[key] = mergeMessages(existing as MessageTree, incoming as MessageTree)
      } else {
        result[key] = incoming
      }
    }
  }
  return result
}
