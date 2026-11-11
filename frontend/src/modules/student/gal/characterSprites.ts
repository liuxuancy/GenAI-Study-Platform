import type { GalEmotion } from '@/modules/student/gal/galTypes'

/** 立绘扩展名：AI 出图后把 PNG 放进 public/gal/ 并改为 `.png`。 */
const spriteExt = 'png'

const base = '/gal'

const emotions: GalEmotion[] = ['normal', 'happy', 'angry', 'surprised', 'shy', 'proud']

export const characterSprites: Record<GalEmotion, string> = Object.fromEntries(
  emotions.map((emotion) => [emotion, `${base}/${emotion}.${spriteExt}`]),
) as Record<GalEmotion, string>

export function spriteForEmotion(emotion: string) {
  const key = emotion.toLowerCase() as GalEmotion
  return characterSprites[key] ?? characterSprites.normal
}
