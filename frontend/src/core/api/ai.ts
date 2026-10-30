import { http } from '@/core/http'

export type AiGenerateProvider = 'deepseek' | 'teamily'

export interface AiProviderOptions {
  default: AiGenerateProvider
  providers: {
    deepseek: boolean
    teamily: boolean
  }
}

export function getAiProviderOptions() {
  return http.get<AiProviderOptions>('/ai/providers')
}

export interface GenerateCourseRequest {
  provider?: AiGenerateProvider
  topic?: string
  audience?: string
  language?: string
  extraRequirements?: string
  chapterCountHint?: number
}
