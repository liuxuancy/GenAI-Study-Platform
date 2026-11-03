import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'

import {
  ancestorIds,
  bubbleLinks,
  findBubble,
  fitScale,
  mapExtent,
  parentBubbleId,
  placeBubbles,
} from '@/modules/student/bubbleLayout'
import type { BubbleNode } from '@/modules/student/bubbleTree'
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

function trimNote(text: string, max = 320) {
  const clean = text.replace(/\s+/g, ' ').trim()
  if (clean.length <= max) {
    return clean
  }
  return `${clean.slice(0, max - 1)}…`
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
}: {
  tree: BubbleNode
  code?: string
  embedded?: boolean
}) {
  const { t } = useI18n()
  const stageRef = useRef<HTMLDivElement>(null)
  const canvasBoxRef = useRef<number>(0)
  const panRef = useRef({ down: false, sx: 0, sy: 0, left: 0, top: 0 })
  const [openIds, setOpenIds] = useState<ReadonlySet<string>>(() => new Set())
  const [focusId, setFocusId] = useState(ROOT_ID)
  const [scale, setScale] = useState(1)
  const [stageSize, setStageSize] = useState({ width: 800, height: 600 })
  const [panning, setPanning] = useState(false)
  const [poppingId, setPoppingId] = useState<string | null>(null)
  const [hintVisible, setHintVisible] = useState(true)

  const focusNode = useMemo(() => findBubble(tree, focusId) ?? tree, [focusId, tree])

  const layout = useMemo(() => {
    const extent = mapExtent(tree, openIds)
    const nextScale = fitScale(extent, stageSize.width, stageSize.height)
    const bubbles = placeBubbles(tree, openIds, nextScale)
    const links = bubbleLinks(bubbles)
    const box = Math.ceil(extent * 2 * nextScale) + 8
    return { bubbles, links, box, nextScale }
  }, [openIds, stageSize.height, stageSize.width, tree])

  useLayoutEffect(() => {
    setScale(layout.nextScale)
  }, [layout.nextScale])

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
    if (!stage || canvasBoxRef.current === layout.box) {
      canvasBoxRef.current = layout.box
      return
    }
    canvasBoxRef.current = layout.box
    requestAnimationFrame(() => {
      stage.scrollLeft = (stage.scrollWidth - stage.clientWidth) / 2
      stage.scrollTop = (stage.scrollHeight - stage.clientHeight) / 2
    })
  }, [layout.box])

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

  const collapseTo = useCallback(
    (id: string) => {
      setOpenIds((current) => {
        const next = new Set(current)
        closeOpenBelow(next, tree, id)
        next.delete(id)
        return next
      })
      setFocusId(id)
    },
    [tree],
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

  const breadcrumb = ancestorIds(focusId).slice(1)

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
        <div className={styles.canvas} style={{ width: layout.box, height: layout.box }}>
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
            const ariaLabel = `${levelName(bubble.depth, t)}：${bubble.node.label}${hasChildren ? '' : t('student.bubbleMap.noChildren')}`
            return (
              <button
                key={bubble.id}
                type="button"
                className={[
                  styles.node,
                  depthClass(bubble.depth),
                  isOpen ? styles.nodeOpen : '',
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
                  <BubbleLabel depth={bubble.depth} scale={scale} text={bubble.node.label} />
                </span>
              </button>
            )
          })}
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

      <footer className={`${styles.hud} ${styles.hudBl}`}>
        {breadcrumb.length > 0 ? (
          <nav className={styles.crumb} aria-label={t('student.bubbleMap.pathAria')}>
            {breadcrumb.map((id, index) => {
              const node = findBubble(tree, id)
              if (!node) {
                return null
              }
              const label = index === 0 && code ? code : node.label
              return (
                <span key={id}>
                  {index > 0 ? <span className={styles.crumbSep}>/</span> : null}
                  <button type="button" title={t('student.bubbleMap.collapseTo', { label: node.label })} onClick={() => collapseTo(id)}>
                    {label}
                  </button>
                </span>
              )
            })}
          </nav>
        ) : null}
        <p className={styles.note}>
          <b>{focusNode.label}</b> — {trimNote(focusNode.note)}
        </p>
      </footer>

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
    let size = BASE_FONT[depth] * Math.max(scale, 0.86)
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
