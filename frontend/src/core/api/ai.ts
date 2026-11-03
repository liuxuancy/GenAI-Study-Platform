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
  outlineOnly?: boolean
}

export interface AiGenerationMeta {
  scene: string
  templateId: string
  templateVersion: string
  modelRequested: string
  modelUsed: string
  fallbackUsed: boolean
  latencyMs: number
  warnings: string[]
}

export interface AiGenerationResult<T> {
  course: T
  meta: AiGenerationMeta
}
