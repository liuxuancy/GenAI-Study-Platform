import { Link } from 'react-router-dom'
import { useEffect, useRef, useState } from 'react'

import { CourseMarkdownBody } from '@/modules/student/courseContentView'
import {
  askStudentChapter,
} from '@/modules/student/api'
import styles from '@/modules/student/CourseBubblePage.module.css'

const shortcuts = [
  { label: '章节总结', prompt: '请用简洁的中文总结当前章节的核心要点。' },
  { label: '知识点解释', prompt: '请解释当前章节里最重要的概念，并给一个例子。' },
  { label: '生成练习', prompt: '请根据当前章节内容出 3 道练习题，并给出参考答案。这些练习只用于本次对话，不要当作正式测验。' },
  { label: '提问答疑', prompt: '' },
] as const

interface ChatMessage {
  id: string
  kind: 'message'
  role: 'user' | 'assistant'
  content: string
  status: 'done' | 'error'
}

interface ChatDivider {
  id: string
  kind: 'divider'
  label: string
}

type ChatItem = ChatMessage | ChatDivider

interface SendOptions {
  selection?: string
  display?: string
}

interface RetryRequest {
  question: string
  selection?: string
  display: string
}

let chatSequence = 0

function nextChatId() {
  chatSequence += 1
  return `chat-${chatSequence}`
}

function markLastUser(items: readonly ChatItem[], status: ChatMessage['status']) {
  const next = items.slice()
  for (let index = next.length - 1; index >= 0; index -= 1) {
    const item = next[index]
    if (item?.kind === 'message' && item.role === 'user') {
      next[index] = { ...item, status }
      break
    }
  }
  return next
}

