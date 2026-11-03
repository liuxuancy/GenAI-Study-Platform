import { useEffect, useMemo, useRef, useState } from 'react'
import { Popconfirm } from 'antd'

import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import { useI18n } from '@/core/i18n/useI18n'
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
import {
  localizedSkillDescription,
  localizedSkillName,
  localizedSkillParamLabel,
  localizedSkillSelectOption,
} from '@/modules/workspace/workspaceSkillI18n'
import styles from '@/modules/workspace/AiWorkspace.module.css'

const SKILL_ICONS = ['◇', '▤', '☑', '✦', '▦', '☑']

type DayBucket = 'today' | 'yesterday' | 'earlier'

function dayBucket(value: string): DayBucket {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return 'earlier'
  }
  const start = new Date()
  start.setHours(0, 0, 0, 0)
  const diff = start.getTime() - new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
  if (diff <= 0) {
    return 'today'
  }
  if (diff <= 86400000) {
    return 'yesterday'
  }
  return 'earlier'
}

function dayBucketLabel(bucket: DayBucket, t: (key: string) => string) {
  if (bucket === 'today') return t('workspace.dayToday')
  if (bucket === 'yesterday') return t('workspace.dayYesterday')
  return t('workspace.dayEarlier')
}

function courseLabel(
  courses: Array<{ id: number; title: string }>,
  courseId: number | null,
  t: (key: string, params?: Record<string, string | number>) => string,
) {
  if (!courseId) {
    return t('workspace.noCourse')
  }
  return courses.find((course) => course.id === courseId)?.title ?? t('workspace.courseFallback', { id: courseId })
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
  const { t, locale } = useI18n()
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

  const welcomeTitle = role === 'teacher' ? t('workspace.welcomeTeacher') : t('workspace.welcomeStudent')
  const eyebrow = role === 'teacher' ? t('workspace.eyebrowTeacher') : t('workspace.eyebrowStudent')
  const modeTag = role === 'teacher' ? t('workspace.modeTeacher') : t('workspace.modeStudent')
  const workspaceLabel = role === 'teacher' ? t('workspace.labelTeacher') : t('workspace.labelStudent')
  const welcomeMeta = role === 'teacher' ? t('workspace.metaTeacher') : t('workspace.metaStudent')
  const welcomeSkills = useMemo(
    () => (role === 'teacher' ? skills.slice(0, 3) : skills),
    [role, skills],
  )

  const examples = useMemo(
    () =>
      role === 'teacher'
        ? [
            { label: t('workspace.exampleLessonPlan'), prompt: t('workspace.exampleLessonPlanPrompt') },
            { label: t('workspace.exampleDiscussion'), prompt: t('workspace.exampleDiscussionPrompt') },
            { label: t('workspace.exampleObjectives'), prompt: t('workspace.exampleObjectivesPrompt') },
          ]
        : [
            { label: t('workspace.exampleStudyPlan'), prompt: t('workspace.exampleStudyPlanPrompt') },
            { label: t('workspace.exampleMindMap'), prompt: t('workspace.exampleMindMapPrompt') },
            { label: t('workspace.exampleReview'), prompt: t('workspace.exampleReviewPrompt') },
          ],
    [role, t],
  )

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
    const buckets = new Map<DayBucket, WorkspaceConversation[]>()
    for (const item of filtered) {
      const bucket = dayBucket(item.updatedAt)
      buckets.set(bucket, [...(buckets.get(bucket) ?? []), item])
    }
    return (['today', 'yesterday', 'earlier'] as const).flatMap((bucket) =>
      buckets.has(bucket)
        ? [{ key: dayBucketLabel(bucket, t), items: buckets.get(bucket) ?? [] }]
        : [],
    )
  }, [conversations, courseFilter, query, t])

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
      const normalized = normalizeSkillParams(skill, params ?? {}, locale)
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
      setError(reason instanceof Error ? reason.message : t('workspace.sendFailed'))
    } finally {
      setSending(false)
    }
  }

  async function rename(item: WorkspaceConversation) {
    const next = window.prompt(t('workspace.renamePrompt'), item.title)
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
          <div className={styles.sessionHeading}>{t('workspace.historyHeading')}</div>
          <button className={styles.newChat} type="button" onClick={startNew}>
            {t('workspace.newChat')}
          </button>
          <label className={styles.search}>
            ⌕
            <input
              aria-label={t('workspace.searchHistory')}
              placeholder={t('workspace.searchHistory')}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>
          <select
            className={styles.filter}
            aria-label={t('workspace.filterSessions')}
            value={courseFilter}
            onChange={(event) => setCourseFilter(event.target.value)}
          >
            <option value="">{t('workspace.filterAll')}</option>
            <option value="0">{t('workspace.filterNoCourse')}</option>
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
                      <small>{item.courseTitle || t('workspace.filterNoCourse')}</small>
                    </button>
                    <button
                      className={styles.more}
                      type="button"
                      aria-label={t('workspace.moreActions')}
                      onClick={() => setMenuId(menuId === item.id ? null : item.id)}
                    >
                      ···
                    </button>
                    {menuId === item.id ? (
                      <div className={styles.menu}>
                        <button type="button" onClick={() => void rename(item)}>
                          {t('workspace.rename')}
                        </button>
                        <Popconfirm
                          title={t('workspace.deleteConfirm')}
                          okText={t('workspace.delete')}
                          cancelText={t('workspace.cancel')}
                          onConfirm={() => remove(item)}
                        >
                          <button type="button">{t('workspace.delete')}</button>
                        </Popconfirm>
                      </div>
                    ) : null}
                  </div>
                ))}
              </section>
            ))}
          </div>
          <div className={styles.sidebarFoot}>{t('workspace.sidebarFoot')}</div>
        </div>
      </aside>

      <section className={styles.workspaceMain}>
        <header className={styles.topbar}>
          <div className={styles.topLeft}>
            <button
              className={`${styles.iconBtn} ${styles.sidebarToggle}`}
              type="button"
              aria-label={sidebarOpen ? t('workspace.collapseHistory') : t('workspace.expandHistory')}
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
              {t('workspace.aiReady')}
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
              <p className={styles.subtitle}>{t('workspace.subtitle')}</p>
              <p className={styles.welcomeMeta}>{welcomeMeta}</p>
              <div className={styles.cards}>
                {welcomeSkills.map((item, index) => (
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
                    <strong>{localizedSkillName(locale, item.id, item.name)}</strong>
                    <span>{localizedSkillDescription(locale, item.id, item.description)}</span>
                    <span className={styles.skillArrow} aria-hidden="true">
                      ↗
                    </span>
                  </button>
                ))}
              </div>
              <p className={styles.tipsTitle}>{t('workspace.tipsTitle')}</p>
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
                      {copyId === item.id
                        ? t('workspace.copied')
                        : copyId === -item.id
                          ? t('workspace.copyFailed')
                          : t('workspace.copyContent')}
                    </button>
                  </div>
                ) : (
                  <div key={item.id} className={styles.messageUser}>
                    <div className={styles.bubble}>{item.content}</div>
                  </div>
                ),
              )}
              {sending ? <p className={styles.pending}>{t('workspace.answering')}</p> : null}
              {error ? (
                <p className={styles.error}>
                  {error}
                  <button type="button" onClick={() => void send(draft, skill?.id, skillValues)}>
                    {t('workspace.retry')}
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
              <div className={styles.popup} role="dialog" aria-label={t('workspace.pickSkillAria')}>
                <h3 className={styles.popupTitle}>{t('workspace.pickSkillTitle')}</h3>
                <div className={styles.popupList}>
                  {skills.map((item, index) => (
                    <button
                      key={item.id}
                      type="button"
                      className={styles.popupItem}
                      onClick={() => {
                        setSkill(item)
                        setSkillValues(buildSkillDefaults(item, locale))
                        setSkillPopupOpen(false)
                        promptRef.current?.focus()
                      }}
                    >
                      {SKILL_ICONS[index % SKILL_ICONS.length]} {localizedSkillName(locale, item.id, item.name)}
                      <span className={styles.popupItemDesc}>
                        {localizedSkillDescription(locale, item.id, item.description)}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            ) : null}
            {coursePopupOpen ? (
              <div className={styles.popup} role="dialog" aria-label={t('workspace.linkCourseAria')}>
                <h3 className={styles.popupTitle}>{t('workspace.linkCourseTitle')}</h3>
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
                    {t('workspace.noCourse')}
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
                  <strong>{localizedSkillName(locale, skill.id, skill.name)}</strong>
                  {skill.params.map((param) => {
                    const paramLabel = localizedSkillParamLabel(locale, skill.id, param.key, param.label)
                    return param.type === 'select' ? (
                      <select
                        key={param.key}
                        aria-label={paramLabel}
                        value={skillValues[param.key] ?? ''}
                        onChange={(event) => setSkillValues((current) => ({ ...current, [param.key]: event.target.value }))}
                      >
                        <option value="">{paramLabel}</option>
                        {param.options?.map((option, optionIndex) => (
                          <option key={option} value={option}>
                            {localizedSkillSelectOption(locale, skill.id, param.key, optionIndex, option)}
                          </option>
                        ))}
                      </select>
                    ) : param.type === 'number' ? (
                      <input
                        key={param.key}
                        type="number"
                        aria-label={paramLabel}
                        placeholder={
                          param.min != null && param.max != null
                            ? t('workspace.skillParamRangeHint', { label: paramLabel, min: param.min, max: param.max })
                            : paramLabel
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
                        aria-label={paramLabel}
                        placeholder={paramLabel}
                        value={skillValues[param.key] ?? ''}
                        onChange={(event) => setSkillValues((current) => ({ ...current, [param.key]: event.target.value }))}
                      />
                    )
                  })}
                  <button type="button" onClick={() => setSkill(null)}>
                    {t('workspace.clearSkill')}
                  </button>
                </div>
              ) : null}
              <textarea
                ref={promptRef}
                aria-label={t('workspace.askAria')}
                rows={2}
                value={draft}
                placeholder={t('workspace.askPlaceholder')}
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
                  <button className={`${styles.tool} ${styles.toolAttach}`} type="button" title={t('workspace.attachUnsupported')} disabled>
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
                    <span className={styles.toolLabel}>{t('workspace.skillTool')}</span>
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
                    <span className={styles.toolLabel}>{courseLabel(courses, courseId, t)}</span>
                    <span className={styles.toolCaret} aria-hidden="true">
                      ⌄
                    </span>
                  </button>
                </div>
                <button className={styles.send} type="submit" disabled={!canSend} aria-label={t('workspace.sendAria')}>
                  ↑
                </button>
              </div>
            </div>
          </form>
          <p className={styles.hint}>{t('workspace.disclaimer')}</p>
        </div>
      </section>
    </div>
  )
}
