export type GalEmotion = 'normal' | 'happy' | 'angry' | 'surprised' | 'shy' | 'proud'

export type GalScriptItem =
  | {
      kind: 'dialogue'
      speaker: string
      text: string
      emotion: GalEmotion
      note?: string
      /** Scene key matching `frontend/public/gal/bg/<name>.png` (optional API field). */
      background?: string
    }
  | {
      kind: 'choice'
      prompt: string
      options: string[]
      correctIndex: number
      feedbackCorrect: string
      feedbackWrong: string
      emotionOnCorrect?: GalEmotion
      emotionOnWrong?: GalEmotion
    }

/** @deprecated use GalScriptItem dialogue shape */
export interface GalDialogueLine {
  speaker: string
  text: string
  emotion: GalEmotion
  note?: string
}

export interface GalScriptResponse {
  title: string
  chapterKey?: string
  nextChapterKey?: string | null
  script: GalScriptItem[]
}

export type GalDetailLevel = 'brief' | 'detailed' | 'academic'
export type GalPersonality = 'tsundere' | 'gentle' | 'strict'

export interface GalGameSettings {
  detailLevel: GalDetailLevel
  personality: GalPersonality
}

export const defaultGalSettings: GalGameSettings = {
  detailLevel: 'detailed',
  personality: 'gentle',
}

export function normalizeGalScriptItem(raw: Record<string, unknown>): GalScriptItem | null {
  const kind = raw.kind
  if (kind === 'choice' || (typeof raw.prompt === 'string' && Array.isArray(raw.options))) {
    const options = raw.options as unknown[]
    if (typeof raw.prompt !== 'string' || !Array.isArray(options)) {
      return null
    }
    return {
      kind: 'choice',
      prompt: raw.prompt,
      options: options.filter((item): item is string => typeof item === 'string'),
      correctIndex: typeof raw.correctIndex === 'number' ? raw.correctIndex : 0,
      feedbackCorrect: typeof raw.feedbackCorrect === 'string' ? raw.feedbackCorrect : '',
      feedbackWrong: typeof raw.feedbackWrong === 'string' ? raw.feedbackWrong : '',
      emotionOnCorrect: raw.emotionOnCorrect as GalEmotion | undefined,
      emotionOnWrong: raw.emotionOnWrong as GalEmotion | undefined,
    }
  }
  if (typeof raw.text === 'string') {
    return {
      kind: 'dialogue',
      speaker: typeof raw.speaker === 'string' ? raw.speaker : '',
      text: raw.text,
      emotion: (typeof raw.emotion === 'string' ? raw.emotion : 'normal') as GalEmotion,
      note: typeof raw.note === 'string' ? raw.note : undefined,
      background: typeof raw.background === 'string' ? raw.background : undefined,
    }
  }
  return null
}

export function mapGalScriptFromApi(script: unknown[]): GalScriptItem[] {
  return script
    .map((item) => (item && typeof item === 'object' ? normalizeGalScriptItem(item as Record<string, unknown>) : null))
    .filter((item): item is GalScriptItem => item !== null)
}
