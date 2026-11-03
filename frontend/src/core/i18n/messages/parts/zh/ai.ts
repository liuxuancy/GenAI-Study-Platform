import type { MessageTree } from '@/core/i18n/types'

export const zhAi: MessageTree = {
  ai: {
    providerLabel: 'AI 创作模型',
    providerLoadFailed: '无法读取服务端 AI 模型配置。',
    providerNotConfigured: '服务端尚未配置可用的 AI 模型。',
    providerNotConfiguredSuffix: '（服务端未配置）',
    providers: {
      deepseek: 'DeepSeek',
      teamily: 'Teamily',
    },
    generate: {
      topic: '主题',
      topicRequired: '请输入课程主题',
      audience: '受众',
      language: '语言',
      chapterCountHint: '期望章节数',
      extraRequirements: '补充要求',
      audienceGeneral: '通用',
      audienceCs: '计算机专业',
      audienceNonCs: '非计算机专业',
      langZh: '简体中文',
      langEn: 'English',
      metaLatency: '耗时 {{ms}} ms',
      metaModel: '模型 {{model}}',
      metaFallback: '已使用备用模型',
      complete: '内容生成完成',
      warningJoin: '；',
      summaryJoin: ' · ',
    },
  },
}
