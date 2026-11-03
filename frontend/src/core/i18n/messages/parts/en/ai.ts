import type { MessageTree } from '@/core/i18n/types'

export const enAi: MessageTree = {
  ai: {
    providerLabel: 'AI model',
    providerLoadFailed: 'Could not load AI model settings from the server.',
    providerNotConfigured: 'No AI models are configured on the server.',
    providerNotConfiguredSuffix: ' (not configured on server)',
    providers: {
      deepseek: 'DeepSeek',
      teamily: 'Teamily',
    },
    generate: {
      topic: 'Topic',
      topicRequired: 'Enter a course topic.',
      audience: 'Audience',
      language: 'Language',
      chapterCountHint: 'Expected chapters',
      extraRequirements: 'Additional requirements',
      audienceGeneral: 'General',
      audienceCs: 'Computer science',
      audienceNonCs: 'Non-CS',
      langZh: 'Simplified Chinese',
      langEn: 'English',
      metaLatency: '{{ms}} ms elapsed',
      metaModel: 'Model {{model}}',
      metaFallback: 'Fallback model used',
      complete: 'Content generated',
      warningJoin: '; ',
      summaryJoin: ' · ',
    },
  },
}
