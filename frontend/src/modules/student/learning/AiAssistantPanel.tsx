import { Link } from 'react-router-dom'
import { useEffect, useMemo, useRef, useState } from 'react'

import { CourseMarkdownBody } from '@/modules/student/courseContentView'
import { useI18n } from '@/core/i18n/useI18n'
import {
  askStudentChapter,
} from '@/modules/student/api'
import styles from '@/modules/student/CourseBubblePage.module.css'

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
  const { t } = useI18n()
  const shortcuts = useMemo(
    () =>
      [
        { id: 'summary', label: t('student.aiAssistant.quickSummary'), prompt: t('student.aiAssistant.promptSummary') },
        { id: 'explain', label: t('student.aiAssistant.quickExplain'), prompt: t('student.aiAssistant.promptExplain') },
        { id: 'practice', label: t('student.aiAssistant.quickPractice'), prompt: t('student.aiAssistant.promptPractice') },
        { id: 'ask', label: t('student.aiAssistant.quickAsk'), prompt: '' },
      ] as const,
    [t],
  )
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

  const lessonLabel = sectionTitle || chapterTitle || t('student.aiAssistant.currentSection')

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
      setError(reason instanceof Error ? reason.message : t('student.aiAssistant.qaFailed'))
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
        label: t('student.aiAssistant.switchedLesson', { section: lessonLabel }),
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
    sendRef.current(t('student.aiAssistant.explainSelection'), {
      selection: excerpt,
      display: t('student.aiAssistant.explainDisplay', { text: excerpt }),
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

  function sendShortcut(id: string, prompt: string) {
    if (id === 'ask') {
      inputRef.current?.focus()
      return
    }
    if (id === 'explain' && selectedExcerpt.trim()) {
      const excerpt = selectedExcerpt.trim()
      void sendMessage(t('student.aiAssistant.explainSelection'), {
        selection: excerpt,
        display: t('student.aiAssistant.explainDisplay', { text: excerpt }),
      })
      return
    }
    void sendMessage(prompt)
  }

  return (
    <aside className={styles.aiPane} aria-label={t('student.aiAssistant.panelAria')}>
      <div className={styles.aiHead}>
        <div>
          <h2>{t('student.aiAssistant.title')}</h2>
          <p className={styles.aiContext}>{t('student.aiAssistant.context', { section: lessonLabel })}</p>
        </div>
        <div className={styles.aiHeadActions}>
          <Link className={styles.newChatBtn} to={`/student/ai?courseId=${courseId}`}>
            {t('student.aiAssistant.history')}
          </Link>
          {savedConversationId ? (
            <Link className={styles.newChatBtn} to={`/student/ai?conversation=${savedConversationId}`}>
              {t('student.aiAssistant.continueInWorkspace')}
            </Link>
          ) : null}
          <button className={styles.newChatBtn} type="button" onClick={startNewChat}>
            {t('student.aiAssistant.newChat')}
          </button>
        </div>
      </div>
      <div className={styles.chatLog} ref={logRef} aria-live="polite">
        {messages.length === 0 ? (
          <p className={styles.chatPlaceholder}>{t('student.aiAssistant.welcome')}</p>
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
        {sending ? <p className={styles.chatPending}>{t('student.aiAssistant.pending')}</p> : null}
        {error ? <p className={styles.doneError}>{error}</p> : null}
        {retry ? (
          <button className={styles.chatRetry} type="button" onClick={() => void sendMessage(retry.question, retry)}>
            {t('student.aiAssistant.retry')}
          </button>
        ) : null}
      </div>
      <div className={styles.chatComposer}>
        <ul className={styles.aiShortcuts}>
          {shortcuts.map((item) => (
            <li key={item.id}>
              <button type="button" disabled={sending || !chapterKey} onClick={() => sendShortcut(item.id, item.prompt)}>
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
            aria-label={t('student.aiAssistant.askAria')}
            rows={2}
            value={draft}
            placeholder={
              chapterKey ? t('student.aiAssistant.askPlaceholder') : t('student.aiAssistant.askPlaceholderNoKey')
            }
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
            {t('student.aiAssistant.send')}
          </button>
        </form>
      </div>
    </aside>
  )
}
