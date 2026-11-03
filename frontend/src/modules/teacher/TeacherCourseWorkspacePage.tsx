import { useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Progress,
  Radio,
  Segmented,
  Select,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload,
  message,
} from 'antd'
import { ArrowLeftOutlined, DownloadOutlined, PlusOutlined, UploadOutlined } from '@ant-design/icons'
import { Navigate, useNavigate, useParams } from 'react-router-dom'
import * as XLSX from 'xlsx'

import { homePathFor } from '@/core/auth/homePath'
import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import type { AiGenerationMeta, GenerateCourseRequest } from '@/core/api/ai'
import { useAuthStore } from '@/core/auth/store'
import { ApiError } from '@/core/http'
import {
  confirmQuestionImport,
  createTeacherQuestion,
  deleteTeacherQuestion,
  generateCourseQualityReport,
  generateTeacherActivity,
  generateTeacherChapter,
  generateTeacherCourse,
  generateTeacherQuestions,
  getCourseMermaid,
  getCourseQualityReport,
  getTeacherCourse,
  getTeacherQuestions,
  previewQuestionImport,
  saveTeacherCourseContent,
  updateTeacherQuestion,
  type CourseActivity,
  type CourseContent,
  type CourseQualityReport,
  type ImportPreview,
  type ImportQuestionRow,
  type TeacherCourse,
  type TeacherQuestion,
  type TeacherQuestionInput,
} from '@/modules/teacher/api'
import { activityLabel, CourseMarkdownBody } from '@/modules/student/courseContentView'
import { TeacherCourseResourcesTab } from '@/modules/teacher/resources/TeacherCourseResourcesTab'
import styles from '@/modules/teacher/TeacherCourseWorkspacePage.module.css'

const emptyQuestion: TeacherQuestionInput = {
  content: '',
  options: ['', '', '', ''],
  correctAnswer: '',
  explanation: '',
  difficulty: 'medium',
}

const CONTENT_PENDING_MARK = '（待生成）'

function newContent(course: TeacherCourse): CourseContent {
  return {
    schemaVersion: '1.0',
    title: course.title,
    description: course.description,
    audience: 'GENERAL',
    language: 'zh-CN',
    estimatedMinutes: 30,
    chapters: [{
      key: `chapter-${Date.now()}`,
      title: '新章节',
      summary: '',
      order: 1,
      activities: [{
        key: `activity-${Date.now()}`,
        type: 'rich_text',
        title: '学习内容',
        order: 1,
        payload: { markdown: CONTENT_PENDING_MARK, highlights: [] },
      }],
    }],
  }
}

function activityText(activity: CourseActivity) {
  if (activity.type === 'rich_text') return String(activity.payload.markdown ?? '')
  if (activity.type === 'coding') return String(activity.payload.promptMarkdown ?? '')
  if (activity.type === 'mock_interview') return String(activity.payload.scenarioMarkdown ?? '')
  return String(activity.payload.instructionsMarkdown ?? '')
}

function withActivityText(activity: CourseActivity, value: string): CourseActivity {
  const field = activity.type === 'rich_text'
    ? 'markdown'
    : activity.type === 'coding'
      ? 'promptMarkdown'
      : activity.type === 'mock_interview'
        ? 'scenarioMarkdown'
        : 'instructionsMarkdown'
  return { ...activity, payload: { ...activity.payload, [field]: value } }
}

function defaultPayload(type: CourseActivity['type']): Record<string, unknown> {
  if (type === 'coding') return { language: 'python', promptMarkdown: '请填写编程任务。', starterCode: '', solutionHint: '', testCases: [] }
  if (type === 'mock_interview') return {
    role: '面试官',
    scenarioMarkdown: '请填写面试场景。',
    questions: [{ question: '请填写面试问题。', followUpHints: [] }],
    evaluationRubricMarkdown: '根据准确性与表达评分。',
  }
  if (type === 'quiz_game') return {
    instructionsMarkdown: '请选择正确答案。',
    questions: [{ question: '示例题目', options: ['选项 A', '选项 B'], correctIndex: 0, explanation: '请填写解析。' }],
  }
  return { markdown: CONTENT_PENDING_MARK, highlights: [] }
}

function isPendingActivityContent(activity: CourseActivity) {
  const text = activityText(activity).trim()
  if (!text) {
    return true
  }
  return text === CONTENT_PENDING_MARK || text.startsWith('请填写')
}

function contentFillProgress(content: CourseContent) {
  let total = 0
  let pending = 0
  for (const chapter of content.chapters) {
    for (const activity of chapter.activities) {
      total += 1
      if (isPendingActivityContent(activity)) {
        pending += 1
      }
    }
  }
  return { total, pending, filled: total - pending }
}

function findNextPending(content: CourseContent, fromChapter: number, fromActivity: number) {
  const flat: { chapterIndex: number; activityIndex: number }[] = []
  content.chapters.forEach((chapter, chapterIndex) => {
    chapter.activities.forEach((_activity, activityIndex) => {
      flat.push({ chapterIndex, activityIndex })
    })
  })
  if (!flat.length) {
    return null
  }
  const current = flat.findIndex((item) => item.chapterIndex === fromChapter && item.activityIndex === fromActivity)
  const start = current >= 0 ? current : 0
  for (let offset = 1; offset <= flat.length; offset += 1) {
    const point = flat[(start + offset) % flat.length]
    const activity = content.chapters[point.chapterIndex].activities[point.activityIndex]
    if (isPendingActivityContent(activity)) {
      return point
    }
  }
  return null
}

function findPrevPending(content: CourseContent, fromChapter: number, fromActivity: number) {
  const flat: { chapterIndex: number; activityIndex: number }[] = []
  content.chapters.forEach((chapter, chapterIndex) => {
    chapter.activities.forEach((_activity, activityIndex) => {
      flat.push({ chapterIndex, activityIndex })
    })
  })
  if (!flat.length) {
    return null
  }
  const current = flat.findIndex((item) => item.chapterIndex === fromChapter && item.activityIndex === fromActivity)
  const start = current >= 0 ? current : flat.length - 1
  for (let offset = 1; offset <= flat.length; offset += 1) {
    const point = flat[(start - offset + flat.length) % flat.length]
    const activity = content.chapters[point.chapterIndex].activities[point.activityIndex]
    if (isPendingActivityContent(activity)) {
      return point
    }
  }
  return null
}

function listPendingActivities(content: CourseContent) {
  const items: { chapterIndex: number; activityIndex: number }[] = []
  content.chapters.forEach((chapter, chapterIndex) => {
    chapter.activities.forEach((activity, activityIndex) => {
      if (isPendingActivityContent(activity)) {
        items.push({ chapterIndex, activityIndex })
      }
    })
  })
  return items
}

