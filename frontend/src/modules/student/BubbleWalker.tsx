import { useEffect, useLayoutEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'

import { useI18n } from '@/core/i18n/useI18n'
import type { PlacedBubble } from '@/modules/student/bubbleLayout'
import { nearestWalkPlacement, pointOnWalkRoute, remapWalkPlacement, type Point, type WalkApproach, type WalkRoute } from '@/modules/student/bubbleWalk'
import { PixelSpriteView, type CourseAvatar, type PixelSprite } from '@/modules/student/PixelSprite'
import { advanceWalker, canShowWalkerDialogue, registerWalkerClick, scheduleDialogueExpiry, type DialogueKind, type WalkerPhase } from '@/modules/student/walkerBehavior'
import styles from '@/modules/student/CourseBubbleMap.module.css'

const walkSpeed = 18
const spriteSize = 32
const dragThreshold = 5

function randomDuration(min: number, max: number) {
  return min + Math.random() * (max - min)
}

interface DragState {
  pointerId: number
  startX: number
  startY: number
  offset: Point
  dragging: boolean
}

interface WalkerState {
  point: Point
  distance: number
  direction: 1 | -1
  approach: WalkApproach | null
  approachDistance: number
  phase: WalkerPhase
  remaining: number
  drag: DragState | null
}

function canvasPoint(canvas: HTMLElement, clientX: number, clientY: number): Point {
  const rect = canvas.getBoundingClientRect()
  return {
    x: (clientX - rect.left) * canvas.clientWidth / rect.width - canvas.clientWidth / 2,
    y: (clientY - rect.top) * canvas.clientHeight / rect.height - canvas.clientHeight / 2,
  }
}

export function BubbleWalker({ sprite, route, bubbles, dialogue }: {
  sprite: PixelSprite
  route: WalkRoute
  bubbles: readonly PlacedBubble[]
  dialogue: CourseAvatar['dialogue']
}) {
  const { locale, t } = useI18n()
  const walkerRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const bubbleTimer = useRef<ReturnType<typeof setTimeout>>()
  const suppressClick = useRef(false)
  const bubbleKind = useRef<DialogueKind | null>(null)
  const clicks = useRef<number[]>([])
  const dialogueRef = useRef(dialogue)
  const geometryRef = useRef({ route, bubbles })
  const previousGeometryRef = useRef({ route, bubbles })
  const stateRef = useRef<WalkerState>({
    point: route.start, distance: 0, direction: 1, approach: null, approachDistance: 0,
    phase: 'walking', remaining: randomDuration(15000, 30000), drag: null,
  })
  dialogueRef.current = dialogue
  geometryRef.current = { route, bubbles }
  const [bubble, setBubble] = useState<DialogueKind | null>(null)
  const [dragging, setDragging] = useState(false)

  function showBubble(kind: DialogueKind) {
    if (!canShowWalkerDialogue(bubbleKind.current, kind, Boolean(dialogueRef.current))) return
    clearTimeout(bubbleTimer.current)
    bubbleKind.current = kind
    setBubble(kind)
    bubbleTimer.current = scheduleDialogueExpiry(() => {
      bubbleKind.current = null
      setBubble(null)
    })
  }

  function place(point: Point) {
    const walker = walkerRef.current
    if (!walker) return
    stateRef.current.point = point
    walker.style.transform = `translate(${point.x}px, ${point.y}px) translate(-50%, -100%)`
    const width = walker.parentElement?.clientWidth ?? 0
    const height = walker.parentElement?.clientHeight ?? 0
    const x = width / 2 + point.x
    const shift = Math.max(0, 90 - x) - Math.max(0, x + 90 - width)
    walker.style.setProperty('--dialogue-shift', `${shift}px`)
    walker.classList.toggle(styles.walkerDialogueBelow, height / 2 + point.y < 100)
  }

  function finishDrag() {
    const state = stateRef.current
    const drag = state.drag
    if (!drag) return
    state.drag = null
    setDragging(false)
    if (drag.dragging) {
      state.direction = Math.random() < 0.5 ? 1 : -1
      const { route: currentRoute, bubbles: currentBubbles } = geometryRef.current
      const placement = nearestWalkPlacement(currentRoute, currentBubbles, state.point, state.direction)
      state.distance = placement.distance
      state.approach = placement.approach
      state.approachDistance = 0
      place(placement.point)
      clicks.current = []
      suppressClick.current = true
    }
    if (buttonRef.current?.hasPointerCapture(drag.pointerId)) {
      buttonRef.current.releasePointerCapture(drag.pointerId)
    }
  }

  useEffect(() => () => clearTimeout(bubbleTimer.current), [])

  useLayoutEffect(() => {
    const walker = walkerRef.current
    if (!walker) return
    const state = stateRef.current
    const previousGeometry = previousGeometryRef.current
    previousGeometryRef.current = { route, bubbles }
    const motion = window.matchMedia('(prefers-reduced-motion: reduce)')
    let frame = 0
    let previous: number | null = null

    const wasDragging = state.drag?.dragging ?? false
    if (state.drag) finishDrag()
    if (!wasDragging) {
      const placement = remapWalkPlacement(previousGeometry.route, previousGeometry.bubbles, route, bubbles,
        state.point, state.direction, state.distance, state.approach !== null)
      state.distance = placement.distance
      state.approach = placement.approach
      state.approachDistance = 0
      place(placement.point)
    }

    function tick(time: number) {
      if (state.drag) {
        previous = null
        walker?.classList.remove(styles.walkerMoving)
      } else {
        if (previous !== null) {
          const elapsed = Math.min(time - previous, 1000)
          const next = advanceWalker(state.phase, state.remaining, elapsed, (phase) =>
            phase === 'resting' ? randomDuration(2000, 5000) : randomDuration(15000, 30000))
          let movement = next.walkingTime * walkSpeed / 1000
          state.phase = next.phase
          state.remaining = next.remaining
          if (state.approach) {
            const approach = state.approach
            const step = Math.min(movement, approach.length - state.approachDistance)
            state.approachDistance += step
            movement -= step
            const angle = approach.angle + approach.sweep * state.approachDistance / approach.length
            place({ x: approach.center.x + Math.cos(angle) * approach.radius, y: approach.center.y + Math.sin(angle) * approach.radius })
            if (state.approachDistance >= approach.length) {
              state.distance = approach.targetDistance
              state.approach = null
            }
          }
          if (!state.approach) {
            state.distance += state.direction * movement
            place(pointOnWalkRoute(route, state.distance))
          }
          if (next.beganRest) showBubble('rest')
        }
        previous = time
        walker?.classList.toggle(styles.walkerMoving, state.phase === 'walking')
      }
      frame = requestAnimationFrame(tick)
    }

    function updateAnimation() {
      cancelAnimationFrame(frame)
      previous = null
      if (document.hidden && state.drag) finishDrag()
      const active = route.length > 0 && !motion.matches && !document.hidden
      walker?.classList.toggle(styles.walkerMoving, active && !state.drag && state.phase === 'walking')
      if (active) frame = requestAnimationFrame(tick)
    }

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape' && state.drag) {
        event.stopPropagation()
        finishDrag()
      }
    }

    updateAnimation()
    motion.addEventListener('change', updateAnimation)
    document.addEventListener('visibilitychange', updateAnimation)
    window.addEventListener('keydown', onKeyDown, true)
    return () => {
      cancelAnimationFrame(frame)
      motion.removeEventListener('change', updateAnimation)
      document.removeEventListener('visibilitychange', updateAnimation)
      window.removeEventListener('keydown', onKeyDown, true)
    }
  }, [route, bubbles])

  function onPointerDown(event: ReactPointerEvent<HTMLButtonElement>) {
    if (!event.isPrimary || (event.pointerType === 'mouse' && event.button !== 0) || stateRef.current.drag) return
    const canvas = walkerRef.current?.parentElement
    if (!canvas) return
    suppressClick.current = false
    const pointer = canvasPoint(canvas, event.clientX, event.clientY)
    const state = stateRef.current
    state.drag = {
      pointerId: event.pointerId, startX: event.clientX, startY: event.clientY,
      offset: { x: state.point.x - pointer.x, y: state.point.y - pointer.y }, dragging: false,
    }
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  function onPointerMove(event: ReactPointerEvent<HTMLButtonElement>) {
    const drag = stateRef.current.drag
    const canvas = walkerRef.current?.parentElement
    if (!drag || drag.pointerId !== event.pointerId || !canvas) return
    if (!drag.dragging && Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY) >= dragThreshold) {
      drag.dragging = true
      setDragging(true)
    }
    if (drag.dragging) {
      const point = canvasPoint(canvas, event.clientX, event.clientY)
      place({ x: point.x + drag.offset.x, y: point.y + drag.offset.y })
    }
  }

  function onClick(event: React.MouseEvent<HTMLButtonElement>) {
    if (event.detail !== 0 && suppressClick.current) {
      suppressClick.current = false
      return
    }
    const result = registerWalkerClick(clicks.current, performance.now())
    clicks.current = result.clicks
    if (result.triggered) showBubble('touch')
  }

  return (
    <div
      ref={walkerRef}
      className={`${styles.walker}${dragging ? ` ${styles.walkerDragging}` : ''}`}
      style={{ transform: `translate(${route.start.x}px, ${route.start.y}px) translate(-50%, -100%)` }}
    >
      {bubble && dialogue ? <span className={styles.walkerDialogue} role="status">{dialogue[locale][bubble]}</span> : null}
      <button
        ref={buttonRef}
        type="button"
        className={styles.walkerButton}
        aria-label={t('student.bubbleMap.walkerLabel')}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={(event) => {
          if (stateRef.current.drag?.pointerId === event.pointerId) {
            onPointerMove(event)
            finishDrag()
          }
        }}
        onPointerCancel={(event) => {
          if (stateRef.current.drag?.pointerId === event.pointerId) finishDrag()
        }}
        onLostPointerCapture={(event) => {
          if (stateRef.current.drag?.pointerId === event.pointerId) finishDrag()
        }}
        onClick={onClick}
      >
        <span className={styles.walkerSprite}><PixelSpriteView sprite={sprite} size={spriteSize} /></span>
      </button>
    </div>
  )
}
