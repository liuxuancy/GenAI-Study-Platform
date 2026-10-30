// 课程详情：气泡一次展开或收回一层。点叶子时说明和对话框一起出现在右侧。
// 新建课程还没有入库，详情暂时用这棵示例树，标题用课程列表里刚填的名字。
import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { Link, Navigate, useParams, useSearchParams } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { bubbleLevelName, sampleCourseTree, type BubbleNode } from '@/modules/student/bubbleTree'
import {
  ancestorIds,
  bubbleLinks,
  closeDeepestLayer,
  findBubble,
  fitScale,
  mapExtent,
  placeBubbles,
  stepOpenLayer,
} from '@/modules/student/bubbleLayout'
import { getStudentCourse, updateStudentCourseProgress, type StudentCourseDetail } from '@/modules/student/api'
import styles from '@/modules/student/CourseBubblePage.module.css'

const labelSize = [18, 13.5, 11.5, 11]

export function CourseBubblePage() {
  const { courseId } = useParams()
  const [searchParams] = useSearchParams()
  const currentUser = useAuthStore((state) => state.currentUser)
  const isNewCourse = courseId === 'new'
  const previewTitle = searchParams.get('title')?.trim() ?? ''
  const [course, setCourse] = useState<StudentCourseDetail | null>(
    isNewCourse ? { id: 0, title: previewTitle } : null,
  )
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    if (isNewCourse || currentUser?.role !== 'ROLE_STUDENT' || !courseId) {
      return
    }
    let cancelled = false
    getStudentCourse(courseId)
      .then((detail) => {
        if (!cancelled) {
          setCourse(detail)
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setErrorMessage(error.message)
        }
      })
    return () => {
      cancelled = true
    }
  }, [courseId, currentUser?.role, isNewCourse])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_STUDENT') {
    return <Navigate to="/" replace />
  }

  return (
    <div className={styles.page}>
      <div className={styles.back}>
        <Link to="/student">Course list</Link>
        {course?.title ? <span className={styles.courseName}>{course.title}</span> : null}
      </div>
      {errorMessage ? <p className={styles.error}>{errorMessage}</p> : null}
      {course ? <BubbleMap courseId={isNewCourse ? '' : (courseId ?? '')} /> : null}
    </div>
  )
}

function BubbleMap({ courseId }: { courseId: string }) {
  const stageRef = useRef<HTMLDivElement>(null)
  const drag = useRef<{ x: number; y: number; panX: number; panY: number } | null>(null)
  const [openIds, setOpenIds] = useState<ReadonlySet<string>>(() => new Set())
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [detailId, setDetailId] = useState<string | null>(null)
  const [doneIds, setDoneIds] = useState<ReadonlySet<string>>(() => new Set())
  const [saving, setSaving] = useState(false)
  const [doneError, setDoneError] = useState('')
  const [scale, setScale] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })

  useLayoutEffect(() => {
    const stage = stageRef.current
    if (!stage) {
      return
    }
    const update = () => {
      setScale(fitScale(mapExtent(sampleCourseTree, openIds), stage.clientWidth, stage.clientHeight))
    }
    update()
    const observer = new ResizeObserver(update)
    observer.observe(stage)
    return () => observer.disconnect()
  }, [openIds])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') {
        return
      }
      if (detailId) {
        setDetailId(null)
        setPan({ x: 0, y: 0 })
      }
      setOpenIds((current) => (current.size === 0 ? current : closeDeepestLayer(current)))
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [detailId])

  const bubbles = placeBubbles(sampleCourseTree, openIds, scale)
  const links = bubbleLinks(bubbles)
  const extent = mapExtent(sampleCourseTree, openIds)
  const box = Math.ceil(extent * 2 * scale) + 8
  const selected = selectedId ? findBubble(sampleCourseTree, selectedId) : null
  const detail = detailId ? findBubble(sampleCourseTree, detailId) : null
  const leaf = detail && !detail.children?.length ? detail : null

  const closeDetail = () => {
    setDetailId(null)
    setPan({ x: 0, y: 0 })
  }

  async function markDone() {
    if (!detailId || saving) {
      return
    }
    const next = new Set(doneIds)
    if (next.has(detailId)) {
      next.delete(detailId)
    } else {
      next.add(detailId)
    }
    if (courseId) {
      setSaving(true)
      setDoneError('')
      try {
        await updateStudentCourseProgress(courseId, leafPercent(next))
      } catch (error) {
        setDoneError(error instanceof Error ? error.message : 'Could not save progress.')
        setSaving(false)
        return
      }
      setSaving(false)
    }
    setDoneIds(next)
  }

  const chooseBubble = (id: string, hasChildren: boolean) => {
    setSelectedId(id)
    if (!hasChildren) {
      if (detailId === null) {
        setPan({ x: 0, y: 0 })
      }
      setDetailId(id)
      return
    }
    setOpenIds((currentOpen) => stepOpenLayer(currentOpen, id))
  }

  return (
    <>
      <div className={styles.map} data-leaf={leaf ? 'true' : 'false'}>
      <button
        className={styles.collapse}
        type="button"
        onClick={() => {
          setOpenIds(new Set())
          setSelectedId('r')
          setDetailId(null)
          setPan({ x: 0, y: 0 })
        }}
      >
        Collapse all
      </button>
      <div
        ref={stageRef}
        className={styles.stage}
        onPointerDown={(event) => {
          const target = event.target as HTMLElement
          if (target.closest('button, a')) {
            return
          }
          drag.current = { x: event.clientX, y: event.clientY, panX: pan.x, panY: pan.y }
          event.currentTarget.setPointerCapture(event.pointerId)
        }}
        onPointerMove={(event) => {
          if (!drag.current) {
            return
          }
          setPan({
            x: drag.current.panX + event.clientX - drag.current.x,
            y: drag.current.panY + event.clientY - drag.current.y,
          })
        }}
        onPointerUp={() => {
          drag.current = null
        }}
      >
        <div className={styles.canvas} style={{ width: box, height: box, transform: `translate(calc(-50% + ${pan.x}px), calc(-50% + ${pan.y}px))` }}>
          {links.map((link) => (
            <span
              key={link.id}
              className={styles.link}
              style={{
                width: link.length,
                transform: `translate(${link.x}px, ${link.y}px) rotate(${link.angle}deg)`,
              }}
            />
          ))}
          {bubbles.map((bubble) => {
            const hasChildren = Boolean(bubble.node.children?.length)
            return (
              <button
                key={bubble.id}
                type="button"
                className={styles.bubble}
                data-depth={bubble.depth}
                data-open={openIds.has(bubble.id) ? 'true' : 'false'}
                data-done={isBubbleDone(bubble.node, bubble.id, doneIds) ? 'true' : 'false'}
                aria-label={`${bubbleLevelName[bubble.depth]}: ${bubble.node.label}`}
                style={{
                  width: bubble.size,
                  height: bubble.size,
                  transform: `translate(-50%, -50%) translate(${bubble.x}px, ${bubble.y}px)`,
                }}
                onClick={() => chooseBubble(bubble.id, hasChildren)}
              >
                {bubble.node.code ? <span className={styles.code}>{bubble.node.code}</span> : null}
                <span className={styles.label} style={{ fontSize: labelSize[bubble.depth] * Math.max(scale, 0.86) }}>
                  {bubble.node.label}
                </span>
              </button>
            )
          })}
        </div>
      </div>
      {selected && selectedId ? (
        <div className={styles.note}>
          <p className={styles.crumb}>
            {ancestorIds(selectedId).map((id) => {
              const ancestor = findBubble(sampleCourseTree, id)
              if (!ancestor) {
                return null
              }
              return (
                <button key={id} type="button" onClick={() => chooseBubble(id, Boolean(ancestor.children?.length))}>
                  {ancestor.label}
                </button>
              )
            })}
          </p>
          {leaf && selectedId === detailId ? null : <p>{selected.note}</p>}
        </div>
      ) : null}
      </div>
      {leaf && detailId ? (
        <aside className={styles.detail}>
          <button
            className={styles.done}
            type="button"
            aria-pressed={doneIds.has(detailId)}
            disabled={saving}
            onClick={() => {
              void markDone()
            }}
          >
            {doneIds.has(detailId) ? 'done' : 'undone'}
          </button>
          <button className={styles.close} type="button" aria-label="Close" onClick={closeDetail}>
            ×
          </button>
          <div className={styles.detailBody}>
            <p className={styles.level}>{bubbleLevelName[bubbleDepth(detailId)]}</p>
            <h2 className={styles.detailTitle}>{leaf.label}</h2>
            <p className={styles.detailNote}>{leaf.note}</p>
            {doneError ? <p className={styles.doneError}>{doneError}</p> : null}
          </div>
          <CornerChat />
        </aside>
      ) : null}
    </>
  )
}

