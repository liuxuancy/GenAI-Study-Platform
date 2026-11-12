import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'

import {
  bubbleLinks,
  findBubble,
  fitScale,
  mapBounds,
  parentBubbleId,
  placeBubbles,
} from '@/modules/student/bubbleLayout'
import type { BubbleNode } from '@/modules/student/bubbleTree'
import { bubbleWalkRoute } from '@/modules/student/bubbleWalk'
import { BubbleWalker } from '@/modules/student/BubbleWalker'
import type { CourseAvatar, PixelSprite } from '@/modules/student/PixelSprite'
import { completedBubbleIds } from '@/modules/student/courseTreeNav'
import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/modules/student/CourseBubbleMap.module.css'

const BASE_FONT = [18, 13.5, 11.5, 11]
const ROOT_ID = 'r'

function levelName(depth: number, t: (key: string) => string) {
  const keys = [
    'student.bubbleMap.levelCourse',
    'student.bubbleMap.levelChapter',
    'student.bubbleMap.levelSection',
    'student.bubbleMap.levelConcept',
  ] as const
  return t(keys[depth] ?? 'student.bubbleMap.levelFallback')
}

function closeOpenBelow(openIds: Set<string>, root: BubbleNode, id: string) {
  const node = findBubble(root, id)
  if (!node?.children?.length) {
    return
  }
  function walk(parentId: string, current: BubbleNode) {
    current.children?.forEach((child, index) => {
      const childId = `${parentId}.${index}`
      openIds.delete(childId)
      walk(childId, child)
    })
  }
  walk(id, node)
}

function depthClass(depth: number) {
  if (depth === 0) {
    return styles.nodeDepth0
  }
  if (depth === 1) {
    return styles.nodeDepth1
  }
  if (depth === 2) {
    return styles.nodeDepth2
  }
  return styles.nodeDepth3
}