export function TeacherCourseWorkspacePage() {
  const { courseId } = useParams()
  const id = Number(courseId)
  const navigate = useNavigate()
  const currentUser = useAuthStore((state) => state.currentUser)
  const [course, setCourse] = useState<TeacherCourse | null>(null)
  const [content, setContent] = useState<CourseContent | null>(null)
  const [questions, setQuestions] = useState<TeacherQuestion[]>([])
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [questionOpen, setQuestionOpen] = useState(false)
  const [editingQuestion, setEditingQuestion] = useState<TeacherQuestion | null>(null)
  const [questionDraft, setQuestionDraft] = useState<TeacherQuestionInput>(emptyQuestion)
  const [questionSaving, setQuestionSaving] = useState(false)
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
  const [importing, setImporting] = useState(false)
  const [aiOpen, setAiOpen] = useState(false)
  const [aiGenerating, setAiGenerating] = useState(false)
  const [aiQuestions, setAiQuestions] = useState<TeacherQuestionInput[]>([])
  const [fragmentOpen, setFragmentOpen] = useState<'chapter' | 'activity' | null>(null)
  const [fragmentGenerating, setFragmentGenerating] = useState(false)
  const [fragmentTarget, setFragmentTarget] = useState({ chapterKey: '', activityKey: '', topic: '', requirements: '' })
  const [quality, setQuality] = useState<CourseQualityReport | null>(null)
  const [qualityLoading, setQualityLoading] = useState(false)
  const [qualityRubric, setQualityRubric] = useState('')
  const [mermaid, setMermaid] = useState('')
  const [chapterIndex, setChapterIndex] = useState(0)
  const [activityIndex, setActivityIndex] = useState(0)
  const [editPhase, setEditPhase] = useState<'structure' | 'content'>('structure')
  const [outlineOpen, setOutlineOpen] = useState(false)
  const [outlineGenerating, setOutlineGenerating] = useState(false)
  const [fillGenerating, setFillGenerating] = useState(false)
  const [fillBatch, setFillBatch] = useState<{ done: number; total: number } | null>(null)
  const fillBatchAbortRef = useRef(false)
  const [aiForm] = Form.useForm<{ count: number; difficulty: string; extraRequirements?: string }>()
  const [bankTarget, setBankTarget] = useState<{ chapterKey: string; sectionKey: string } | null>(null)
  const [outlineForm] = Form.useForm<GenerateCourseRequest>()
  const { provider, selector } = useAiProviderSelect()

  useEffect(() => {
    if (!Number.isInteger(id) || currentUser?.role !== 'ROLE_TEACHER') return
    Promise.all([getTeacherCourse(id), getTeacherQuestions(id)])
      .then(([nextCourse, nextQuestions]) => {
        setCourse(nextCourse)
        setContent(nextCourse.content ?? newContent(nextCourse))
        setQuestions(nextQuestions)
      })
      .catch((reason: Error) => setError(reason.message))
    getCourseQualityReport(id).then(setQuality).catch(() => undefined)
    getCourseMermaid(id).then((result) => setMermaid(result.source)).catch(() => undefined)
  }, [currentUser?.role, id])

  useEffect(() => {
    if (!content?.chapters.length) {
      return
    }
    if (chapterIndex >= content.chapters.length) {
      setChapterIndex(0)
    }
  }, [chapterIndex, content])

  useEffect(() => {
    const chapter = content?.chapters[chapterIndex]
    if (!chapter?.activities.length) {
      return
    }
    if (activityIndex >= chapter.activities.length) {
      setActivityIndex(0)
    }
  }, [activityIndex, chapterIndex, content])

  useEffect(() => {
    if (bankTarget || !content) {
      return
    }
    for (const chapter of content.chapters) {
      const quiz = chapter.activities.find((activity) => activity.type === 'quiz_game')
      if (quiz) {
        setBankTarget({ chapterKey: chapter.key, sectionKey: quiz.key })
        return
      }
    }
  }, [bankTarget, content])

  const fillProgress = useMemo(
    () => (content ? contentFillProgress(content) : { total: 0, pending: 0, filled: 0 }),
    [content],
  )

  if (!currentUser) return null
  if (currentUser.role !== 'ROLE_TEACHER') return <Navigate to={homePathFor(currentUser.role)} replace />
  if (!Number.isInteger(id) || id <= 0) return <Navigate to="/teacher" replace />

  function updateChapter(index: number, patch: Partial<CourseContent['chapters'][number]>) {
    if (!content) return
    setContent({ ...content, chapters: content.chapters.map((chapter, current) => current === index ? { ...chapter, ...patch } : chapter) })
  }

  function updateActivity(chapterIndex: number, activityIndex: number, next: CourseActivity) {
    if (!content) return
    const chapter = content.chapters[chapterIndex]
    updateChapter(chapterIndex, {
      activities: chapter.activities.map((activity, current) => current === activityIndex ? next : activity),
    })
  }

  async function saveContent() {
    if (!content) return
    setSaving(true)
    setError('')
    try {
      const normalized = {
        ...content,
        chapters: content.chapters.map((chapter, chapterIndex) => ({
          ...chapter,
          order: chapterIndex + 1,
          activities: chapter.activities.map((activity, activityIndex) => ({ ...activity, order: activityIndex + 1 })),
        })),
      }
      const saved = await saveTeacherCourseContent(id, normalized)
      setCourse(saved)
      setContent(saved.content ?? normalized)
      setMermaid((await getCourseMermaid(id)).source)
      setQuestions(await getTeacherQuestions(id))
      message.success('课程内容已保存')
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  function openQuestion(item?: TeacherQuestion) {
    setEditingQuestion(item ?? null)
    setQuestionDraft(item ? {
      content: item.content,
      options: [...item.options],
      correctAnswer: item.correctAnswer,
      explanation: item.explanation,
      difficulty: item.difficulty,
    } : { ...emptyQuestion, options: [...emptyQuestion.options] })
    setQuestionOpen(true)
  }

  async function saveQuestion() {
    setQuestionSaving(true)
    try {
      if (editingQuestion) await updateTeacherQuestion(editingQuestion.id, questionDraft)
      else await createTeacherQuestion(id, {
        ...questionDraft,
        chapterKey: bankTarget?.chapterKey,
        sectionKey: bankTarget?.sectionKey,
      })
      setQuestions(await getTeacherQuestions(id))
      setQuestionOpen(false)
      message.success(editingQuestion ? '题目已更新' : '题目已新增')
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : '保存题目失败')
    } finally {
      setQuestionSaving(false)
    }
  }

  async function removeQuestion(item: TeacherQuestion) {
    try {
      await deleteTeacherQuestion(item.id)
      setQuestions(await getTeacherQuestions(id))
      message.success('题目已删除')
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : '删除题目失败')
    }
  }

  function downloadTemplate() {
    const sheet = XLSX.utils.json_to_sheet([{
      题干: 'Go 中声明变量使用哪个关键字？',
      选项A: 'var',
      选项B: 'func',
      选项C: 'package',
      选项D: 'import',
      正确答案: 'A',
      解析: 'var 用于变量声明。',
      难度: 'easy',
    }])
    sheet['!cols'] = [{ wch: 36 }, { wch: 18 }, { wch: 18 }, { wch: 18 }, { wch: 18 }, { wch: 12 }, { wch: 36 }, { wch: 12 }]
    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, sheet, '选择题')
    XLSX.writeFile(workbook, '选择题导入模板.xlsx')
  }

  async function readImport(file: File) {
    setImporting(true)
    try {
      const workbook = XLSX.read(await file.arrayBuffer(), { type: 'array' })
      const sheet = workbook.Sheets[workbook.SheetNames[0]]
      const records = XLSX.utils.sheet_to_json<Record<string, unknown>>(sheet, { defval: '' })
      const rows: ImportQuestionRow[] = records.map((record, index) => {
        const options = ['选项A', '选项B', '选项C', '选项D', '选项E', '选项F']
          .map((key) => String(record[key] ?? '').trim())
          .filter(Boolean)
        const rawAnswer = String(record.正确答案 ?? '').trim()
        const answerIndex = /^[A-F]$/i.test(rawAnswer) ? rawAnswer.toUpperCase().charCodeAt(0) - 65 : -1
        return {
          row: index + 2,
          content: String(record.题干 ?? ''),
          options,
          correctAnswer: answerIndex >= 0 ? options[answerIndex] ?? rawAnswer : rawAnswer,
          explanation: String(record.解析 ?? ''),
          difficulty: (String(record.难度 ?? 'medium').toLowerCase() as TeacherQuestionInput['difficulty']),
        }
      })
      setImportPreview(await previewQuestionImport(id, rows))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : '无法读取 Excel 文件')
    } finally {
      setImporting(false)
    }
    return false
  }

  async function confirmImport() {
    if (!importPreview || importPreview.errors.length > 0) return
    setImporting(true)
    try {
      await confirmQuestionImport(id, importPreview.rows, {
        chapterKey: bankTarget?.chapterKey,
        sectionKey: bankTarget?.sectionKey,
        creationMethod: 'excel',
      })
      setQuestions(await getTeacherQuestions(id))
      setImportPreview(null)
      message.success(`已导入 ${importPreview.validCount} 道题`)
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : '导入失败')
    } finally {
      setImporting(false)
    }
  }

  async function runAI() {
    const values = await aiForm.validateFields()
    setAiGenerating(true)
    try {
      const result = await generateTeacherQuestions(id, {
        ...values,
        provider,
        chapterKey: bankTarget?.chapterKey,
      })
      setAiQuestions(result.questions)
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : 'AI 生成失败')
    } finally {
      setAiGenerating(false)
    }
  }

  async function adoptAIQuestions() {
    const rows = aiQuestions.map((question, index) => ({ ...question, row: index + 1 }))
    const preview = await previewQuestionImport(id, rows)
    if (preview.errors.length) {
      message.error(preview.errors[0].message)
      return
    }
    await confirmQuestionImport(id, preview.rows, {
      chapterKey: bankTarget?.chapterKey,
      sectionKey: bankTarget?.sectionKey,
      creationMethod: 'ai',
    })
    setQuestions(await getTeacherQuestions(id))
    setAiOpen(false)
    setAiQuestions([])
    message.success(`已采用 ${rows.length} 道 AI 题目`)
  }

  function openOutlineModal() {
    if (!course || !content) {
      return
    }
    outlineForm.setFieldsValue({
      topic: course.title,
      audience: content.audience,
      language: content.language,
      chapterCountHint: content.chapters.length || undefined,
      extraRequirements: '请规划章节与小节标题，小节类型以图文 rich_text 为主。',
    })
    setOutlineOpen(true)
  }

  async function runOutlineGeneration() {
    const values = await outlineForm.validateFields()
    setOutlineGenerating(true)
    try {
      const result = await generateTeacherCourse(id, { ...values, provider, outlineOnly: true })
      setCourse(result.course)
      setContent(result.course.content)
      setChapterIndex(0)
      setActivityIndex(0)
      setEditPhase('structure')
      setOutlineOpen(false)
      const graph = await getCourseMermaid(id)
      setMermaid(graph.source)
      message.success('课程大纲已生成，请确认章与小节后进入内容填充')
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : '大纲生成失败')
    } finally {
      setOutlineGenerating(false)
    }
  }

  async function generateActivityFillAt(
    courseContent: CourseContent,
    targetChapterIndex: number,
    targetActivityIndex: number,
  ) {
    const chapter = courseContent.chapters[targetChapterIndex]
    const activity = chapter?.activities[targetActivityIndex]
    if (!chapter || !activity) {
      throw new Error('小节不存在')
    }
    const result = await generateTeacherActivity(id, {
      provider,
      topic: `${courseContent.title} · ${chapter.title} · ${activity.title}`,
      extraRequirements: '',
      targetChapterKey: chapter.key,
      targetActivityKey: activity.key,
    })
    setCourse(result.course)
    if (!result.course.content) {
      throw new Error('课程内容为空')
    }
    return result.course.content
  }

  async function fillCurrentActivityWithAI() {
    if (!content) {
      return
    }
    setFillGenerating(true)
    try {
      const nextContent = await generateActivityFillAt(content, chapterIndex, activityIndex)
      setContent(nextContent)
      const graph = await getCourseMermaid(id)
      setMermaid(graph.source)
      const activity = content.chapters[chapterIndex].activities[activityIndex]
      message.success(`「${activity.title}」内容已生成`)
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : '小节内容生成失败')
    } finally {
      setFillGenerating(false)
    }
  }

  function cancelFillBatch() {
    fillBatchAbortRef.current = true
  }

  function fillAllPendingWithAI() {
    if (!content || fillGenerating) {
      return
    }
    const queue = listPendingActivities(content)
    if (!queue.length) {
      message.info('没有待生成的小节')
      return
    }
    Modal.confirm({
      title: `一键生成 ${queue.length} 个小节正文？`,
      content: '将按课程目录顺序依次调用 AI。小节较多时耗时与模型用量都会增加，过程中可点击「停止批量生成」。',
      okText: '开始批量生成',
      onOk: () => {
        void (async () => {
          fillBatchAbortRef.current = false
          setFillGenerating(true)
          setFillBatch({ done: 0, total: queue.length })
          let workingContent = content
          let succeeded = 0
          try {
            for (let index = 0; index < queue.length; index += 1) {
              if (fillBatchAbortRef.current) {
                break
              }
              const point = queue[index]
              setChapterIndex(point.chapterIndex)
              setActivityIndex(point.activityIndex)
              setFillBatch({ done: index, total: queue.length })
              try {
                workingContent = await generateActivityFillAt(workingContent, point.chapterIndex, point.activityIndex)
                setContent(workingContent)
                succeeded += 1
                setFillBatch({ done: index + 1, total: queue.length })
              } catch (reason) {
                const chapter = workingContent.chapters[point.chapterIndex]
                const activity = chapter?.activities[point.activityIndex]
                message.error(
                  reason instanceof Error
                    ? `「${activity?.title ?? '小节'}」失败：${reason.message}`
                    : '批量生成已中断',
                )
                break
              }
            }
            const graph = await getCourseMermaid(id)
            setMermaid(graph.source)
            if (fillBatchAbortRef.current) {
              message.warning(`已停止。成功生成 ${succeeded} / ${queue.length} 个小节。`)
            } else if (succeeded === queue.length) {
              message.success(`已全部生成 ${succeeded} 个小节正文`)
            } else if (succeeded > 0) {
              message.warning(`部分完成：成功 ${succeeded} / ${queue.length} 个小节`)
            }
          } finally {
            setFillBatch(null)
            setFillGenerating(false)
          }
        })()
      },
    })
  }

  function jumpToPending(direction: 'next' | 'prev') {
    if (!content) {
      return
    }
    const target = direction === 'next'
      ? findNextPending(content, chapterIndex, activityIndex)
      : findPrevPending(content, chapterIndex, activityIndex)
    if (!target) {
      message.info('没有更多待填充的小节')
      return
    }
    setChapterIndex(target.chapterIndex)
    setActivityIndex(target.activityIndex)
  }

  function openFragment(kind: 'chapter' | 'activity', chapterKey: string, activityKey = '') {
    setFragmentTarget({ chapterKey, activityKey, topic: course?.title ?? '', requirements: '' })
    setFragmentOpen(kind)
  }

  async function runFragmentGeneration() {
    if (!fragmentOpen || !fragmentTarget.chapterKey.trim()) return
    const targetLabel = fragmentOpen === 'chapter'
      ? `章节 ${fragmentTarget.chapterKey}`
      : `活动 ${fragmentTarget.activityKey}`
    Modal.confirm({
      title: `确认替换 ${targetLabel}？`,
      content: '生成成功后会立即写入并替换同 key 内容；该操作不会只保存为预览。',
      okText: '确认生成并替换',
      okButtonProps: { danger: true },
      onOk: async () => {
        setFragmentGenerating(true)
        try {
          const base = {
            provider,
            topic: fragmentTarget.topic,
            extraRequirements: fragmentTarget.requirements,
            targetChapterKey: fragmentTarget.chapterKey.trim(),
          }
          const result = fragmentOpen === 'chapter'
            ? await generateTeacherChapter(id, base)
            : await generateTeacherActivity(id, { ...base, targetActivityKey: fragmentTarget.activityKey.trim() })
          setCourse(result.course)
          setContent(result.course.content)
          setFragmentOpen(null)
          const graph = await getCourseMermaid(id)
          setMermaid(graph.source)
          message.success(`${targetLabel} 已生成并替换`)
        } catch (reason) {
          message.error(reason instanceof Error ? reason.message : '局部生成失败')
        } finally {
          setFragmentGenerating(false)
        }
      },
    })
  }

  async function runQualityReport() {
    setQualityLoading(true)
    try {
      setQuality(await generateCourseQualityReport(id, { provider, rubric: qualityRubric }))
      message.success('质量报告已生成')
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : '质量评估失败')
    } finally {
      setQualityLoading(false)
    }
  }

  if (!course || !content) {
    return <div className={styles.loading}>{error ? <Alert type="error" message={error} /> : <Spin size="large" />}</div>
  }

  const quizSections = content.chapters.flatMap((chapter) =>
    chapter.activities
      .filter((activity) => activity.type === 'quiz_game')
      .map((activity) => ({
        chapterKey: chapter.key,
        chapterTitle: chapter.title,
        sectionKey: activity.key,
        sectionTitle: activity.title,
      })),
  )
  const activeQuiz = quizSections.find((section) =>
    section.chapterKey === bankTarget?.chapterKey && section.sectionKey === bankTarget?.sectionKey,
  ) ?? quizSections[0] ?? null
  const looseQuestions = questions.filter((item) => !item.sectionKey)

  const questionBank = (
    <div>
      {quizSections.length === 0 ? (
        <Alert
          type="info"
          showIcon
          message="还没有测验小节"
          description="先在「课程结构」里把某个小节的类型设为「测验」，保存课程后，再来为这个小节维护题库。"
        />
      ) : (
        <>
          <div className={styles.toolbar}>
            <Space wrap>
              <Select
                style={{ minWidth: 280 }}
                value={activeQuiz ? `${activeQuiz.chapterKey}::${activeQuiz.sectionKey}` : undefined}
                options={quizSections.map((section) => ({
                  value: `${section.chapterKey}::${section.sectionKey}`,
                  label: `${section.chapterTitle} / ${section.sectionTitle}`,
                }))}
                onChange={(value) => {
                  const [chapterKey, sectionKey] = String(value).split('::')
                  setBankTarget({ chapterKey, sectionKey })
                }}
              />
              <Button type="primary" icon={<PlusOutlined />} disabled={!activeQuiz} onClick={() => {
                if (activeQuiz) setBankTarget({ chapterKey: activeQuiz.chapterKey, sectionKey: activeQuiz.sectionKey })
                openQuestion()
              }}>手工新增</Button>
              <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>下载 Excel 模板</Button>
              <Upload accept=".xlsx" showUploadList={false} beforeUpload={(file) => { void readImport(file); return false }}>
                <Button loading={importing} disabled={!activeQuiz} icon={<UploadOutlined />}>导入 .xlsx</Button>
              </Upload>
              <Button disabled={!activeQuiz} onClick={() => { setAiOpen(true); aiForm.setFieldsValue({ count: 5, difficulty: 'medium' }) }}>AI 生成题目</Button>
            </Space>
            <span>当前小节 {questions.filter((item) => item.chapterKey === activeQuiz?.chapterKey && item.sectionKey === activeQuiz?.sectionKey).length} 题</span>
          </div>
          {content.chapters.map((chapter) => {
            const sections = chapter.activities.filter((activity) => activity.type === 'quiz_game')
            if (sections.length === 0) return null
            return (
              <section key={chapter.key} style={{ marginBottom: 28 }}>
                <h3 style={{ marginBottom: 12 }}>{chapter.title}</h3>
                {sections.map((section) => {
                  const rows = questions.filter((item) => item.chapterKey === chapter.key && item.sectionKey === section.key)
                  const selected = activeQuiz?.chapterKey === chapter.key && activeQuiz.sectionKey === section.key
                  return (
                    <div key={section.key} style={{ marginBottom: 16, padding: 12, borderRadius: 12, background: selected ? '#fff7f4' : '#fff' }}>
                      <button
                        type="button"
                        onClick={() => setBankTarget({ chapterKey: chapter.key, sectionKey: section.key })}
                        style={{ fontWeight: 700, marginBottom: 8, background: 'none', border: 0, padding: 0, cursor: 'pointer' }}
                      >
                        {section.title}
                        <span style={{ marginLeft: 8, fontWeight: 500, color: '#8a8178' }}>{rows.length} 题</span>
                      </button>
                      <Table<TeacherQuestion>
                        rowKey="id"
                        dataSource={rows}
                        pagination={false}
                        locale={{ emptyText: '这个测验小节还没有题目' }}
                        columns={[
                          { title: '题干', dataIndex: 'content' },
                          { title: '难度', dataIndex: 'difficulty', width: 100, render: (value: string) => <Tag>{value}</Tag> },
                          { title: '正确答案', dataIndex: 'correctAnswer', width: 180 },
                          { title: '操作', width: 140, render: (_, item) => (
                            <Space size={0}>
                              <Button type="link" onClick={() => openQuestion(item)}>编辑</Button>
                              <Popconfirm title="删除这道题？" okText="删除" cancelText="取消" onConfirm={() => { void removeQuestion(item) }}>
                                <Button type="link" danger>删除</Button>
                              </Popconfirm>
                            </Space>
                          ) },
                        ]}
                      />
                    </div>
                  )
                })}
              </section>
            )
          })}
          {looseQuestions.length > 0 ? (
            <section>
              <h3>未挂到测验小节</h3>
              <Table<TeacherQuestion>
                rowKey="id"
                dataSource={looseQuestions}
                pagination={false}
                columns={[
                  { title: '题干', dataIndex: 'content' },
                  { title: '难度', dataIndex: 'difficulty', width: 100, render: (value: string) => <Tag>{value}</Tag> },
                  { title: '正确答案', dataIndex: 'correctAnswer', width: 180 },
                  { title: '操作', width: 90, render: (_, item) => (
                    <Popconfirm title="删除这道题？" okText="删除" cancelText="取消" onConfirm={() => { void removeQuestion(item) }}>
                      <Button type="link" danger>删除</Button>
                    </Popconfirm>
                  ) },
                ]}
              />
            </section>
          ) : null}
        </>
      )}
    </div>
  )

  const qualityReport = (
    <div className={styles.report}>
      <Card title="AI 课程质量报告" extra={<span>{quality ? new Date(quality.createdAt).toLocaleString() : '尚未生成'}</span>}>
        <div className={styles.reportControls}>
          <div>{selector}</div>
          <Input.TextArea
            rows={2}
            maxLength={1000}
            value={qualityRubric}
            placeholder="可选：补充评分要求"
            onChange={(event) => setQualityRubric(event.target.value)}
          />
          <Button type="primary" loading={qualityLoading} disabled={!content.chapters.length} onClick={() => { void runQualityReport() }}>
            {quality ? '重新生成报告' : '生成报告'}
          </Button>
        </div>
        {quality ? (
          <>
            <div className={styles.overallScore}><Progress type="circle" percent={quality.report.overallScore} /><strong>综合评分</strong></div>
            <div className={styles.dimensionGrid}>
              {quality.report.dimensions.map((item) => (
                <Card size="small" title={dimensionLabel(item.name)} key={item.name}>
                  <Progress percent={item.score} />
                  <div className={styles.reportMarkdown}>
                    <CourseMarkdownBody source={item.comment} codeClassName={styles.codeBlock} />
                  </div>
                </Card>
              ))}
            </div>
            <Typography.Title level={4}>摘要</Typography.Title>
            <div className={styles.reportMarkdown}>
              <CourseMarkdownBody source={quality.report.summaryMarkdown} codeClassName={styles.codeBlock} />
            </div>
            <Typography.Title level={4}>改进建议</Typography.Title>
            {quality.report.suggestions.length
              ? (
                <ol className={styles.suggestionList}>
                  {quality.report.suggestions.map((suggestion, index) => (
                    <li key={`${index}-${suggestion.slice(0, 24)}`}>
                      <CourseMarkdownBody source={suggestion} codeClassName={styles.codeBlock} />
                    </li>
                  ))}
                </ol>
              )
              : <Typography.Paragraph type="secondary">暂无建议</Typography.Paragraph>}
            {quality.meta ? <StudentGenerationMetaLike meta={quality.meta} /> : null}
          </>
        ) : <Alert type="info" showIcon message="生成后将保存并展示最新一份报告" />}
      </Card>
    </div>
  )

  const courseMap = (
    <div className={styles.report}>
      <Card title="课程 Mermaid 导图" extra={<Tag color="green">由 content_json 本地生成，不调用模型</Tag>}>
        <div className={styles.mapGraph}>
          <div className={styles.mapCourse}>{content.title}</div>
          {content.chapters.map((chapter) => (
            <div className={styles.mapChapter} key={chapter.key}>
              <strong>{chapter.title}</strong>
              <div>{chapter.activities.map((activity) => <span key={activity.key}>{activity.title}</span>)}</div>
            </div>
          ))}
        </div>
        <Typography.Title level={5}>Mermaid 源码</Typography.Title>
        <pre className={styles.mermaidSource}>{mermaid || '保存或生成课程内容后即可生成导图。'}</pre>
      </Card>
    </div>
  )

  return (
    <main className={styles.page}>
      <header className={styles.topBar}>
        <div className={styles.topLeft}>
          <Button type="link" className={styles.backBtn} icon={<ArrowLeftOutlined />} onClick={() => navigate('/teacher')}>
            课程中心
          </Button>
          <span className={styles.topSep}>/</span>
          <span className={styles.courseTitle}>{course.title}</span>
        </div>
        <div className={styles.topActions}>
          <Button type="primary" loading={saving} onClick={() => { void saveContent() }}>
            保存课程
          </Button>
        </div>
      </header>

      <div className={styles.pageBody}>
        {error ? <Alert closable type="error" message={error} onClose={() => setError('')} style={{ marginBottom: 16 }} /> : null}
        <Tabs
          className={styles.tabs}
          items={[
            {
              key: 'content',
              label: '课程结构',
              children: (
                <TeacherStructureWorkspace
                  content={content}
                  setContent={setContent}
                  chapterIndex={chapterIndex}
                  activityIndex={activityIndex}
                  setChapterIndex={setChapterIndex}
                  setActivityIndex={setActivityIndex}
                  updateChapter={updateChapter}
                  updateActivity={updateActivity}
                  editPhase={editPhase}
                  setEditPhase={setEditPhase}
                  fillProgress={fillProgress}
                  onOpenOutline={openOutlineModal}
                  onFillCurrentActivity={() => { void fillCurrentActivityWithAI() }}
                  onFillAllPending={fillAllPendingWithAI}
                  onCancelFillBatch={cancelFillBatch}
                  fillBatch={fillBatch}
                  onJumpPending={jumpToPending}
                  onOpenFragment={openFragment}
                  fillGenerating={fillGenerating}
                />
              ),
            },
            {
              key: 'resources',
              label: '课程资料',
              children: <TeacherCourseResourcesTab courseId={id} content={content} />,
            },
            {
              key: 'preview',
              label: '学生视角预览',
              children: (
                <TeacherPreviewWorkspace
                  content={content}
                  chapterIndex={chapterIndex}
                  activityIndex={activityIndex}
                  setChapterIndex={setChapterIndex}
                  setActivityIndex={setActivityIndex}
                />
              ),
            },
            { key: 'map', label: '课程导图', children: courseMap },
            { key: 'quality', label: '质量报告', children: qualityReport },
            { key: 'questions', label: `题库（${questions.length}）`, children: questionBank },
          ]}
        />
      </div>

      <Modal title={editingQuestion ? '编辑选择题' : '新增选择题'} open={questionOpen} width={720} okText="保存" confirmLoading={questionSaving} onOk={() => { void saveQuestion() }} onCancel={() => setQuestionOpen(false)}>
        <QuestionEditor value={questionDraft} onChange={setQuestionDraft} />
      </Modal>

      <Modal title="Excel 导入预检" open={importPreview !== null} width={760} okText="确认并事务导入" okButtonProps={{ disabled: Boolean(importPreview?.errors.length) }} confirmLoading={importing} onOk={() => { void confirmImport() }} onCancel={() => setImportPreview(null)}>
        <Alert type={importPreview?.errors.length ? 'error' : 'success'} showIcon message={
          importPreview?.errors.length ? `发现 ${importPreview.errors.length} 个错误，请修正 Excel 后重新导入` : `${importPreview?.validCount ?? 0} 道题通过预检`
        } />
        {importPreview?.errors.map((item, index) => <p key={`${item.row}-${index}`}>第 {item.row || '—'} 行 · {item.field}：{item.message}</p>)}
      </Modal>

      <Modal title="AI 生成选择题" open={aiOpen} width={820} okText={aiQuestions.length ? `采用 ${aiQuestions.length} 道题` : '生成预览'} confirmLoading={aiGenerating} onOk={() => { if (aiQuestions.length) void adoptAIQuestions(); else void runAI() }} onCancel={() => { setAiOpen(false); setAiQuestions([]) }}>
        <div className={styles.aiProvider}>{selector}</div>
        <Form form={aiForm} layout="vertical">
          <Space size="large" align="start">
            <Form.Item name="count" label="题目数量" rules={[{ required: true }]}><InputNumber min={1} max={20} /></Form.Item>
            <Form.Item name="difficulty" label="难度" rules={[{ required: true }]}><Select style={{ width: 140 }} options={[
              { value: 'easy', label: '简单' }, { value: 'medium', label: '中等' }, { value: 'hard', label: '困难' },
            ]} /></Form.Item>
          </Space>
          <Form.Item name="extraRequirements" label="补充要求"><Input.TextArea rows={2} /></Form.Item>
        </Form>
        {aiQuestions.map((item, index) => <Card size="small" title={`预览题 ${index + 1}`} key={index} className={styles.aiQuestion}>
          <QuestionEditor value={item} onChange={(next) => setAiQuestions(aiQuestions.map((question, current) => current === index ? next : question))} />
        </Card>)}
      </Modal>

      <Modal
        title={fragmentOpen === 'chapter' ? 'AI 生成章节' : 'AI 生成活动'}
        open={fragmentOpen !== null}
        okText="下一步：确认替换"
        confirmLoading={fragmentGenerating}
        onOk={() => { void runFragmentGeneration() }}
        onCancel={() => setFragmentOpen(null)}
      >
        <Alert type="warning" showIcon message="同 key 内容将在确认后被替换；新 key 会追加到课程。" style={{ marginBottom: 16 }} />
        <div className={styles.aiProvider}>{selector}</div>
        <Form layout="vertical">
          <Form.Item label="目标章节 key" required>
            <Input value={fragmentTarget.chapterKey} onChange={(event) => setFragmentTarget({ ...fragmentTarget, chapterKey: event.target.value })} />
          </Form.Item>
          {fragmentOpen === 'activity' ? (
            <Form.Item label="目标活动 key" required>
              <Input value={fragmentTarget.activityKey} onChange={(event) => setFragmentTarget({ ...fragmentTarget, activityKey: event.target.value })} />
            </Form.Item>
          ) : null}
          <Form.Item label="主题">
            <Input value={fragmentTarget.topic} onChange={(event) => setFragmentTarget({ ...fragmentTarget, topic: event.target.value })} />
          </Form.Item>
          <Form.Item label="补充要求">
            <Input.TextArea rows={3} maxLength={2000} showCount value={fragmentTarget.requirements} onChange={(event) => setFragmentTarget({ ...fragmentTarget, requirements: event.target.value })} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="AI 生成课程大纲"
        open={outlineOpen}
        okText="生成大纲"
        confirmLoading={outlineGenerating}
        onOk={() => { void runOutlineGeneration() }}
        onCancel={() => setOutlineOpen(false)}
      >
        <Alert
          type="info"
          showIcon
          message="仅生成章节与小节标题，正文留为「待生成」。确认结构后，在「内容填充」阶段逐节生成正文。"
          style={{ marginBottom: 16 }}
        />
        <div className={styles.aiProvider}>{selector}</div>
        <Form form={outlineForm} layout="vertical">
          <Form.Item name="topic" label="课程主题" rules={[{ required: true, whitespace: true, message: '请输入主题' }]}>
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item name="audience" label="受众">
            <Select options={[
              { value: 'GENERAL', label: '通用' }, { value: 'CS', label: '计算机专业' }, { value: 'NON_CS', label: '非计算机专业' },
            ]} />
          </Form.Item>
          <Form.Item name="language" label="语言">
            <Select options={[{ value: 'zh-CN', label: '简体中文' }, { value: 'en-US', label: 'English' }]} />
          </Form.Item>
          <Form.Item name="chapterCountHint" label="期望章节数">
            <InputNumber min={1} max={20} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="extraRequirements" label="结构要求">
            <Input.TextArea rows={3} maxLength={2000} showCount placeholder="例如：每章 3～5 个小节，先概念后练习" />
          </Form.Item>
        </Form>
      </Modal>
    </main>
  )
}

