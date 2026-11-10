import styles from '@/modules/student/PixelSprite.module.css'

export interface PixelSprite {
  palette: string[]
  pixels: string[]
}

export interface CourseAvatar {
  roleType: string
  bodyType: string
  hairColor: string
  hairStyle: string
  facialFeature: string
  accessory: string
  silhouetteKey: string
  personality: string
  dialogue: {
    zh: { touch: string; rest: string }
    en: { touch: string; rest: string }
  } | null
  sprites: {
    base: PixelSprite
    '25': PixelSprite
    '50': PixelSprite
    '75': PixelSprite
    '100': PixelSprite
  }
}

export function spriteForAffection(avatar: CourseAvatar, affection: number): PixelSprite {
  if (affection >= 100) return avatar.sprites['100']
  if (affection >= 75) return avatar.sprites['75']
  if (affection >= 50) return avatar.sprites['50']
  if (affection >= 25) return avatar.sprites['25']
  return avatar.sprites.base
}

export function PixelSpriteView({ sprite, size = 112 }: { sprite: PixelSprite; size?: number }) {
  const cells = sprite.pixels.flatMap((row, y) =>
    [...row].flatMap((symbol, x) => {
      if (symbol === '.') return []
      const index = Number.parseInt(symbol, 16)
      const color = sprite.palette[index]
      if (!color) return []
      return [<rect key={`${x}-${y}`} x={x} y={y} width={1} height={1} fill={color} />]
    }),
  )

  return (
    <svg
      className={styles.sprite}
      width={size}
      height={size}
      viewBox="0 0 16 16"
      aria-hidden="true"
    >
      {cells}
    </svg>
  )
}
