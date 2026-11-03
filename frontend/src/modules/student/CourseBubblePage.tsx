// 课程学习工作区：左目录、中内容、右 AI 助手。
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { Alert, Form, Input, InputNumber, Modal, Select, Typography } from 'antd'

import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import type { AiGenerationMeta, GenerateCourseRequest } from '@/core/api/ai'
import { useAuthStore } from '@/core/auth/store'
import {
  generateStudentCourse,
  getStudentCourse,
  setStudentCourseSharing,
  updateStudentCourseProgress,
  type StudentCourseDetail,
} from '@/modules/student/api'
import type { BubbleNode } from '@/modules/student/bubbleTree'
import { activityLabel, chapterBadgeLabel, CourseMarkdownBody, useActivityNeighbors } from '@/modules/student/courseContentView'
import { AiAssistantPanel } from '@/modules/student/learning/AiAssistantPanel'
import { StudentCourseResourcesPanel } from '@/modules/student/resources/StudentCourseResourcesPanel'
import type { CourseContent } from '@/modules/teacher/api'
import {
  chapterProgress,
  flattenActivities,
  leafPercent,
  listChapters,
  listChapterActivities,
} from '@/modules/student/courseTreeNav'
import styles from '@/modules/student/CourseBubblePage.module.css'
import {
  doneIdsAfterComplete,
  firstActivityOfNextChapter,
  isLastActivityInChapter,
  lessonPrimaryAction,
  mergeProgressPercent,
  nextActivityInSameChapter,
  progressStorageKey,
  readDoneIds,
} from '@/modules/student/learning/lessonProgress'
import {
  aiPixelWidth,
  clampAiRatio,
  defaultAiRatio,
  layoutStorageKey,
  readAiRatio,
  separatorWidth,
  syllabusWidth,
} from '@/modules/student/learning/workspaceLayout'
import { useCourseLearningHeartbeat } from '@/modules/student/learning/useCourseLearningHeartbeat'

type CourseViewTab = 'learn' | 'resources'

function resourceContentFromTree(
  tree: BubbleNode,
  meta: Pick<StudentCourseDetail, 'title' | 'audience' | 'language'>,
): CourseContent {
  const audience =
    meta.audience === 'CS' || meta.audience === 'NON_CS' ? meta.audience : 'GENERAL'
  const language = meta.language === 'en-US' ? 'en-US' : 'zh-CN'
  return {
    schemaVersion: '1.0',
    title: meta.title,
    description: '',
    audience,
    language,
    estimatedMinutes: 0,
    chapters: listChapters(tree).map((chapter, index) => ({
      key: chapter.node.key || `chapter-${index}`,
      title: chapter.node.label,
      summary: chapter.node.note ?? '',
      order: index + 1,
      activities: listChapterActivities(chapter.id, chapter.node).map((activity, actIndex) => ({
        key: activity.node.key || `activity-${actIndex}`,
        type: 'rich_text',
        title: activity.node.label,
        order: actIndex + 1,
        payload: {},
      })),
    })),
  }
}