export function CourseBubbleMap({
  tree,
  code,
  embedded,
  doneIds,
  affection = 0,
  sprite,
  dialogue,
}: {
  tree: BubbleNode
  code?: string
  embedded?: boolean
  doneIds?: ReadonlySet<string>
  affection?: number
  sprite?: PixelSprite | null
  dialogue?: CourseAvatar['dialogue']
}) {
  const { t } = useI18n()
  const stageRef = useRef<HTMLDivElement>(null)
  const canvasBoxRef = useRef('')
  const panRef = useRef({ down: false, sx: 0, sy: 0, left: 0, top: 0 })
  const [openIds, setOpenIds] = useState<ReadonlySet<string>>(() => new Set())
  const [focusId, setFocusId] = useState(ROOT_ID)
  const [stageSize, setStageSize] = useState({ width: 800, height: 600 })
  const [panning, setPanning] = useState(false)
  const [poppingId, setPoppingId] = useState<string | null>(null)
  const [hintVisible, setHintVisible] = useState(true)

  const completedIds = useMemo(() => doneIds ? completedBubbleIds(tree, doneIds) : new Set<string>(), [tree, doneIds])

  const layout = useMemo(() => {
    const bounds = mapBounds(placeBubbles(tree, openIds, 1))
    const padding = sprite ? 48 : 16
    const nextScale = fitScale(bounds, stageSize.width - (padding - 16) * 2, stageSize.height - (padding - 16) * 2)
    const bubbles = placeBubbles(tree, openIds, nextScale)
    const links = bubbleLinks(bubbles)
    return { bubbles, links, width: Math.ceil(bounds.width * nextScale) + padding * 2, height: Math.ceil(bounds.height * nextScale) + padding * 2, nextScale }
  }, [openIds, stageSize.height, stageSize.width, tree, sprite])

  const walkRoute = useMemo(() => bubbleWalkRoute(layout.bubbles, layout.links), [layout.bubbles, layout.links])

  useEffect(() => {
    const stage = stageRef.current
    if (!stage) {
      return
    }
    const observer = new ResizeObserver(() => {
      setStageSize({ width: stage.clientWidth, height: stage.clientHeight })
    })
    observer.observe(stage)
    setStageSize({ width: stage.clientWidth, height: stage.clientHeight })
    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    const stage = stageRef.current
    const box = `${layout.width}:${layout.height}`
    if (!stage || canvasBoxRef.current === box) {
      canvasBoxRef.current = box
      return
    }
    canvasBoxRef.current = box
    requestAnimationFrame(() => {
      const focused = layout.bubbles.find((bubble) => bubble.id === focusId)
      stage.scrollLeft = (stage.scrollWidth - stage.clientWidth) / 2 + (focused?.x ?? 0)
      stage.scrollTop = (stage.scrollHeight - stage.clientHeight) / 2 + (focused?.y ?? 0)
    })
  }, [layout.width, layout.height, layout.bubbles, focusId])

  const selectBubble = useCallback(
    (id: string) => {
      const node = findBubble(tree, id)
      if (!node) {
        return
      }
      setFocusId(id)
      setOpenIds((current) => {
        const next = new Set(current)
        if (node.children?.length) {
          if (next.has(id)) {
            closeOpenBelow(next, tree, id)
            next.delete(id)
          } else {
            next.add(id)
          }
        }
        return next
      })
      setPoppingId(id)
      window.setTimeout(() => setPoppingId((current) => (current === id ? null : current)), 480)
      if (hintVisible) {
        setHintVisible(false)
      }
    },
    [hintVisible, tree],
  )

  const collapseAll = useCallback(() => {
    setOpenIds(new Set())
    setFocusId(ROOT_ID)
  }, [])

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key !== 'Escape') {
        return
      }
      setOpenIds((current) => {
        if (current.size === 0) {
          return current
        }
        let deepest: string | null = null
        for (const id of current) {
          if (!deepest || id.split('.').length > deepest.split('.').length) {
            deepest = id
          }
        }
        if (!deepest) {
          return current
        }
        const next = new Set(current)
        closeOpenBelow(next, tree, deepest)
        next.delete(deepest)
        setFocusId(parentBubbleId(deepest) ?? ROOT_ID)
        return next
      })
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [tree])

  useEffect(() => {
    function onPointerMove(event: PointerEvent) {
      if (!panRef.current.down || !stageRef.current) {
        return
      }
      const stage = stageRef.current
      stage.scrollLeft = panRef.current.left - (event.clientX - panRef.current.sx)
      stage.scrollTop = panRef.current.top - (event.clientY - panRef.current.sy)
    }
    function onPointerUp() {
      panRef.current.down = false
      setPanning(false)
    }
    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
    return () => {
      window.removeEventListener('pointermove', onPointerMove)
      window.removeEventListener('pointerup', onPointerUp)
    }
  }, [])

  const deepestLevel = useMemo(() => {
    let depth = 0
    for (const bubble of layout.bubbles) {
      depth = Math.max(depth, bubble.depth)
    }
    return depth + 1
  }, [layout.bubbles])

  return (
    <div className={`${styles.root}${embedded ? ` ${styles.rootEmbedded}` : ''}`}>
      <div
        className={`${styles.stage}${panning ? ` ${styles.stagePanning}` : ''}`}
        ref={stageRef}
        onPointerDown={(event) => {
          if (event.target instanceof Element && event.target.closest('button')) {
            return
          }
          if (!stageRef.current) {
            return
          }
          panRef.current = {
            down: true,
            sx: event.clientX,
            sy: event.clientY,
            left: stageRef.current.scrollLeft,
            top: stageRef.current.scrollTop,
          }
          setPanning(true)
        }}
      >
        <div className={styles.canvas} style={{ width: layout.width, height: layout.height }}>
          {layout.links.map((link) => (
            <span
              key={link.id}
              className={styles.link}
              style={{
                transform: `translate(${link.x}px, ${link.y}px) rotate(${link.angle}deg)`,
                width: `${link.length}px`,
              }}
            />
          ))}
          {layout.bubbles.map((bubble) => {
            const hasChildren = Boolean(bubble.node.children?.length)
            const isOpen = openIds.has(bubble.id) && hasChildren
            const completed = completedIds.has(bubble.id)
            const ariaLabel = `${levelName(bubble.depth, t)}：${bubble.node.label}${hasChildren ? '' : t('student.bubbleMap.noChildren')}${completed ? t('student.bubbleMap.completed') : ''}`
            return (
              <button
                key={bubble.id}
                type="button"
                className={[
                  styles.node,
                  depthClass(bubble.depth),
                  isOpen ? styles.nodeOpen : '',
                  completed ? styles.nodeCompleted : '',
                  poppingId === bubble.id ? styles.nodePopping : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
                aria-label={ariaLabel}
                aria-expanded={hasChildren ? isOpen : undefined}
                style={{
                  width: bubble.size,
                  height: bubble.size,
                  transform: `translate(-50%, -50%) translate(${bubble.x}px, ${bubble.y}px)`,
                }}
                onClick={() => selectBubble(bubble.id)}
              >
                <span className={styles.bubble}>
                  {bubble.depth === 0 && code ? <span className={styles.codeTag}>{code}</span> : null}
                  <BubbleLabel depth={bubble.depth} scale={layout.nextScale} text={bubble.node.label} />
                </span>
              </button>
            )
          })}
          {sprite ? <BubbleWalker sprite={sprite} route={walkRoute} bubbles={layout.bubbles} dialogue={dialogue ?? null} showHearts={affection >= 100} /> : null}
        </div>
      </div>

      <div className={`${styles.hud} ${styles.hudTr}`}>
        <div className={styles.meter}>
          <span>
            {t('student.bubbleMap.depth')} <b>{deepestLevel}</b> / 4
          </span>
          <span>{t('student.bubbleMap.bubbleCount', { count: layout.bubbles.length })}</span>
        </div>
        <button className={styles.ghost} type="button" disabled={openIds.size === 0} onClick={collapseAll}>
          {t('student.bubbleMap.collapseAll')}
        </button>
      </div>

      <div className={`${styles.hud} ${styles.hudBr}`}>
        <p className={`${styles.hint}${hintVisible ? '' : ` ${styles.hintGone}`}`} hidden={!hintVisible}>
          {t('student.bubbleMap.hint')}
        </p>
      </div>
    </div>
  )
}

function BubbleLabel({ depth, scale, text }: { depth: number; scale: number; text: string }) {
  const labelRef = useRef<HTMLSpanElement>(null)
  useLayoutEffect(() => {
    const label = labelRef.current
    if (!label) {
      return
    }
    let size = (BASE_FONT[depth] ?? BASE_FONT[BASE_FONT.length - 1]) * Math.max(scale, 0.86)
    label.style.fontSize = `${size}px`
    let guard = 0
    while (label.scrollHeight > label.clientHeight + 0.5 && size > 7.2 && guard < 40) {
      size -= 0.4
      label.style.fontSize = `${size}px`
      guard += 1
    }
  }, [depth, scale, text])
  return (
    <span ref={labelRef} className={styles.label}>
      {text}
    </span>
  )
}
