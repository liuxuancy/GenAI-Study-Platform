import { useEffect, useMemo, useRef, useState } from 'react'
import { Popconfirm } from 'antd'

import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import { CourseMarkdownBody } from '@/modules/student/courseContentView'
import {
  deleteWorkspaceConversation,
  getWorkspaceConversation,
  listWorkspaceConversations,
  listWorkspaceSkills,
  renameWorkspaceConversation,
  sendWorkspaceMessage,
  type WorkspaceConversation,
  type WorkspaceMessage,
  type WorkspaceSkill,
} from '@/modules/workspace/api'
import { buildSkillDefaults, normalizeSkillParams } from '@/modules/workspace/skillParams'
import styles from '@/modules/workspace/AiWorkspace.module.css'

const SKILL_ICONS = ['◇', '▤', '☑', '✦', '▦', '☑']

function dayGroup(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '更早'
  }
  const start = new Date()
  start.setHours(0, 0, 0, 0)
  const diff = start.getTime() - new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
  if (diff <= 0) {
    return '今天'
  }
  if (diff <= 86400000) {
    return '昨天'
  }
  return '更早'
}

function courseLabel(courses: Array<{ id: number; title: string }>, courseId: number | null) {
  if (!courseId) {
    return '不关联课程'
  }
  return courses.find((course) => course.id === courseId)?.title ?? `课程 #${courseId}`
}