function bubbleDepth(id: string) {
  return id.split('.').length - 1
}

function isBubbleDone(node: BubbleNode, id: string, doneIds: ReadonlySet<string>): boolean {
  if (!node.children?.length) {
    return doneIds.has(id)
  }
  return node.children.every((child, index) => isBubbleDone(child, `${id}.${index}`, doneIds))
}

function leafPercent(doneIds: ReadonlySet<string>) {
  const total = countLeaves(sampleCourseTree)
  if (total <= 0) {
    return 0
  }
  const percent = Math.floor((countDoneLeaves(sampleCourseTree, 'r', doneIds) * 100) / total)
  if (percent > 100) {
    return 100
  }
  return percent
}

function countLeaves(node: BubbleNode): number {
  if (!node.children?.length) {
    return 1
  }
  let total = 0
  for (const child of node.children) {
    total += countLeaves(child)
  }
  return total
}

function countDoneLeaves(node: BubbleNode, id: string, doneIds: ReadonlySet<string>): number {
  if (!node.children?.length) {
    return doneIds.has(id) ? 1 : 0
  }
  let total = 0
  for (let index = 0; index < node.children.length; index += 1) {
    total += countDoneLeaves(node.children[index], `${id}.${index}`, doneIds)
  }
  return total
}

interface ChatMessage {
  role: 'user' | 'reply'
  text: string
}

function CornerChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [draft, setDraft] = useState('')

  function sendMessage() {
    const text = draft.trim()
    if (!text) {
      return
    }
    setMessages((current) => [...current, { role: 'user', text }, { role: 'reply', text: localReply(text) }])
    setDraft('')
  }

  return (
    <section className={styles.chat} aria-label="Ask">
      <div className={styles.chatLog}>
        {messages.map((message, index) => (
          <p key={`${message.role}-${index}`} className={message.role === 'user' ? styles.chatUser : styles.chatReply}>
            {message.text}
          </p>
        ))}
      </div>
      <form
        className={styles.chatForm}
        onSubmit={(event) => {
          event.preventDefault()
          sendMessage()
        }}
      >
        <input
          aria-label="Message"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
        />
        <button type="submit">Send</button>
      </form>
    </section>
  )
}

function localReply(text: string) {
  return `Noted: ${text}`
}