export function CourseBubblePage() {
  const { courseId } = useParams()
  const currentUser = useAuthStore((state) => state.currentUser)
  const [course, setCourse] = useState<StudentCourseDetail | null>(null)
  const [errorMessage, setErrorMessage] = useState('')
  const [shareError, setShareError] = useState('')
  const [sharing, setSharing] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [generateOpen, setGenerateOpen] = useState(false)
  const [generateError, setGenerateError] = useState('')
  const [generateMeta, setGenerateMeta] = useState<AiGenerationMeta | null>(null)
  const [generateForm] = Form.useForm<GenerateCourseRequest>()
  const [courseViewTab, setCourseViewTab] = useState<CourseViewTab>('learn')
  const {
    provider: aiProvider,
    selector: aiProviderSelector,
    error: aiProviderError,
    optionsAvailable: aiOptionsAvailable,
  } = useAiProviderSelect()

  useCourseLearningHeartbeat(courseId, currentUser?.role === 'ROLE_STUDENT' && Boolean(courseId) && courseId !== 'new')

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_STUDENT' || !courseId || courseId === 'new') {
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
  }, [courseId, currentUser?.role])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_STUDENT' || courseId === 'new') {
    return <Navigate to="/student" replace />
  }

  async function toggleSharing() {
    if (!course || sharing) {
      return
    }
    setSharing(true)
    setShareError('')
    try {
      const result = await setStudentCourseSharing(course.id, !course.shared)
      setCourse({
        ...course,
        shared: result.shared,
        classmateCount: result.classmateCount,
        visibility: result.shared ? 'class' : 'private',
      })
    } catch (error) {
      setShareError(error instanceof Error ? error.message : 'Could not update sharing.')
    } finally {
      setSharing(false)
    }
  }

  async function generateCourseContent() {
    if (!course || generating) return
    const values = await generateForm.validateFields()
    setGenerating(true)
    setGenerateError('')
    setGenerateMeta(null)
    try {
      const result = await generateStudentCourse(course.id, { ...values, provider: aiProvider })
      setCourse(result.course)
      setGenerateMeta(result.meta)
    } catch (error) {
      setGenerateError(error instanceof Error ? error.message : '课程生成失败。')
    } finally {
      setGenerating(false)
    }
  }

  return (
    <div className={styles.page}>
      <header className={styles.topBar}>
        <div className={styles.topLeft}>
          <Link className={styles.backLink} to="/student">
            CaveClass
          </Link>
          <span className={styles.topSep}>/</span>
          <span className={styles.courseTitle}>{course?.title ?? '加载中…'}</span>
        </div>
        <div className={styles.topActions}>
          {course?.owned ? (
            <>
              <span className={styles.aiModel}>AI {aiProviderSelector}</span>
              <button className={styles.ghostBtn} type="button" disabled={sharing} onClick={() => void toggleSharing()}>
                {course.shared ? '改为仅自己' : '对同班公开'}
              </button>
              <button
                className={styles.ghostBtn}
                type="button"
                disabled={generating || !aiOptionsAvailable}
                onClick={() => {
                  setGenerateError('')
                  setGenerateMeta(null)
                  generateForm.setFieldsValue({
                    topic: course?.title,
                    audience: course?.audience || 'GENERAL',
                    language: course?.language || 'zh-CN',
                    extraRequirements: '',
                    chapterCountHint: undefined,
                  })
                  setGenerateOpen(true)
                }}
              >
                {generating ? '生成中…' : course?.tree ? '重新生成整课' : 'AI 生成整课'}
              </button>
            </>
          ) : null}
        </div>
      </header>

      {errorMessage ? <p className={styles.bannerError}>{errorMessage}</p> : null}
      {shareError ? <p className={styles.bannerError}>{shareError}</p> : null}
      {generateError ? <p className={styles.bannerError}>{generateError}</p> : null}
      {aiProviderError ? <p className={styles.bannerError}>{aiProviderError}</p> : null}
      {course?.owned && course.shared && course.classmateCount === 0 ? (
        <p className={styles.bannerNote}>这门课已对同班同学公开，但你还没有同班同学，所以暂时没有其他人能看到。</p>
      ) : null}

      {course ? (
        <>
          <nav className={styles.viewTabs} aria-label="课程视图">
            <button
              type="button"
              role="tab"
              aria-selected={courseViewTab === 'learn'}
              className={courseViewTab === 'learn' ? styles.viewTabActive : styles.viewTab}
              onClick={() => setCourseViewTab('learn')}
            >
              课程学习
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={courseViewTab === 'resources'}
              className={courseViewTab === 'resources' ? styles.viewTabActive : styles.viewTab}
              onClick={() => setCourseViewTab('resources')}
            >
              课程资料
            </button>
          </nav>
          {course.tree ? (
            <CourseWorkspace
              key={course.id}
              courseId={courseId ?? ''}
              courseDbId={course.id}
              tree={course.tree}
              provider={aiProvider}
              userId={currentUser.id}
              initialPercent={course.percent ?? 0}
              viewTab={courseViewTab}
              resourceContent={resourceContentFromTree(course.tree, course)}
            />
          ) : courseViewTab === 'resources' ? (
            <div className={styles.resourcesStandalone}>
              <StudentCourseResourcesPanel courseId={course.id} />
            </div>
          ) : (
            <div className={styles.emptyState}>
              <strong>这门课还没有章节</strong>
              <p>请使用「AI 生成整课」创建章节内容，或等待教师发布完整课程。</p>
            </div>
          )}
        </>
      ) : (
        <div className={styles.loadingState}>正在加载课程…</div>
      )}

      <Modal
        title="AI 生成整课"
        open={generateOpen}
        okText={generating ? '生成中…' : '生成内容'}
        confirmLoading={generating}
        okButtonProps={{ disabled: !aiOptionsAvailable }}
        onCancel={() => setGenerateOpen(false)}
        onOk={() => {
          void generateCourseContent()
        }}
      >
        {generateError ? <Alert type="error" showIcon message={generateError} style={{ marginBottom: 16 }} /> : null}
        {generateMeta ? <StudentGenerationMeta meta={generateMeta} /> : null}
        <Typography.Paragraph type="secondary">将替换当前整课内容，请按学习目标补充生成要求。</Typography.Paragraph>
        <Form form={generateForm} layout="vertical">
          <Form.Item name="topic" label="主题" rules={[{ required: true, whitespace: true, message: '请输入课程主题' }]}>
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item name="audience" label="受众">
            <Select
              options={[
                { value: 'GENERAL', label: '通用' },
                { value: 'CS', label: '计算机专业' },
                { value: 'NON_CS', label: '非计算机专业' },
              ]}
            />
          </Form.Item>
          <Form.Item name="language" label="语言">
            <Select
              options={[
                { value: 'zh-CN', label: '简体中文' },
                { value: 'en-US', label: 'English' },
              ]}
            />
          </Form.Item>
          <Form.Item name="chapterCountHint" label="期望章节数">
            <InputNumber min={1} max={20} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="extraRequirements" label="补充要求">
            <Input.TextArea rows={3} maxLength={2000} showCount />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

function StudentGenerationMeta({ meta }: { meta: AiGenerationMeta }) {
  const summary = [`耗时 ${meta.latencyMs} ms`]
  if (meta.modelUsed) summary.push(`模型 ${meta.modelUsed}`)
  if (meta.fallbackUsed) summary.push('已使用备用模型')
  return (
    <Alert
      type={meta.warnings.length || meta.fallbackUsed ? 'warning' : 'success'}
      showIcon
      message={summary.join(' · ')}
      description={meta.warnings.length ? meta.warnings.join('；') : '内容生成完成'}
      style={{ marginBottom: 16 }}
    />
  )
}

function CourseWorkspace({
  courseId,
  courseDbId,
  tree,
  provider,
  userId,
  initialPercent,
  viewTab,
  resourceContent,
}: {
  courseId: string
  courseDbId: number
  tree: BubbleNode
  provider?: string
  userId: number
  initialPercent: number
  viewTab: CourseViewTab
  resourceContent: CourseContent
}) {
  const chapters = useMemo(() => listChapters(tree), [tree])
  const activities = useMemo(() => flattenActivities(tree), [tree])
  const activityIds = useMemo(() => activities.map((item) => item.id), [activities])

  const [chapterId, setChapterId] = useState(() => chapters[0]?.id ?? '')
  const [activityId, setActivityId] = useState(() => activities[0]?.id ?? '')
  const [doneIds, setDoneIds] = useState<ReadonlySet<string>>(() => new Set())
  const [serverPercent, setServerPercent] = useState(initialPercent)
  const [saving, setSaving] = useState(false)
  const [doneError, setDoneError] = useState('')
  const [showChapterFinish, setShowChapterFinish] = useState(false)
  const [selectedExcerpt, setSelectedExcerpt] = useState('')
  const [selectionPop, setSelectionPop] = useState<{ text: string; top: number; left: number } | null>(null)
  const [explainRequest, setExplainRequest] = useState<{ id: number; text: string } | null>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const columnsRef = useRef<HTMLDivElement>(null)
  const syllabusScrollRef = useRef<HTMLDivElement>(null)
  const syllabusScrollTop = useRef(0)
  const loadedKey = useRef('')
  const explainSeq = useRef(0)
  const ratioRef = useRef(defaultAiRatio)
  const [aiRatio, setAiRatio] = useState(defaultAiRatio)
  const [paneWidth, setPaneWidth] = useState(0)
  const [narrow, setNarrow] = useState(false)
  const [syllabusCollapsed, setSyllabusCollapsed] = useState(false)
  const [dragging, setDragging] = useState(false)

  useEffect(() => {
    if (!chapters.length) {
      return
    }
    if (!chapters.some((item) => item.id === chapterId)) {
      setChapterId(chapters[0].id)
    }
  }, [chapterId, chapters])

  useEffect(() => {
    const chapterActivities = activities.filter((item) => item.chapterId === chapterId)
    if (!chapterActivities.length) {
      return
    }
    if (!chapterActivities.some((item) => item.id === activityId)) {
      setActivityId(chapterActivities[0].id)
    }
  }, [activityId, activities, chapterId])

  useEffect(() => {
    const key = `${userId}:${courseId}`
    if (!courseId || activityIds.length === 0 || loadedKey.current === key) {
      return
    }
    loadedKey.current = key
    try {
      const stored = readDoneIds(localStorage.getItem(progressStorageKey(userId, courseId)), new Set(activityIds))
      setDoneIds(new Set(stored))
    } catch {
      setDoneIds(new Set())
    }
  }, [activityIds, courseId, userId])

  useEffect(() => {
    setServerPercent((current) => mergeProgressPercent(current, initialPercent))
  }, [initialPercent])

  useEffect(() => {
    contentRef.current?.scrollTo({ top: 0 })
    setSelectionPop(null)
    setSelectedExcerpt('')
  }, [activityId])

  useEffect(() => {
    const media = window.matchMedia('(max-width: 1100px)')
    const apply = () => setNarrow(media.matches)
    apply()
    media.addEventListener('change', apply)
    return () => media.removeEventListener('change', apply)
  }, [])

  useEffect(() => {
    try {
      setAiRatio(readAiRatio(localStorage.getItem(layoutStorageKey)))
    } catch {
      setAiRatio(defaultAiRatio)
    }
  }, [])

  useEffect(() => {
    const node = columnsRef.current
    if (!node) {
      return
    }
    const observer = new ResizeObserver(() => setPaneWidth(node.clientWidth))
    observer.observe(node)
    setPaneWidth(node.clientWidth)
    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    if (syllabusCollapsed || !syllabusScrollRef.current) {
      return
    }
    syllabusScrollRef.current.scrollTop = syllabusScrollTop.current
  }, [syllabusCollapsed])

  const resourcesView = viewTab === 'resources'
  const collapsed = syllabusCollapsed && !narrow
  const syllabusTrack = collapsed ? 0 : syllabusWidth
  const available = Math.max(0, paneWidth - syllabusTrack)
  const aiWidth = aiPixelWidth(aiRatio, available)
  ratioRef.current = available > separatorWidth ? aiWidth / (available - separatorWidth) : aiRatio
  const showAiPane = !resourcesView && !narrow

  function rememberRatio(next: number) {
    const safe = clampAiRatio(next)
    setAiRatio(safe)
    try {
      localStorage.setItem(layoutStorageKey, JSON.stringify({ aiRatio: safe }))
    } catch {
      // 浏览器拒绝存储时，本次打开的宽度仍然保留。
    }
  }

  function ratioAt(clientX: number) {
    const node = columnsRef.current
    if (!node) {
      return ratioRef.current
    }
    const rect = node.getBoundingClientRect()
    const room = rect.width - syllabusTrack
    const usable = room - separatorWidth
    if (usable <= 0) {
      return ratioRef.current
    }
    return (rect.right - clientX) / usable
  }

  const activeActivity = activities.find((item) => item.id === activityId) ?? null
  const activeChapter = chapters.find((item) => item.id === chapterId) ?? null
  const percent = mergeProgressPercent(serverPercent, leafPercent(tree, doneIds))
  const { prev } = useActivityNeighbors(activityIds, activityId)
  const sectionDone = Boolean(activityId && doneIds.has(activityId))
  const primary = activityId
    ? lessonPrimaryAction(activityId, activities, sectionDone)
    : { label: '完成并学习下一节 →', action: 'complete-next' as const, disabled: false }
  const nextChapterActivity = activityId ? firstActivityOfNextChapter(activityId, activities) : null
  const showFinishNote =
    Boolean(activeChapter?.node.questionId) &&
    (showChapterFinish || primary.action === 'finished' || primary.action === 'next-chapter' || (sectionDone && isLastActivityInChapter(activityId, activities)))
  const showCourseDone = !nextChapterActivity && (showChapterFinish || primary.action === 'finished')

  async function persistProgress(nextDone: ReadonlySet<string>) {
    if (!courseId) {
      return
    }
    setSaving(true)
    setDoneError('')
    const saved = mergeProgressPercent(serverPercent, leafPercent(tree, nextDone))
    try {
      await updateStudentCourseProgress(courseId, saved)
      setServerPercent(saved)
      try {
        localStorage.setItem(progressStorageKey(userId, courseId), JSON.stringify([...nextDone]))
      } catch {
        // 浏览器拒绝本地存储时，本次会话里的勾选仍然保留。
      }
    } catch (error) {
      setDoneError(error instanceof Error ? error.message : '进度保存失败，请重试。')
      throw error
    } finally {
      setSaving(false)
    }
  }

  function selectActivity(id: string) {
    const activity = activities.find((item) => item.id === id)
    if (!activity) {
      return
    }
    setChapterId(activity.chapterId)
    setActivityId(id)
    setShowChapterFinish(false)
  }

  function selectChapter(id: string) {
    if (id === chapterId) {
      return
    }
    const chapter = chapters.find((item) => item.id === id)
    const first = chapter ? listChapterActivities(id, chapter.node)[0] : undefined
    if (first) {
      selectActivity(first.id)
      return
    }
    setChapterId(id)
    setShowChapterFinish(false)
  }

  function toggleSyllabus() {
    if (!syllabusCollapsed && syllabusScrollRef.current) {
      syllabusScrollTop.current = syllabusScrollRef.current.scrollTop
    }
    setSyllabusCollapsed((current) => !current)
  }

  async function completeAndContinue() {
    if (!activityId || saving || primary.disabled) {
      return
    }
    if (primary.action === 'next') {
      const following = nextActivityInSameChapter(activityId, activities)
      if (following) {
        selectActivity(following)
      }
      return
    }
    if (primary.action === 'next-chapter') {
      if (nextChapterActivity) {
        selectActivity(nextChapterActivity)
      }
      return
    }
    const { next, changed } = doneIdsAfterComplete(doneIds, activityId)
    const following = nextActivityInSameChapter(activityId, activities)
    try {
      if (changed) {
        await persistProgress(next)
        setDoneIds(next)
      }
      if (primary.action === 'complete-next' && following) {
        selectActivity(following)
        return
      }
      setShowChapterFinish(true)
      contentRef.current?.scrollTo({ top: 0 })
    } catch {
      // 保存失败时留在当前小节，错误文案已显示。
    }
  }

  function onContentMouseUp(event: React.MouseEvent<HTMLDivElement>) {
    const root = event.currentTarget
    const target = event.target
    if (!(target instanceof Node) || !root.contains(target)) {
      return
    }
    if (target instanceof Element && target.closest('[data-explain]')) {
      return
    }
    if (target instanceof Element && target.closest('a, button, textarea, input')) {
      setSelectionPop(null)
      return
    }
    const selection = window.getSelection()
    if (!selection || selection.isCollapsed || selection.rangeCount === 0) {
      setSelectionPop(null)
      return
    }
    const text = selection.toString().trim()
    if (text.length < 2) {
      setSelectionPop(null)
      return
    }
    const range = selection.getRangeAt(0)
    if (!root.contains(range.commonAncestorContainer)) {
      setSelectionPop(null)
      return
    }
    const rect = range.getBoundingClientRect()
    const host = root.getBoundingClientRect()
    const excerpt = text.slice(0, 2000)
    setSelectedExcerpt(excerpt)
    setSelectionPop({
      text: excerpt,
      top: Math.max(8, rect.top - host.top + root.scrollTop - 32),
      left: Math.max(8, rect.left - host.left + root.scrollLeft),
    })
  }

  return (
    <div className={styles.workspace}>
      {resourcesView ? null : (
        <div className={styles.progressStrip}>
          <span>学习进度 {percent}%</span>
          <div className={styles.progressTrack} aria-hidden="true">
            <span className={styles.progressFill} style={{ width: `${percent}%` }} />
          </div>
        </div>
      )}

      <div
        className={`${styles.columns}${dragging ? ` ${styles.columnsDragging}` : ''}${resourcesView ? ` ${styles.columnsResources}` : ''}`}
        ref={columnsRef}
        style={
          narrow || paneWidth <= 0
            ? undefined
            : resourcesView
              ? {
                  gridTemplateColumns: collapsed ? 'minmax(0, 1fr)' : `${syllabusWidth}px minmax(0, 1fr)`,
                }
              : showAiPane
                ? {
                    gridTemplateColumns: collapsed
                      ? `minmax(0, 1fr) ${separatorWidth}px ${aiWidth}px`
                      : `${syllabusWidth}px minmax(0, 1fr) ${separatorWidth}px ${aiWidth}px`,
                  }
                : {
                    gridTemplateColumns: collapsed ? 'minmax(0, 1fr)' : `${syllabusWidth}px minmax(0, 1fr)`,
                  }
        }
      >
        {collapsed ? null : (
        <aside className={styles.syllabus} aria-label="课程目录">
          <div className={styles.syllabusHead}>
            {collapsed ? null : <h2>课程目录</h2>}
            <button
              className={styles.collapseBtn}
              type="button"
              aria-label="收起目录"
              aria-expanded="true"
              onClick={toggleSyllabus}
            >
              ‹
            </button>
          </div>
          <div className={styles.syllabusScroll} ref={syllabusScrollRef}>
          <ul>
            {chapters.map((chapter) => {
              const status = chapterProgress(chapter.id, chapter.node, doneIds, chapterId)
              const badge = chapterBadgeLabel(status)
              const expanded = chapter.id === chapterId
              const sections = expanded ? listChapterActivities(chapter.id, chapter.node) : []
              return (
                <li key={chapter.id} className={expanded ? styles.chapterGroupOpen : styles.chapterGroup}>
                  <button
                    type="button"
                    className={expanded ? styles.chapterOn : styles.chapterBtn}
                    onClick={() => selectChapter(chapter.id)}
                  >
                    <span className={styles.chapterIndex}>{String(chapter.index + 1).padStart(2, '0')}</span>
                    <span className={styles.chapterName}>{chapter.node.label}</span>
                    {badge ? (
                      <span className={styles.chapterBadge} data-status={status}>
                        {badge}
                      </span>
                    ) : null}
                  </button>
                  {expanded && sections.length > 0 ? (
                    <ul className={styles.sectionList} aria-label={`${chapter.node.label} 小节`}>
                      {sections.map((item, index) => (
                        <li key={item.id}>
                          <button
                            type="button"
                            className={item.id === activityId ? styles.sectionOn : styles.sectionBtn}
                            aria-current={item.id === activityId ? 'page' : undefined}
                            onClick={() => selectActivity(item.id)}
                          >
                            <span className={styles.sectionIndex}>{index + 1}</span>
                            <span className={styles.sectionName}>{item.node.label}</span>
                            {doneIds.has(item.id) ? (
                              <span className={styles.sectionDone} aria-label="已完成">
                                ✓
                              </span>
                            ) : null}
                          </button>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </li>
              )
            })}
          </ul>
          </div>
        </aside>
        )}

        <main className={styles.contentPane}>
          {collapsed ? (
            <div className={styles.contentTools}>
              <button className={styles.collapseBtn} type="button" aria-label="展开目录" onClick={toggleSyllabus}>
                ›
              </button>
            </div>
          ) : null}
          {resourcesView ? (
            <div className={styles.resourcesPane}>
              <StudentCourseResourcesPanel courseId={courseDbId} content={resourceContent} />
            </div>
          ) : activeChapter && activeActivity ? (
            <>
              <div className={styles.contentScroll} ref={contentRef} onMouseUp={onContentMouseUp}>
                <div className={styles.reading}>
                <p className={styles.contentKicker}>CURRENT CHAPTER</p>
                <h1 className={styles.contentTitle}>{activeActivity.node.label}</h1>
                <p className={styles.contentChapter}>{activeChapter.node.label}</p>
                {activeActivity.node.activityType ? (
                  <p className={styles.contentType}>{activityLabel(activeActivity.node.activityType)}</p>
                ) : null}
                {activeChapter.node.note ? (
                  <p className={styles.chapterSummary}>{activeChapter.node.note}</p>
                ) : null}

                <div className={styles.contentBody}>
                  <CourseMarkdownBody source={activeActivity.node.note} codeClassName={styles.codeBlock} />
                </div>

                {activeChapter.node.questionId ? (
                  <Link className={styles.quizLink} to={`/student/questions/${activeChapter.node.questionId}`}>
                    进入本章测验
                  </Link>
                ) : null}
                </div>
                {selectionPop ? (
                  <button
                    type="button"
                    className={styles.explainPop}
                    data-explain="true"
                    style={{ top: selectionPop.top, left: selectionPop.left }}
                    onMouseDown={(event) => event.preventDefault()}
                    onClick={() => {
                      explainSeq.current += 1
                      setExplainRequest({ id: explainSeq.current, text: selectionPop.text })
                      setSelectionPop(null)
                      window.getSelection()?.removeAllRanges()
                    }}
                  >
                    AI 解释
                  </button>
                ) : null}
              </div>

              <div className={styles.contentFoot}>
                {doneError ? <p className={styles.doneError}>{doneError}</p> : null}
                {showCourseDone || showFinishNote ? (
                  <div className={styles.finishNote}>
                    {showCourseDone ? <p>你已学完这门课的全部小节。</p> : null}
                    {showFinishNote && activeChapter.node.questionId ? (
                      <Link className={styles.quizLink} to={`/student/questions/${activeChapter.node.questionId}`}>
                        进入本章测验
                      </Link>
                    ) : null}
                  </div>
                ) : null}
                <div className={styles.footActions}>
                  <button
                    type="button"
                    className={styles.navBtn}
                    disabled={!prev || saving}
                    onClick={() => prev && selectActivity(prev)}
                  >
                    上一节
                  </button>
                  <button
                    type="button"
                    className={styles.doneBtn}
                    disabled={saving || primary.disabled}
                    onClick={() => void completeAndContinue()}
                  >
                    {saving ? '保存中…' : primary.label}
                  </button>
                </div>
              </div>
            </>
          ) : (
            <p className={styles.emptyLesson}>本章暂无活动内容。</p>
          )}
        </main>

        {showAiPane ? (
          <div
            className={styles.paneSplit}
            role="separator"
            aria-orientation="vertical"
            aria-label="调整正文和 AI 助手宽度"
            aria-valuemin={15}
            aria-valuemax={70}
            aria-valuenow={Math.round(ratioRef.current * 100)}
            tabIndex={0}
            onPointerDown={(event) => {
              event.preventDefault()
              event.currentTarget.setPointerCapture(event.pointerId)
              setDragging(true)
            }}
            onPointerMove={(event) => {
              if (!event.currentTarget.hasPointerCapture(event.pointerId)) {
                return
              }
              setAiRatio(clampAiRatio(ratioAt(event.clientX)))
            }}
            onPointerUp={(event) => {
              if (event.currentTarget.hasPointerCapture(event.pointerId)) {
                event.currentTarget.releasePointerCapture(event.pointerId)
              }
              setDragging(false)
              rememberRatio(ratioAt(event.clientX))
            }}
            onKeyDown={(event) => {
              if (event.key === 'ArrowLeft') {
                event.preventDefault()
                rememberRatio(ratioRef.current - 0.02)
              }
              if (event.key === 'ArrowRight') {
                event.preventDefault()
                rememberRatio(ratioRef.current + 0.02)
              }
            }}
          />
        ) : null}

        {showAiPane ? (
          <AiAssistantPanel
            courseId={courseId}
            chapterKey={activeActivity?.chapterKey ?? activeChapter?.node.key ?? ''}
            activityKey={activeActivity?.node.key ?? ''}
            chapterTitle={activeChapter?.node.label ?? ''}
            sectionTitle={activeActivity?.node.label ?? ''}
            provider={provider}
            selectedExcerpt={selectedExcerpt}
            explainRequest={explainRequest}
          />
        ) : null}
      </div>
    </div>
  )
}