export function AiWorkspace({
  role,
  courses,
  initialConversationId,
  initialCourseId,
}: {
  role: 'student' | 'teacher'
  courses: Array<{ id: number; title: string }>
  initialConversationId?: number | null
  initialCourseId?: number | null
}) {
  const { provider, selector } = useAiProviderSelect()
  const [conversations, setConversations] = useState<WorkspaceConversation[]>([])
  const [skills, setSkills] = useState<WorkspaceSkill[]>([])
  const [query, setQuery] = useState('')
  const [courseFilter, setCourseFilter] = useState(initialCourseId ? String(initialCourseId) : '')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [activeId, setActiveId] = useState<number | null>(initialConversationId ?? null)
  const [messages, setMessages] = useState<WorkspaceMessage[]>([])
  const [title, setTitle] = useState('')
  const [courseId, setCourseId] = useState<number | null>(initialCourseId ?? null)
  const [accessible, setAccessible] = useState(true)
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [menuId, setMenuId] = useState<number | null>(null)
  const [skill, setSkill] = useState<WorkspaceSkill | null>(null)
  const [skillValues, setSkillValues] = useState<Record<string, string>>({})
  const [skillPopupOpen, setSkillPopupOpen] = useState(false)
  const [coursePopupOpen, setCoursePopupOpen] = useState(false)
  const [copyId, setCopyId] = useState<number | null>(null)
  const openedRef = useRef<number | null>(null)
  const promptRef = useRef<HTMLTextAreaElement>(null)
  const composerAnchorRef = useRef<HTMLFormElement>(null)

  const welcomeTitle = role === 'teacher' ? '今天想准备哪一节课？' : '今天想探索什么？'
  const eyebrow = role === 'teacher' ? 'YOUR PERSONAL TEACHING COMPANION' : 'YOUR PERSONAL LEARNING COMPANION'
  const modeTag = role === 'teacher' ? '✧  教学模式' : '✧  学习模式'
  const workspaceLabel = role === 'teacher' ? 'AI 工作台' : 'AI 学习工作台'

  const examples =
    role === 'teacher'
      ? [
          { label: '✧ 备课提纲', prompt: '请帮我设计下一节课的导入、讲解与练习环节提纲：' },
          { label: '◇ 课堂讨论', prompt: '请围绕本节课主题生成 5 个有深度的课堂讨论问题：' },
          { label: '✦ 教学目标', prompt: '请帮我整理本单元的教学目标与评价要点：' },
        ]
      : [
          { label: '✧ 制定学习计划', prompt: '帮我制定一份本周学习计划' },
          { label: '◇ 梳理思维导图', prompt: '帮我梳理一个复杂概念的思维导图' },
          { label: '✦ 高效复习建议', prompt: '给我一些高效复习的建议' },
        ]

  async function refresh() {
    setConversations(await listWorkspaceConversations())
  }

  useEffect(() => {
    void refresh().catch(() => setConversations([]))
    listWorkspaceSkills()
      .then(setSkills)
      .catch(() => setSkills([]))
  }, [])

  useEffect(() => {
    if (!initialConversationId || openedRef.current === initialConversationId || conversations.length === 0) {
      return
    }
    openedRef.current = initialConversationId
    void openConversation(initialConversationId)
  }, [conversations, initialConversationId])

  useEffect(() => {
    const el = promptRef.current
    if (!el) {
      return
    }
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 210)}px`
  }, [draft])

  useEffect(() => {
    function onPointerDown(event: MouseEvent) {
      const root = composerAnchorRef.current
      if (!root || root.contains(event.target as Node)) {
        return
      }
      setSkillPopupOpen(false)
      setCoursePopupOpen(false)
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setSkillPopupOpen(false)
        setCoursePopupOpen(false)
      }
    }
    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [])

  const groups = useMemo(() => {
    const needle = query.trim().toLowerCase()
    const filtered = conversations.filter((item) => {
      if (courseFilter && String(item.courseId || '') !== courseFilter) {
        return false
      }
      return !needle || item.title.toLowerCase().includes(needle)
    })
    const buckets = new Map<string, WorkspaceConversation[]>()
    for (const item of filtered) {
      const key = dayGroup(item.updatedAt)
      buckets.set(key, [...(buckets.get(key) ?? []), item])
    }
    return ['今天', '昨天', '更早'].flatMap((key) => (buckets.get(key) ? [{ key, items: buckets.get(key) ?? [] }] : []))
  }, [conversations, courseFilter, query])

  async function openConversation(id: number) {
    const summary = conversations.find((item) => item.id === id)
    setAccessible(summary ? summary.courseAccessible || summary.courseId === 0 : true)
    const detail = await getWorkspaceConversation(id)
    setActiveId(detail.id)
    setTitle(detail.title)
    setMessages(detail.messages)
    setCourseId(summary?.courseId || null)
    setSkill(null)
    setError('')
    setMenuId(null)
  }

  function startNew() {
    setActiveId(null)
    setMessages([])
    setTitle('')
    setDraft('')
    setError('')
    setSkill(null)
    setAccessible(true)
    setCourseId(courseFilter ? Number(courseFilter) : null)
    openedRef.current = null
    setMenuId(null)
    promptRef.current?.focus()
  }

  async function send(content: string, skillId?: string, params?: Record<string, string>) {
    const text = content.trim()
    if ((!text && !skillId) || sending || !accessible) {
      return
    }
    let submitParams = params
    if (skillId && skill) {
      const normalized = normalizeSkillParams(skill, params ?? {})
      if ('error' in normalized) {
        setError(normalized.error)
        return
      }
      submitParams = normalized.params
    }
    setSending(true)
    setError('')
    setSkillPopupOpen(false)
    setCoursePopupOpen(false)
    try {
      const result = await sendWorkspaceMessage({
        conversationId: activeId ?? undefined,
        courseId: courseId ?? undefined,
        content: text,
        provider,
        skillId,
        params: submitParams,
      })
      setActiveId(result.conversationId)
      setTitle(result.title)
      const detail = await getWorkspaceConversation(result.conversationId)
      setMessages(detail.messages)
      setDraft('')
      setSkill(null)
      await refresh()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '发送失败，请重试。')
    } finally {
      setSending(false)
    }
  }

  async function rename(item: WorkspaceConversation) {
    const next = window.prompt('重命名会话', item.title)
    if (!next || next.trim() === item.title) {
      return
    }
    await renameWorkspaceConversation(item.id, next.trim())
    setConversations((current) => current.map((entry) => (entry.id === item.id ? { ...entry, title: next.trim() } : entry)))
    if (activeId === item.id) {
      setTitle(next.trim())
    }
    setMenuId(null)
  }

  async function remove(item: WorkspaceConversation) {
    await deleteWorkspaceConversation(item.id)
    setConversations((current) => current.filter((entry) => entry.id !== item.id))
    if (activeId === item.id) {
      startNew()
    }
  }

  async function copyAnswer(messageId: number, text: string) {
    try {
      await navigator.clipboard.writeText(text)
      setCopyId(messageId)
      window.setTimeout(() => setCopyId(null), 1500)
    } catch {
      setCopyId(-messageId)
      window.setTimeout(() => setCopyId(null), 1500)
    }
  }

  function chooseExample(prompt: string) {
    setDraft(prompt)
    promptRef.current?.focus()
  }

  const showWelcome = messages.length === 0 && !sending
  const canSend = Boolean((draft.trim() || skill) && !sending && accessible)
  const headerTitle = title || workspaceLabel

  return (
    <div className={styles.workspace}>
      <aside
        id="workspace-sessions"
        className={`${styles.sessions} ${sidebarOpen ? '' : styles.sessionsClosed}`}
        aria-hidden={!sidebarOpen}
      >
        <div className={styles.sessionInner}>
          <div className={styles.sessionHeading}>历史对话</div>
          <button className={styles.newChat} type="button" onClick={startNew}>
            ＋ &nbsp; 新建对话
          </button>
          <label className={styles.search}>
            ⌕
            <input
              aria-label="搜索历史对话"
              placeholder="搜索历史对话"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
          <select
            className={styles.filter}
            aria-label="筛选会话"
            value={courseFilter}
            onChange={(event) => setCourseFilter(event.target.value)}
          >
            <option value="">全部会话</option>
            <option value="0">未关联课程</option>
            {courses.map((course) => (
              <option key={course.id} value={course.id}>
                {course.title}
              </option>
            ))}
          </select>
          <div className={styles.history}>
            {groups.map((group) => (
              <section key={group.key}>
                <p className={styles.group}>{group.key}</p>
                {group.items.map((item) => (
                  <div key={item.id} className={styles.threadRow}>
                    <button
                      type="button"
                      className={`${styles.thread} ${item.id === activeId ? styles.threadActive : ''}`}
                      onClick={() => void openConversation(item.id)}
                    >
                      {item.title}
                      <small>{item.courseTitle || '未关联课程'}</small>
                    </button>
                    <button
                      className={styles.more}
                      type="button"
                      aria-label="更多操作"
                      onClick={() => setMenuId(menuId === item.id ? null : item.id)}
                    >
                      ···
                    </button>
                    {menuId === item.id ? (
                      <div className={styles.menu}>
                        <button type="button" onClick={() => void rename(item)}>
                          重命名
                        </button>
                        <Popconfirm title="删除这个会话？" okText="删除" cancelText="取消" onConfirm={() => remove(item)}>
                          <button type="button">删除</button>
                        </Popconfirm>
                      </div>
                    ) : null}
                  </div>
                ))}
              </section>
            ))}
          </div>
          <div className={styles.sidebarFoot}>✦ 让每一次提问都成为进步</div>
        </div>
      </aside>

      <section className={styles.workspaceMain}>
        <header className={styles.topbar}>
          <div className={styles.topLeft}>
            <button
              className={`${styles.iconBtn} ${styles.sidebarToggle}`}
              type="button"
              aria-label={sidebarOpen ? '收起历史对话' : '展开历史对话'}
              aria-expanded={sidebarOpen}
              aria-controls="workspace-sessions"
              onClick={() => setSidebarOpen((open) => !open)}
            >
              {sidebarOpen ? '☷' : '☰'}
            </button>
            <div className={styles.breadcrumb}>
              CaveClass <span>›</span> <b>{headerTitle}</b>
            </div>
          </div>
          <div className={styles.topRight}>
            <span className={styles.headerStatus}>
              <span className={styles.statusDot} aria-hidden="true" />
              AI 助手已就绪
            </span>
            <span className={`${styles.tag} ${styles.tagPill}`}>{modeTag}</span>
            <div className={styles.modelTag}>{selector}</div>
          </div>
        </header>

        <main className={styles.main}>
          {showWelcome ? (
            <div className={styles.welcome}>
              <div className={styles.eyebrow}>
                <span className={styles.spark} aria-hidden="true">
                  ✦
                </span>
                {eyebrow}
              </div>
              <h1>{welcomeTitle}</h1>
              <p className={styles.subtitle}>从一个问题开始，让 AI 帮你学习、思考与创造。</p>
              <p className={styles.welcomeMeta}>✦ &nbsp; 从下面选择一个学习场景，或者直接开始提问</p>
              <div className={styles.cards}>
                {skills.map((item, index) => (
                  <button
                    key={item.id}
                    type="button"
                    className={styles.skillCard}
                    onClick={() => {
                      setSkill(item)
                      setSkillValues({})
                      promptRef.current?.focus()
                    }}
                  >
                    <span className={styles.skillIcon} aria-hidden="true">
                      {SKILL_ICONS[index % SKILL_ICONS.length]}
                    </span>
                    <strong>{item.name}</strong>
                    <span>{item.description}</span>
                    <span className={styles.skillArrow} aria-hidden="true">
                      ↗
                    </span>
                  </button>
                ))}
              </div>
              <p className={styles.tipsTitle}>试试这样提问</p>
              <div className={styles.examples}>
                {examples.map((item) => (
                  <button key={item.label} type="button" className={styles.example} onClick={() => chooseExample(item.prompt)}>
                    {item.label}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className={styles.chat} aria-live="polite">
              {messages.map((item) =>
                item.role === 'assistant' ? (
                  <div key={item.id} className={styles.messageAi}>
                    <div className={styles.speaker}>CaveClass AI</div>
                    <div className={styles.aiBubble}>
                      <CourseMarkdownBody source={item.content} />
                    </div>
                    <button type="button" className={styles.copy} onClick={() => void copyAnswer(item.id, item.content)}>
                      {copyId === item.id ? '已复制' : copyId === -item.id ? '复制失败' : '复制内容'}
                    </button>
                  </div>
                ) : (
                  <div key={item.id} className={styles.messageUser}>
                    <div className={styles.bubble}>{item.content}</div>
                  </div>
                ),
              )}
              {sending ? <p className={styles.pending}>正在回答…</p> : null}
              {error ? (
                <p className={styles.error}>
                  {error}
                  <button type="button" onClick={() => void send(draft, skill?.id, skillValues)}>
                    重试
                  </button>
                </p>
              ) : null}
            </div>
          )}
        </main>

        <div className={styles.composerWrap}>
          <form
            className={styles.composerAnchor}
            ref={composerAnchorRef}
            onSubmit={(event) => {
              event.preventDefault()
              void send(draft, skill?.id, skillValues)
            }}
          >
            {skillPopupOpen ? (
              <div className={styles.popup} role="dialog" aria-label="选择 AI 技能">
                <h3 className={styles.popupTitle}>选择 AI 技能</h3>
                <div className={styles.popupList}>
                  {skills.map((item, index) => (
                    <button
                      key={item.id}
                      type="button"
                      className={styles.popupItem}
                      onClick={() => {
                        setSkill(item)
                        setSkillValues(buildSkillDefaults(item))
                        setSkillPopupOpen(false)
                        promptRef.current?.focus()
                      }}
                    >
                      {SKILL_ICONS[index % SKILL_ICONS.length]} {item.name}
                      <span className={styles.popupItemDesc}>{item.description}</span>
                    </button>
                  ))}
                </div>
              </div>
            ) : null}
            {coursePopupOpen ? (
              <div className={styles.popup} role="dialog" aria-label="关联课程">
                <h3 className={styles.popupTitle}>关联课程</h3>
                <div className={styles.popupList}>
                  <button
                    type="button"
                    className={`${styles.popupItem} ${courseId === null ? styles.popupItemSelected : ''}`}
                    aria-current={courseId === null ? 'true' : undefined}
                    onClick={() => {
                      setCourseId(null)
                      setCoursePopupOpen(false)
                    }}
                  >
                    不关联课程
                  </button>
                  {courses.map((course) => (
                    <button
                      key={course.id}
                      type="button"
                      className={`${styles.popupItem} ${courseId === course.id ? styles.popupItemSelected : ''}`}
                      aria-current={courseId === course.id ? 'true' : undefined}
                      onClick={() => {
                        setCourseId(course.id)
                        setCoursePopupOpen(false)
                      }}
                    >
                      {course.title}
                    </button>
                  ))}
                </div>
              </div>
            ) : null}
            <div className={styles.composerBox}>
              {skill ? (
                <div className={styles.skillForm}>
                  <strong>{skill.name}</strong>
                  {skill.params.map((param) =>
                    param.type === 'select' ? (
                      <select
                        key={param.key}
                        aria-label={param.label}
                        value={skillValues[param.key] ?? ''}
                        onChange={(event) => setSkillValues((current) => ({ ...current, [param.key]: event.target.value }))}
                      >
                        <option value="">{param.label}</option>
                        {param.options?.map((option) => (
                          <option key={option}>{option}</option>
                        ))}
                      </select>
                    ) : param.type === 'number' ? (
                      <input
                        key={param.key}
                        type="number"
                        aria-label={param.label}
                        placeholder={
                          param.min != null && param.max != null
                            ? `${param.label}（${param.min}–${param.max}）`
                            : param.label
                        }
                        min={param.min}
                        max={param.max}
                        step={1}
                        inputMode="numeric"
                        value={skillValues[param.key] ?? ''}
                        onChange={(event) => setSkillValues((current) => ({ ...current, [param.key]: event.target.value }))}
                      />
                    ) : (
                      <input
                        key={param.key}
                        aria-label={param.label}
                        placeholder={param.label}
                        value={skillValues[param.key] ?? ''}
                        onChange={(event) => setSkillValues((current) => ({ ...current, [param.key]: event.target.value }))}
                      />
                    ),
                  )}
                  <button type="button" onClick={() => setSkill(null)}>
                    取消技能
                  </button>
                </div>
              ) : null}
              <textarea
                ref={promptRef}
                aria-label="向 CaveClass AI 提问"
                rows={2}
                value={draft}
                placeholder="向 CaveClass AI 提问，或选择一个技能开始…"
                disabled={sending || !accessible}
                onChange={(event) => setDraft(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
                    event.preventDefault()
                    void send(draft, skill?.id, skillValues)
                  }
                }}
              />
              <div className={styles.composerBottom}>
                <div className={styles.tools}>
                  <button className={`${styles.tool} ${styles.toolAttach}`} type="button" title="暂不支持上传" disabled>
                    ＋
                  </button>
                  <button
                    className={`${styles.tool} ${skillPopupOpen || skill ? styles.toolSelected : ''}`}
                    type="button"
                    aria-expanded={skillPopupOpen}
                    aria-haspopup="dialog"
                    onClick={() => {
                      setSkillPopupOpen((open) => !open)
                      setCoursePopupOpen(false)
                    }}
                  >
                    <span aria-hidden="true">✧</span>
                    <span className={styles.toolLabel}>技能</span>
                    <span className={styles.toolCaret} aria-hidden="true">
                      ⌄
                    </span>
                  </button>
                  <button
                    className={`${styles.tool} ${coursePopupOpen ? styles.toolSelected : ''}`}
                    type="button"
                    aria-expanded={coursePopupOpen}
                    aria-haspopup="dialog"
                    disabled={Boolean(activeId)}
                    onClick={() => {
                      setCoursePopupOpen((open) => !open)
                      setSkillPopupOpen(false)
                    }}
                  >
                    <span aria-hidden="true">▤</span>
                    <span className={styles.toolLabel}>{courseLabel(courses, courseId)}</span>
                    <span className={styles.toolCaret} aria-hidden="true">
                      ⌄
                    </span>
                  </button>
                </div>
                <button className={styles.send} type="submit" disabled={!canSend} aria-label="发送消息">
                  ↑
                </button>
              </div>
            </div>
          </form>
          <p className={styles.hint}>CaveClass AI 可能产生不准确的信息，请核对重要内容</p>
        </div>
      </section>
    </div>
  )
}
