import { useEffect, useMemo, useState } from 'react'
import { Select } from 'antd'
import { getAiProviderOptions, type AiGenerateProvider } from '@/core/api/ai'

const labels: Record<AiGenerateProvider, string> = {
  deepseek: 'DeepSeek',
  teamily: 'Teamily',
}

export function useAiProviderSelect() {
  const [provider, setProvider] = useState<AiGenerateProvider>('deepseek')
  const [loading, setLoading] = useState(true)
  const [available, setAvailable] = useState<{ deepseek: boolean; teamily: boolean }>({
    deepseek: false,
    teamily: false,
  })

  useEffect(() => {
    let cancelled = false
    getAiProviderOptions()
      .then((options) => {
        if (cancelled) {
          return
        }
        setAvailable({
          deepseek: options.providers.deepseek,
          teamily: options.providers.teamily,
        })
        const fallback: AiGenerateProvider = options.providers.deepseek
          ? 'deepseek'
          : options.providers.teamily
            ? 'teamily'
            : 'deepseek'
        const initial =
          options.default === 'deepseek' || options.default === 'teamily' ? options.default : fallback
        setProvider(
          options.providers[initial] ? initial : fallback,
        )
      })
      .catch(() => {
        if (!cancelled) {
          setAvailable({ deepseek: true, teamily: true })
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [])

  const options = useMemo(() => {
    const items: { value: AiGenerateProvider; label: string; disabled?: boolean }[] = []
    if (available.deepseek) {
      items.push({ value: 'deepseek', label: labels.deepseek })
    }
    if (available.teamily) {
      items.push({ value: 'teamily', label: labels.teamily })
    }
    if (items.length === 0) {
      items.push(
        { value: 'deepseek', label: labels.deepseek, disabled: true },
        { value: 'teamily', label: labels.teamily, disabled: true },
      )
    }
    return items
  }, [available.deepseek, available.teamily])

  const selector = (
    <Select<AiGenerateProvider>
      size="small"
      loading={loading}
      style={{ minWidth: 140 }}
      value={provider}
      options={options}
      onChange={setProvider}
    />
  )

  return { provider, selector, loading, optionsAvailable: available.deepseek || available.teamily }
}