export function AiAssistantPanel({
  courseId,
  chapterKey,
  activityKey,
  chapterTitle,
  sectionTitle,
  provider,
  selectedExcerpt,
  explainRequest,
}: {
  courseId: string
  chapterKey: string
  activityKey: string
  chapterTitle: string
  sectionTitle: string
  provider?: string
  selectedExcerpt: string
  explainRequest: { id: number; text: string } | null
}) {
  const [messages, setMessages] = useState<ChatItem[]>([])
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [retry, setRetry] = useState<RetryRequest | null>(null)
  const [savedConversationId, setSavedConversationId] = useState<number | null>(null)
  const messagesRef = useRef<ChatItem[]>([])
  const conversationIdRef = useRef<number | null>(null)
  const requestGen = useRef(0)
  const sendingRef = useRef(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const logRef = useRef<HTMLDivElement>(null)
  const seenLesson = useRef('')
  const lastExplainId = useRef(0)
  const sendRef = useRef<(question: string, options?: SendOptions) => void>(() => {})

  const lessonLabel = sectionTitle || chapterTitle || '当前章节'

  function commit(next: ChatItem[]) {
    messagesRef.current = next
    setMessages(next)
  }

  async function sendMessage(question: string, options?: SendOptions) {
    const text = question.trim()
    const display = (options?.display ?? text).trim()
    if (!text || sendingRef.current || !chapterKey) {
      return
    }
    const gen = requestGen.current + 1
    requestGen.current = gen
    const prior = messagesRef.current
    const lastUser = [...prior].reverse().find((item) => item.kind === 'message' && item.role === 'user')
    const alreadyShown =
      lastUser?.kind === 'message' && lastUser.content === display && lastUser.status === 'error'
    const withUser = alreadyShown
      ? markLastUser(prior, 'done')
      : [
          ...prior,
          { id: nextChatId(), kind: 'message' as const, role: 'user' as const, content: display, status: 'done' as const },
        ]
    commit(withUser)
    sendingRef.current = true
    setSending(true)
    setError('')
    setRetry(null)
    try {
      const result = await askStudentChapter(courseId, chapterKey, {
        question: text,
        provider,
        activityKey: activityKey || undefined,
        selection: options?.selection,
        conversationId: conversationIdRef.current ?? undefined,
      })
      if (requestGen.current !== gen) {
        return
      }
      conversationIdRef.current = result.conversationId
      setSavedConversationId(result.conversationId)
      commit([
        ...messagesRef.current,
        { id: nextChatId(), kind: 'message', role: 'assistant', content: result.answer, status: 'done' },
      ])
    } catch (reason) {
      if (requestGen.current !== gen) {
        return
      }
      commit(markLastUser(messagesRef.current, 'error'))
      setError(reason instanceof Error ? reason.message : '章节问答失败')
      setRetry({ question: text, selection: options?.selection, display })
    } finally {
      if (requestGen.current === gen) {
        sendingRef.current = false
        setSending(false)
      }
    }
  }

  sendRef.current = (question, options) => {
    void sendMessage(question, options)
  }

  useEffect(() => {
    const marker = `${chapterKey}:${activityKey}:${sectionTitle}`
    if (seenLesson.current === marker) {
      return
    }
    const previous = seenLesson.current
    seenLesson.current = marker
    if (!previous || messagesRef.current.length === 0) {
      return
    }
    commit([
      ...messagesRef.current,
      {
        id: nextChatId(),
        kind: 'divider',
        label: `已切换到「${lessonLabel}」，之后的回答基于这一节`,
      },
    ])
  }, [activityKey, chapterKey, lessonLabel, sectionTitle])

  useEffect(() => {
    if (!explainRequest || explainRequest.id === lastExplainId.current || sending) {
      return
    }
    const excerpt = explainRequest.text.trim()
    if (!excerpt) {
      return
    }
    lastExplainId.current = explainRequest.id
    sendRef.current('请结合当前章节，解释我选中的这段内容。', {
      selection: excerpt,
      display: `请解释这段内容：\n${excerpt}`,
    })
  }, [explainRequest, sending])

  useEffect(() => {
    const node = logRef.current
    if (!node) {
      return
    }
    node.scrollTop = node.scrollHeight
  }, [messages, sending])

  function startNewChat() {
    requestGen.current += 1
    sendingRef.current = false
    conversationIdRef.current = null
    setSavedConversationId(null)
    commit([])
    setSending(false)
    setError('')
    setRetry(null)
    setDraft('')
  }

  function sendShortcut(label: string, prompt: string) {
    if (label === '提问答疑') {
      inputRef.current?.focus()
      return
    }
    if (label === '知识点解释' && selectedExcerpt.trim()) {
      const excerpt = selectedExcerpt.trim()
      void sendMessage('请结合当前章节，解释我选中的这段内容。', {
        selection: excerpt,
        display: `请解释这段内容：\n${excerpt}`,
      })
      return
    }
    void sendMessage(prompt)
  }

  return (
    <aside className={styles.aiPane} aria-label="AI 学习助手">
      <div className={styles.aiHead}>
        <div>
          <h2>AI 学习助手</h2>
          <p className={styles.aiContext}>基于当前章节 · {lessonLabel}</p>
        </div>
        <div className={styles.aiHeadActions}>
          <Link className={styles.newChatBtn} to={`/student/ai?courseId=${courseId}`}>
            历史
          </Link>
          {savedConversationId ? (
            <Link className={styles.newChatBtn} to={`/student/ai?conversation=${savedConversationId}`}>
              在工作台中继续
            </Link>
          ) : null}
          <button className={styles.newChatBtn} type="button" onClick={startNewChat}>
            新对话
          </button>
        </div>
      </div>
      <div className={styles.chatLog} ref={logRef} aria-live="polite">
        {messages.length === 0 ? (
          <p className={styles.chatPlaceholder}>
            你好！我是你的 AI 学习助手。我可以结合当前章节内容，帮你理解知识点、总结要点或进行练习。
          </p>
        ) : null}
        {messages.map((item) =>
          item.kind === 'divider' ? (
            <p key={item.id} className={styles.chatDivider}>
              {item.label}
            </p>
          ) : item.role === 'assistant' ? (
            <div key={item.id} className={styles.chatReply}>
              <CourseMarkdownBody source={item.content} codeClassName={styles.codeBlock} />
            </div>
          ) : (
            <p key={item.id} className={styles.chatUser}>
              {item.content}
            </p>
          ),
        )}
        {sending ? <p className={styles.chatPending}>正在根据本章内容回答…</p> : null}
        {error ? <p className={styles.doneError}>{error}</p> : null}
        {retry ? (
          <button className={styles.chatRetry} type="button" onClick={() => void sendMessage(retry.question, retry)}>
            重试
          </button>
        ) : null}
      </div>
      <div className={styles.chatComposer}>
        <ul className={styles.aiShortcuts}>
          {shortcuts.map((item) => (
            <li key={item.label}>
              <button type="button" disabled={sending || !chapterKey} onClick={() => sendShortcut(item.label, item.prompt)}>
                {item.label}
              </button>
            </li>
          ))}
        </ul>
        <form
          className={styles.chatForm}
          onSubmit={(event) => {
            event.preventDefault()
            void sendMessage(draft)
            setDraft('')
          }}
        >
          <textarea
            ref={inputRef}
            aria-label="向 AI 提问"
            rows={2}
            value={draft}
            placeholder={chapterKey ? '针对当前章节继续提问…' : '当前章节暂无 key，无法问答'}
            disabled={!chapterKey || sending}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
                event.preventDefault()
                if (!draft.trim() || sending) {
                  return
                }
                void sendMessage(draft)
                setDraft('')
              }
            }}
          />
          <button type="submit" disabled={sending || !chapterKey || !draft.trim()}>
            发送
          </button>
        </form>
      </div>
    </aside>
  )
}