function TeacherStructureWorkspace({
  content,
  setContent,
  chapterIndex,
  activityIndex,
  setChapterIndex,
  setActivityIndex,
  updateChapter,
  updateActivity,
  editPhase,
  setEditPhase,
  fillProgress,
  onOpenOutline,
  onFillCurrentActivity,
  onFillAllPending,
  onCancelFillBatch,
  fillBatch,
  onJumpPending,
  onOpenFragment,
  fillGenerating,
}: {
  content: CourseContent
  setContent: (value: CourseContent) => void
  chapterIndex: number
  activityIndex: number
  setChapterIndex: (index: number) => void
  setActivityIndex: (index: number) => void
  updateChapter: (index: number, patch: Partial<CourseContent['chapters'][number]>) => void
  updateActivity: (chapterIndex: number, activityIndex: number, next: CourseActivity) => void
  editPhase: 'structure' | 'content'
  setEditPhase: (phase: 'structure' | 'content') => void
  fillProgress: { total: number; pending: number; filled: number }
  onOpenOutline: () => void
  onFillCurrentActivity: () => void
  onFillAllPending: () => void
  onCancelFillBatch: () => void
  fillBatch: { done: number; total: number } | null
  onJumpPending: (direction: 'next' | 'prev') => void
  onOpenFragment: (kind: 'chapter' | 'activity', chapterKey: string, activityKey?: string) => void
  fillGenerating: boolean
}) {
  const chapter = content.chapters[chapterIndex]
  const activity = chapter?.activities[activityIndex]

  if (!chapter || !activity) {
    return <Alert type="info" message="请先新增章节与活动" />
  }

  return (
    <div className={styles.workspace}>
      <div className={styles.workflowBar}>
        <Segmented
          value={editPhase}
          onChange={(value) => setEditPhase(value as 'structure' | 'content')}
          options={[
            { label: '① 结构编排', value: 'structure' },
            {
              label: fillProgress.pending > 0 ? `② 内容填充（${fillProgress.pending} 待生成）` : '② 内容填充',
              value: 'content',
            },
          ]}
        />
        {editPhase === 'structure' ? (
          <Button type="primary" onClick={() => setEditPhase('content')}>
            确认结构，开始填充
          </Button>
        ) : (
          <div className={styles.fillMeter}>
            <span>正文 {fillProgress.filled}/{fillProgress.total}</span>
            <Progress
              percent={fillProgress.total ? Math.round((fillProgress.filled / fillProgress.total) * 100) : 0}
              size="small"
              showInfo={false}
            />
          </div>
        )}
      </div>
      {fillBatch ? (
        <Alert
          type="info"
          showIcon
          className={styles.batchAlert}
          message={`正在批量生成正文（${fillBatch.done}/${fillBatch.total}）`}
          action={
            <Button size="small" danger onClick={onCancelFillBatch}>
              停止批量生成
            </Button>
          }
        />
      ) : null}
      <div className={styles.columns}>
      <aside className={styles.syllabus}>
        <h2>课程目录</h2>
        <ul>
          {content.chapters.map((item, index) => {
            const expanded = index === chapterIndex
            return (
              <li key={item.key} className={expanded ? styles.chapterGroupOpen : styles.chapterGroup}>
                <button
                  type="button"
                  className={expanded ? styles.chapterOn : styles.chapterBtn}
                  onClick={() => {
                    if (index === chapterIndex) {
                      return
                    }
                    setChapterIndex(index)
                    setActivityIndex(0)
                  }}
                >
                  <span className={styles.chapterIndex}>{String(index + 1).padStart(2, '0')}</span>
                  <span className={styles.chapterName}>{item.title}</span>
                </button>
                {expanded && item.activities.length > 0 ? (
                  <ul className={styles.sectionList} aria-label={`${item.title} 小节`}>
                    {item.activities.map((act, actIndex) => (
                      <li key={act.key}>
                        <button
                          type="button"
                          className={actIndex === activityIndex ? styles.sectionOn : styles.sectionBtn}
                          aria-current={actIndex === activityIndex ? 'page' : undefined}
                          onClick={() => setActivityIndex(actIndex)}
                        >
                          <span className={styles.sectionIndex}>{actIndex + 1}</span>
                          <span className={styles.sectionName}>{act.title}</span>
                          {isPendingActivityContent(act) ? (
                            <span className={styles.sectionPending}>待生成</span>
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
        <button
          type="button"
          className={styles.addChapterBtn}
          onClick={() => {
            setContent({
              ...content,
              chapters: [
                ...content.chapters,
                {
                  key: `chapter-${Date.now()}`,
                  title: '新章节',
                  summary: '',
                  order: content.chapters.length + 1,
                  activities: [{
                    key: `activity-${Date.now()}-1`,
                    type: 'rich_text',
                    title: '学习内容',
                    order: 1,
                    payload: defaultPayload('rich_text'),
                  }],
                },
              ],
            })
            setChapterIndex(content.chapters.length)
            setActivityIndex(0)
          }}
        >
          ＋ 新增章节
        </button>
      </aside>

      <section className={styles.contentPane}>
        {editPhase === 'structure' ? (
          <>
            <Alert
              type="info"
              showIcon
              message="第一步：确定章名、小节名与活动类型"
              description="可手工新增，或使用右侧「AI 生成课程大纲」。正文统一在「内容填充」阶段逐节生成或手工编写。"
              style={{ marginBottom: 18 }}
            />
            <p className={styles.contentKicker}>CHAPTER OUTLINE</p>
            <div className={styles.formGrid}>
              <label>章节标题<Input value={chapter.title} onChange={(event) => updateChapter(chapterIndex, { title: event.target.value })} /></label>
              <label>章节摘要<Input value={chapter.summary} onChange={(event) => updateChapter(chapterIndex, { summary: event.target.value })} /></label>
            </div>
            <p className={styles.contentKicker} style={{ marginTop: 22 }}>本章小节</p>
            <ul className={styles.sectionOutlineList}>
              {chapter.activities.map((item, index) => (
                <li
                  key={item.key}
                  className={index === activityIndex ? styles.sectionRowActive : styles.sectionRow}
                >
                  <button type="button" className={styles.sectionRowSelect} onClick={() => setActivityIndex(index)}>
                    编辑
                  </button>
                  <label className={styles.sectionRowTitle}>
                    小节标题
                    <Input
                      value={item.title}
                      onChange={(event) => updateActivity(chapterIndex, index, { ...item, title: event.target.value })}
                    />
                  </label>
                  <label className={styles.sectionRowType}>
                    类型
                    <Select
                      value={item.type}
                      options={[
                        { value: 'rich_text', label: '图文' },
                        { value: 'coding', label: '编程' },
                        { value: 'mock_interview', label: '面试' },
                        { value: 'quiz_game', label: '测验' },
                      ]}
                      onChange={(type: CourseActivity['type']) => updateActivity(chapterIndex, index, { ...item, type, payload: defaultPayload(type) })}
                    />
                  </label>
                  {isPendingActivityContent(item) ? <Tag color="orange">待生成正文</Tag> : <Tag color="green">已有正文</Tag>}
                </li>
              ))}
            </ul>
          </>
        ) : (
          <>
            <p className={styles.contentKicker}>FILL CONTENT</p>
            <h2 className={styles.contentTitle}>{activity.title}</h2>
            <p className={styles.contentType}>{activityLabel(activity.type)} · {chapter.title}</p>
            {isPendingActivityContent(activity) ? (
              <Alert type="warning" showIcon message="本节正文尚未生成" style={{ marginBottom: 16 }} />
            ) : null}
            <label style={{ display: 'grid', gap: 7, fontWeight: 600 }}>
              小节正文
              <Input.TextArea
                rows={12}
                value={activityText(activity)}
                onChange={(event) => updateActivity(chapterIndex, activityIndex, withActivityText(activity, event.target.value))}
              />
            </label>
            <Space wrap style={{ marginTop: 16 }}>
              <Button loading={fillGenerating && !fillBatch} type="primary" onClick={onFillCurrentActivity}>
                AI 生成本节正文
              </Button>
              <Button
                loading={fillGenerating && Boolean(fillBatch)}
                disabled={fillProgress.pending === 0}
                onClick={onFillAllPending}
              >
                一键生成全部待填充（{fillProgress.pending}）
              </Button>
              <Button disabled={fillGenerating} onClick={() => onJumpPending('prev')}>上一待填充</Button>
              <Button disabled={fillGenerating} onClick={() => onJumpPending('next')}>下一待填充</Button>
            </Space>
          </>
        )}
      </section>

      <aside className={styles.toolsPane}>
        <h2>{editPhase === 'structure' ? '结构工具' : '内容工具'}</h2>
        <div className={styles.toolBlock}>
          <h3>课程设置</h3>
          <div className={styles.formGrid} style={{ gridTemplateColumns: '1fr' }}>
            <label>课程简介<Input.TextArea rows={2} value={content.description} onChange={(event) => setContent({ ...content, description: event.target.value })} /></label>
            <label>面向人群<Select value={content.audience} options={[
              { value: 'CS', label: '计算机专业' }, { value: 'NON_CS', label: '非计算机专业' }, { value: 'GENERAL', label: '通用' },
            ]} onChange={(audience) => setContent({ ...content, audience })} /></label>
            <label>语言<Select value={content.language} options={[
              { value: 'zh-CN', label: '简体中文' }, { value: 'en-US', label: 'English' },
            ]} onChange={(language) => setContent({ ...content, language })} /></label>
            <label>预计分钟<InputNumber min={0} style={{ width: '100%' }} value={content.estimatedMinutes} onChange={(value) => setContent({ ...content, estimatedMinutes: value ?? 0 })} /></label>
          </div>
        </div>
        {editPhase === 'structure' ? (
          <div className={styles.toolBlock}>
            <h3>AI 与结构</h3>
            <div className={styles.toolActions}>
              <Button block type="primary" onClick={onOpenOutline}>AI 生成课程大纲</Button>
              <Button block icon={<PlusOutlined />} onClick={() => updateChapter(chapterIndex, { activities: [...chapter.activities, {
                key: `activity-${Date.now()}`, type: 'rich_text', title: '新小节', order: chapter.activities.length + 1, payload: defaultPayload('rich_text'),
              }] })}>新增小节</Button>
              {chapter.activities.length > 1 ? (
                <Popconfirm title="删除当前小节？" onConfirm={() => {
                  updateChapter(chapterIndex, { activities: chapter.activities.filter((_, index) => index !== activityIndex) })
                  setActivityIndex(0)
                }}>
                  <Button block danger>删除当前小节</Button>
                </Popconfirm>
              ) : null}
              {content.chapters.length > 1 ? (
                <Popconfirm title="删除当前章节？" onConfirm={() => {
                  setContent({ ...content, chapters: content.chapters.filter((_, index) => index !== chapterIndex) })
                  setChapterIndex(0)
                  setActivityIndex(0)
                }}>
                  <Button block danger>删除当前章节</Button>
                </Popconfirm>
              ) : null}
            </div>
          </div>
        ) : (
          <div className={styles.toolBlock}>
            <h3>AI 填充</h3>
            <div className={styles.toolActions}>
              <Button block type="primary" loading={fillGenerating && !fillBatch} onClick={onFillCurrentActivity}>AI 生成本节正文</Button>
              <Button
                block
                loading={fillGenerating && Boolean(fillBatch)}
                disabled={fillProgress.pending === 0}
                onClick={onFillAllPending}
              >
                一键生成全部待填充（{fillProgress.pending}）
              </Button>
              <Button block disabled={fillGenerating} onClick={() => onJumpPending('next')}>跳到下一待填充</Button>
              <Button block disabled={fillGenerating} onClick={() => onOpenFragment('activity', chapter.key, activity.key)}>高级：自定义生成参数</Button>
            </div>
          </div>
        )}
        <p className={styles.toolHint}>
          {editPhase === 'structure'
            ? '推荐流程：先定大纲并保存 → 切换到「内容填充」逐节生成正文。'
            : '按目录顺序逐节或一键批量生成；生成后仍可手工修改，记得保存课程。'}
        </p>
      </aside>
      </div>
    </div>
  )
}

function TeacherPreviewWorkspace({
  content,
  chapterIndex,
  activityIndex,
  setChapterIndex,
  setActivityIndex,
}: {
  content: CourseContent
  chapterIndex: number
  activityIndex: number
  setChapterIndex: (index: number) => void
  setActivityIndex: (index: number) => void
}) {
  const chapter = content.chapters[chapterIndex]
  const activity = chapter?.activities[activityIndex]

  if (!chapter || !activity) {
    return <Alert type="info" message="暂无可预览的章节内容" />
  }

  return (
    <div className={styles.workspace}>
      <div className={styles.columns}>
      <aside className={styles.syllabus}>
        <h2>课程目录</h2>
        <ul>
          {content.chapters.map((item, index) => {
            const expanded = index === chapterIndex
            return (
              <li key={item.key} className={expanded ? styles.chapterGroupOpen : styles.chapterGroup}>
                <button
                  type="button"
                  className={expanded ? styles.chapterOn : styles.chapterBtn}
                  onClick={() => {
                    if (index === chapterIndex) {
                      return
                    }
                    setChapterIndex(index)
                    setActivityIndex(0)
                  }}
                >
                  <span className={styles.chapterIndex}>{String(index + 1).padStart(2, '0')}</span>
                  <span className={styles.chapterName}>{item.title}</span>
                </button>
                {expanded && item.activities.length > 0 ? (
                  <ul className={styles.sectionList} aria-label={`${item.title} 小节`}>
                    {item.activities.map((act, actIndex) => (
                      <li key={act.key}>
                        <button
                          type="button"
                          className={actIndex === activityIndex ? styles.sectionOn : styles.sectionBtn}
                          aria-current={actIndex === activityIndex ? 'page' : undefined}
                          onClick={() => setActivityIndex(actIndex)}
                        >
                          <span className={styles.sectionIndex}>{actIndex + 1}</span>
                          <span className={styles.sectionName}>{act.title}</span>
                          {isPendingActivityContent(act) ? (
                            <span className={styles.sectionPending}>待生成</span>
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
      </aside>

      <section className={`${styles.contentPane} ${styles.previewReadOnly}`}>
        <span className={styles.previewBadge}>学生视角预览</span>
        <p className={styles.contentKicker}>CURRENT CHAPTER</p>
        <h1 className={styles.contentTitle}>{activity.title}</h1>
        <p className={styles.contentType}>{activityLabel(activity.type)}</p>
        {chapter.summary ? <p className={styles.chapterSummary}>{chapter.summary}</p> : null}
        <CourseMarkdownBody source={activityText(activity)} codeClassName={styles.codeBlock} />
      </section>

      <aside className={styles.toolsPane}>
        <h2>AI 学习助手</h2>
        <p className={styles.toolHint}>学生端在此使用章节问答、总结与练习生成。教师预览仅展示布局，不调用模型。</p>
        <div className={styles.aiPanelPlaceholder}>
          <span>章节总结</span>
          <span>知识点解释</span>
          <span>生成练习题</span>
          <span>提问与答疑</span>
        </div>
      </aside>
      </div>
    </div>
  )
}

function QuestionEditor({ value, onChange }: { value: TeacherQuestionInput; onChange: (value: TeacherQuestionInput) => void }) {
  return (
    <div className={styles.questionEditor}>
      <label className={styles.questionField}>
        题干
        <Input.TextArea
          rows={3}
          value={value.content}
          className={styles.questionTextArea}
          onChange={(event) => onChange({ ...value, content: event.target.value })}
        />
      </label>
      <label className={styles.questionField}>
        难度
        <Select
          value={value.difficulty}
          options={[
            { value: 'easy', label: '简单' }, { value: 'medium', label: '中等' }, { value: 'hard', label: '困难' },
          ]}
          onChange={(difficulty) => onChange({ ...value, difficulty })}
        />
      </label>
      <div className={styles.options}>
        {value.options.map((option, index) => (
          <div key={index} className={styles.optionRow}>
            <span className={styles.optionLabel}>{String.fromCharCode(65 + index)}</span>
            <Input.TextArea
              rows={2}
              autoSize={{ minRows: 1, maxRows: 6 }}
              className={styles.optionTextArea}
              value={option}
              onChange={(event) => onChange({ ...value, options: value.options.map((item, current) => current === index ? event.target.value : item) })}
            />
          </div>
        ))}
      </div>
      <Space wrap>
        <Button disabled={value.options.length >= 8} onClick={() => onChange({ ...value, options: [...value.options, ''] })}>增加选项</Button>
        <Button disabled={value.options.length <= 2} onClick={() => onChange({ ...value, options: value.options.slice(0, -1) })}>减少选项</Button>
      </Space>
      <label className={styles.questionField}>
        正确答案
        {value.options.some(Boolean) ? (
          <Radio.Group
            className={styles.correctRadioGroup}
            value={value.correctAnswer || undefined}
            onChange={(event) => onChange({ ...value, correctAnswer: event.target.value })}
          >
            {value.options.map((option, index) =>
              option.trim() ? (
                <Radio key={`${index}-${option.slice(0, 24)}`} value={option} className={styles.correctRadio}>
                  <span className={styles.correctRadioText}>
                    <span className={styles.correctRadioLetter}>{String.fromCharCode(65 + index)}.</span>
                    {option}
                  </span>
                </Radio>
              ) : null,
            )}
          </Radio.Group>
        ) : (
          <Typography.Text type="secondary">请先填写选项内容</Typography.Text>
        )}
      </label>
      <label className={styles.questionField}>
        答案解析
        <Input.TextArea
          rows={2}
          className={styles.questionTextArea}
          value={value.explanation}
          onChange={(event) => onChange({ ...value, explanation: event.target.value })}
        />
      </label>
    </div>
  )
}

function dimensionLabel(name: string) {
  return ({ coverage: '内容覆盖', clarity: '表达清晰', practice: '练习设计', audience_fit: '受众匹配' } as Record<string, string>)[name] ?? name
}

function StudentGenerationMetaLike({ meta }: { meta: AiGenerationMeta }) {
  return (
    <Alert
      type={meta.warnings.length || meta.fallbackUsed ? 'warning' : 'success'}
      showIcon
      message={`生成信息 · ${meta.modelUsed || '默认模型'} · ${meta.latencyMs} ms`}
      description={[
        meta.templateId ? `模板 ${meta.templateId}` : '',
        meta.fallbackUsed ? '已使用备用模型' : '',
        ...meta.warnings,
      ].filter(Boolean).join('；') || '评估完成'}
    />
  )
}
