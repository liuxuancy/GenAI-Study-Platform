import { Alert } from 'antd'
import { useEffect, useState } from 'react'

import type { AiGenerationMeta } from '@/core/api/ai'
import { useI18n } from '@/core/i18n/useI18n'

export function GenerationMetaAlert({ meta }: { meta: AiGenerationMeta }) {
  const { t } = useI18n()
  const [visibleMeta, setVisibleMeta] = useState<AiGenerationMeta | null>(meta)
  const warning = meta.warnings.length > 0 || meta.fallbackUsed

  useEffect(() => {
    setVisibleMeta(meta)
    if (meta.warnings.length > 0 || meta.fallbackUsed) return
    const timer = window.setTimeout(() => setVisibleMeta(null), 2000)
    return () => window.clearTimeout(timer)
  }, [meta])

  if (visibleMeta !== meta) return null
  const details = [t('ai.generate.metaLatency', { ms: meta.latencyMs })]
  if (meta.modelUsed) details.push(t('ai.generate.metaModel', { model: meta.modelUsed }))
  if (meta.fallbackUsed) details.push(t('ai.generate.metaFallback'))
  return (
    <Alert
      type={warning ? 'warning' : 'success'}
      showIcon
      closable={warning}
      onClose={() => setVisibleMeta(null)}
      message={details.join(t('ai.generate.summaryJoin'))}
      description={meta.warnings.length ? meta.warnings.join(t('ai.generate.warningJoin')) : t('ai.generate.complete')}
      style={{ marginBottom: 16 }}
    />
  )
}
