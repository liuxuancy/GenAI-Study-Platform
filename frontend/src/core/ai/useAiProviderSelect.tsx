import { useEffect, useMemo, useState } from 'react'
import { Alert, Select, Space, Typography } from 'antd'
import { getAiProviderOptions, type AiGenerateProvider } from '@/core/api/ai'
import { useI18n } from '@/core/i18n/useI18n'

const storageKey = 'caveclass.ai.provider'
const courseStorageKey = 'caveclass.ai.courseAvatarProvider'

function savedProvider(courseAvatar: boolean): AiGenerateProvider | null {
  const value = window.localStorage.getItem(courseAvatar ? courseStorageKey : storageKey)
    ?? (courseAvatar ? window.localStorage.getItem(storageKey) : null)
  return value === 'deepseek' || value === 'teamily' ? value : null
}

export function useAiProviderSelect({ courseAvatar = false }: { courseAvatar?: boolean } = {}) {
  const { t } = useI18n()
  const [provider, setProviderState] = useState<AiGenerateProvider>(() => savedProvider(courseAvatar) ?? 'deepseek')
  const [loading, setLoading] = useState(true)
  const [available, setAvailable] = useState<Record<AiGenerateProvider, boolean>>({
    deepseek: false,
    teamily: false,
  })
  const [loadError, setLoadError] = useState('')

  function setProvider(next: AiGenerateProvider) {
    setProviderState(next)
    window.localStorage.setItem(courseAvatar ? courseStorageKey : storageKey, next)
  }

  useEffect(() => {
    let cancelled = false
    setLoading(true)
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
        const preferred = savedProvider(courseAvatar)
        const configuredDefault =
          options.default === 'deepseek' || options.default === 'teamily' ? options.default : fallback
        const initial = preferred && options.providers[preferred] ? preferred : configuredDefault
        setProvider(options.providers[initial] ? initial : fallback)
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setAvailable({ deepseek: false, teamily: false })
          setLoadError(error.message || t('ai.providerLoadFailed'))
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
  }, [t, courseAvatar])

  const options = useMemo(() => {
    const providers: AiGenerateProvider[] = ['deepseek', 'teamily']
    return providers.map((value) => ({
      value,
      label: available[value]
        ? t(`ai.providers.${value}`)
        : `${t(`ai.providers.${value}`)}${t('ai.providerNotConfiguredSuffix')}`,
      disabled: !available[value],
    }))
  }, [available.deepseek, available.teamily, t])

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
    options,
    setProvider,
    loading,
    available,
    error: loadError,
    optionsAvailable: available.deepseek || available.teamily,
  }
}

export function AiProviderControl({ compact = false }: { compact?: boolean }) {
  const { t } = useI18n()
  const selection = useAiProviderSelect()
  return (
    <Space direction="vertical" size={4}>
      <Space>
        {!compact ? <Typography.Text strong>{t('ai.providerLabel')}</Typography.Text> : null}
        {selection.selector}
      </Space>
      {selection.error ? <Alert type="error" showIcon message={selection.error} /> : null}
      {!selection.loading && !selection.error && !selection.optionsAvailable ? (
        <Alert type="warning" showIcon message={t('ai.providerNotConfigured')} />
      ) : null}
    </Space>
  )
}
