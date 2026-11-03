import { useEffect, useMemo, useState } from 'react'
import { Alert, Select, Space, Typography } from 'antd'
import { getAiProviderOptions, type AiGenerateProvider } from '@/core/api/ai'

const labels: Record<AiGenerateProvider, string> = {
  deepseek: 'DeepSeek',
  teamily: 'Teamily',
}

const storageKey = 'caveclass.ai.provider'

function savedProvider(): AiGenerateProvider | null {
  const value = window.localStorage.getItem(storageKey)
  return value === 'deepseek' || value === 'teamily' ? value : null
}

export function useAiProviderSelect() {
  const [provider, setProviderState] = useState<AiGenerateProvider>(() => savedProvider() ?? 'deepseek')
  const [loading, setLoading] = useState(true)
  const [available, setAvailable] = useState<{ deepseek: boolean; teamily: boolean }>({
    deepseek: false,
    teamily: false,
  })
  const [loadError, setLoadError] = useState('')

  function setProvider(next: AiGenerateProvider) {
    setProviderState(next)
    window.localStorage.setItem(storageKey, next)
  }

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
        setLoadError('')
        const fallback: AiGenerateProvider = options.providers.deepseek
          ? 'deepseek'
          : options.providers.teamily
            ? 'teamily'
            : 'deepseek'
        const preferred = savedProvider()
        const configuredDefault =
          options.default === 'deepseek' || options.default === 'teamily' ? options.default : fallback
        const initial = preferred && options.providers[preferred] ? preferred : configuredDefault
        setProvider(options.providers[initial] ? initial : fallback)
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setAvailable({ deepseek: false, teamily: false })
          setLoadError(error.message || '无法读取服务端 AI 模型配置。')
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
    return (Object.keys(labels) as AiGenerateProvider[]).map((value) => ({
      value,
      label: available[value] ? labels[value] : `${labels[value]}（服务端未配置）`,
      disabled: !available[value],
    }))
  }, [available.deepseek, available.teamily])

  const selector = (
    <Select<AiGenerateProvider>
      size="small"
      loading={loading}
      value={provider}
      options={options}
      onChange={setProvider}
      disabled={!available.deepseek && !available.teamily}
      popupMatchSelectWidth={false}
    />
  )

  return {
    provider,
    selector,
    loading,
    available,
    error: loadError,
    optionsAvailable: available.deepseek || available.teamily,
  }
}

export function AiProviderControl({ compact = false }: { compact?: boolean }) {
  const selection = useAiProviderSelect()
  return (
    <Space direction="vertical" size={4}>
      <Space>
        {!compact ? <Typography.Text strong>AI 创作模型</Typography.Text> : null}
        {selection.selector}
      </Space>
      {selection.error ? <Alert type="error" showIcon message={selection.error} /> : null}
      {!selection.loading && !selection.error && !selection.optionsAvailable ? (
        <Alert type="warning" showIcon message="服务端尚未配置可用的 AI 模型。" />
      ) : null}
    </Space>
  )
}
