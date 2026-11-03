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
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'
import type { AppLocale } from '@/core/i18n/types'
import { useI18n } from '@/core/i18n/useI18n'
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
import { CourseBubbleMap } from '@/modules/student/CourseBubbleMap'
import { contentToBubbleTree } from '@/modules/student/contentToBubbleTree'
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

const LEGACY_CONTENT_PENDING_MARK = '（待生成）'
const LEGACY_FILL_PREFIX = '请填写'

const LEGACY_IMPORT_COLS = {
  stem: '题干',
  options: ['选项A', '选项B', '选项C', '选项D', '选项E', '选项F'],
  correctAnswer: '正确答案',
  explanation: '解析',
  difficulty: '难度',
} as const

function contentPendingMark(locale = getAppLocale()) {
  return translateApp(locale, 'teacher.workspace.pendingMark')
}

function importColKeys(locale: AppLocale) {
  return {
    stem: translateApp(locale, 'teacher.workspace.importColStem'),
    options: (['A', 'B', 'C', 'D', 'E', 'F'] as const).map((letter) =>
      translateApp(locale, `teacher.workspace.importColOption${letter}`),
    ),
    correctAnswer: translateApp(locale, 'teacher.workspace.importColCorrectAnswer'),
    explanation: translateApp(locale, 'teacher.workspace.importColExplanation'),
    difficulty: translateApp(locale, 'teacher.workspace.importColDifficulty'),
  }
}

function importField(record: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = record[key]
    if (value !== undefined && String(value).trim() !== '') {
      return String(value)
    }
  }
  return ''
}

function parseImportRecord(record: Record<string, unknown>, index: number, locale: AppLocale): ImportQuestionRow {
  const cols = importColKeys(locale)
  const options = cols.options
    .map((key, optionIndex) => importField(record, [key, LEGACY_IMPORT_COLS.options[optionIndex]]))
    .filter(Boolean)
  const rawAnswer = importField(record, [cols.correctAnswer, LEGACY_IMPORT_COLS.correctAnswer]).trim()
  const answerIndex = /^[A-F]$/i.test(rawAnswer) ? rawAnswer.toUpperCase().charCodeAt(0) - 65 : -1
  return {
    row: index + 2,
    content: importField(record, [cols.stem, LEGACY_IMPORT_COLS.stem]),
    options,
    correctAnswer: answerIndex >= 0 ? options[answerIndex] ?? rawAnswer : rawAnswer,
    explanation: importField(record, [cols.explanation, LEGACY_IMPORT_COLS.explanation]),
    difficulty: (importField(record, [cols.difficulty, LEGACY_IMPORT_COLS.difficulty]) || 'medium').toLowerCase() as TeacherQuestionInput['difficulty'],
  }
}

function newContent(course: TeacherCourse, locale = getAppLocale()): CourseContent {
  return {
    schemaVersion: '1.0',
    title: course.title,
    description: course.description,
    audience: 'GENERAL',
    language: 'zh-CN',
    estimatedMinutes: 30,
    chapters: [{
      key: `chapter-${Date.now()}`,
      title: translateApp(locale, 'teacher.workspace.newChapter'),
      summary: '',
      order: 1,
      activities: [{
        key: `activity-${Date.now()}`,
        type: 'rich_text',
        title: translateApp(locale, 'teacher.workspace.learningContent'),
        order: 1,
        payload: { markdown: contentPendingMark(locale), highlights: [] },
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

function defaultPayload(type: CourseActivity['type'], locale = getAppLocale()): Record<string, unknown> {
  if (type === 'coding') {
    return {
      language: 'python',
      promptMarkdown: translateApp(locale, 'teacher.workspace.defaultCodingPrompt'),
      starterCode: '',
      solutionHint: '',
      testCases: [],
    }
  }
  if (type === 'mock_interview') {
    return {
      role: translateApp(locale, 'teacher.workspace.defaultInterviewRole'),
      scenarioMarkdown: translateApp(locale, 'teacher.workspace.defaultInterviewScenario'),
      questions: [{ question: translateApp(locale, 'teacher.workspace.defaultInterviewQuestion'), followUpHints: [] }],
      evaluationRubricMarkdown: translateApp(locale, 'teacher.workspace.defaultInterviewRubric'),
    }
  }
  if (type === 'quiz_game') {
    return {
      instructionsMarkdown: translateApp(locale, 'teacher.workspace.defaultQuizInstructions'),
      questions: [{
        question: translateApp(locale, 'teacher.workspace.defaultQuizSampleQuestion'),
        options: [
          translateApp(locale, 'teacher.workspace.defaultQuizOptionA'),
          translateApp(locale, 'teacher.workspace.defaultQuizOptionB'),
        ],
        correctIndex: 0,
        explanation: translateApp(locale, 'teacher.workspace.defaultQuizExplanation'),
      }],
    }
  }
  return { markdown: contentPendingMark(locale), highlights: [] }
}

function isPendingFillText(text: string, locale = getAppLocale()) {
  const trimmed = text.trim()
  if (!trimmed) {
    return true
  }
  const fillPrefix = translateApp(locale, 'teacher.workspace.fillPrefix')
  return (
    trimmed === contentPendingMark(locale)
    || trimmed === LEGACY_CONTENT_PENDING_MARK
    || trimmed.startsWith(LEGACY_FILL_PREFIX)
    || trimmed.startsWith(fillPrefix)
  )
}

function isPendingActivityContent(activity: CourseActivity, locale = getAppLocale()) {
  return isPendingFillText(activityText(activity), locale)
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
  const { t, locale } = useI18n()
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
        setContent(nextCourse.content ?? newContent(nextCourse, getAppLocale()))
        setQuestions(nextQuestions)
      })
      .catch((reason: Error) => setError(reason.message))
    getCourseQualityReport(id).then(setQuality).catch(() => undefined)
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

  const bubbleTree = useMemo(() => (content ? contentToBubbleTree(content) : null), [content])

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
      setQuestions(await getTeacherQuestions(id))
      message.success(t('teacher.workspace.contentSaved'))
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : t('teacher.workspace.saveContentFailed'))
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
      message.success(editingQuestion ? t('teacher.workspace.questionUpdated') : t('teacher.workspace.questionCreated'))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.saveQuestionFailed'))
    } finally {
      setQuestionSaving(false)
    }
  }

  async function removeQuestion(item: TeacherQuestion) {
    try {
      await deleteTeacherQuestion(item.id)
      setQuestions(await getTeacherQuestions(id))
      message.success(t('teacher.workspace.questionDeleted'))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.deleteQuestionFailed'))
    }
  }

  function downloadTemplate() {
    const cols = importColKeys(locale)
    const sheet = XLSX.utils.json_to_sheet([{
      [cols.stem]: t('teacher.workspace.excelSampleStem'),
      [cols.options[0]]: 'var',
      [cols.options[1]]: 'func',
      [cols.options[2]]: 'package',
      [cols.options[3]]: 'import',
      [cols.correctAnswer]: 'A',
      [cols.explanation]: t('teacher.workspace.excelSampleExplanation'),
      [cols.difficulty]: 'easy',
    }])
    sheet['!cols'] = [{ wch: 36 }, { wch: 18 }, { wch: 18 }, { wch: 18 }, { wch: 18 }, { wch: 12 }, { wch: 36 }, { wch: 12 }]
    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, sheet, t('teacher.workspace.excelSheetName'))
    XLSX.writeFile(workbook, t('teacher.workspace.excelTemplateFilename'))
  }

  async function readImport(file: File) {
    setImporting(true)
    try {
      const workbook = XLSX.read(await file.arrayBuffer(), { type: 'array' })
      const sheet = workbook.Sheets[workbook.SheetNames[0]]
      const records = XLSX.utils.sheet_to_json<Record<string, unknown>>(sheet, { defval: '' })
      const rows: ImportQuestionRow[] = records.map((record, index) => parseImportRecord(record, index, locale))
      setImportPreview(await previewQuestionImport(id, rows))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.readExcelFailed'))
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
      message.success(t('teacher.workspace.importedCount', { count: importPreview.validCount }))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.importFailed'))
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
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.aiGenerateFailed'))
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
    message.success(t('teacher.workspace.adoptedAiCount', { count: rows.length }))
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
      extraRequirements: translateApp(locale, 'teacher.workspace.outlineExtraRequirementsDefault'),
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
      message.success(t('teacher.workspace.outlineGenerated'))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.outlineGenerateFailed'))
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
      throw new Error(t('teacher.workspace.activityNotFound'))
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
      throw new Error(t('teacher.workspace.contentEmpty'))
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
      const activity = content.chapters[chapterIndex].activities[activityIndex]
      message.success(t('teacher.workspace.activityGenerated', { title: activity.title }))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.activityFillFailed'))
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
      message.info(t('teacher.workspace.noPendingSections'))
      return
    }
    Modal.confirm({
      title: t('teacher.workspace.batchConfirmTitle', { count: queue.length }),
      content: t('teacher.workspace.batchConfirmContent'),
      okText: t('teacher.workspace.batchConfirmOk'),
      cancelText: t('common.cancel'),
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
                    ? t('teacher.workspace.batchItemFailed', {
                      title: activity?.title ?? t('teacher.workspace.batchActivityFallback'),
                      message: reason.message,
                    })
                    : t('teacher.workspace.batchInterrupted'),
                )
                break
              }
            }
            if (fillBatchAbortRef.current) {
              message.warning(t('teacher.workspace.batchStopped', { succeeded, total: queue.length }))
            } else if (succeeded === queue.length) {
              message.success(t('teacher.workspace.batchAllDone', { count: succeeded }))
            } else if (succeeded > 0) {
              message.warning(t('teacher.workspace.batchPartial', { succeeded, total: queue.length }))
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
      message.info(t('teacher.workspace.noMorePending'))
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
      ? t('teacher.workspace.fragmentChapterLabel', { key: fragmentTarget.chapterKey })
      : t('teacher.workspace.fragmentActivityLabel', { key: fragmentTarget.activityKey })
    Modal.confirm({
      title: t('teacher.workspace.fragmentConfirmTitle', { target: targetLabel }),
      content: t('teacher.workspace.fragmentConfirmContent'),
      okText: t('teacher.workspace.fragmentConfirmOk'),
      cancelText: t('common.cancel'),
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
          message.success(t('teacher.workspace.fragmentGenerated', { target: targetLabel }))
        } catch (reason) {
          message.error(reason instanceof Error ? reason.message : t('teacher.workspace.fragmentGenerateFailed'))
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
      message.success(t('teacher.workspace.qualityReportGenerated'))
    } catch (reason) {
      message.error(reason instanceof Error ? reason.message : t('teacher.workspace.qualityReportFailed'))
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

  const difficultyOptions = [
    { value: 'easy', label: t('teacher.workspace.difficultyEasy') },
    { value: 'medium', label: t('teacher.workspace.difficultyMedium') },
    { value: 'hard', label: t('teacher.workspace.difficultyHard') },
  ]

  const audienceOptions = [
    { value: 'GENERAL', label: t('teacher.workspace.audienceGeneral') },
    { value: 'CS', label: t('teacher.workspace.audienceCS') },
    { value: 'NON_CS', label: t('teacher.workspace.audienceNonCS') },
  ]

  const languageOptions = [
    { value: 'zh-CN', label: t('teacher.workspace.langZhCN') },
    { value: 'en-US', label: t('teacher.workspace.langEnUS') },
  ]

  const questionBank = (
    <div>
      {quizSections.length === 0 ? (
        <Alert
          type="info"
          showIcon
          message={t('teacher.workspace.noQuizSectionTitle')}
          description={t('teacher.workspace.noQuizSectionDesc')}
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
              }}>{t('teacher.workspace.addManual')}</Button>
              <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>{t('teacher.workspace.downloadTemplate')}</Button>
              <Upload accept=".xlsx" showUploadList={false} beforeUpload={(file) => { void readImport(file); return false }}>
                <Button loading={importing} disabled={!activeQuiz} icon={<UploadOutlined />}>{t('teacher.workspace.importXlsx')}</Button>
              </Upload>
              <Button disabled={!activeQuiz} onClick={() => { setAiOpen(true); aiForm.setFieldsValue({ count: 5, difficulty: 'medium' }) }}>{t('teacher.workspace.aiGenerateQuestions')}</Button>
            </Space>
            <span>{t('teacher.workspace.currentSectionQuestionCount', {
              count: questions.filter((item) => item.chapterKey === activeQuiz?.chapterKey && item.sectionKey === activeQuiz?.sectionKey).length,
            })}</span>
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
                        <span style={{ marginLeft: 8, fontWeight: 500, color: '#8a8178' }}>{t('teacher.workspace.questionCountShort', { count: rows.length })}</span>
                      </button>
                      <Table<TeacherQuestion>
                        rowKey="id"
                        dataSource={rows}
                        pagination={false}
                        locale={{ emptyText: t('teacher.workspace.emptyQuizTable') }}
                        columns={[
                          { title: t('teacher.workspace.colStem'), dataIndex: 'content' },
                          { title: t('teacher.workspace.colDifficulty'), dataIndex: 'difficulty', width: 100, render: (value: string) => <Tag>{value}</Tag> },
                          { title: t('teacher.workspace.colCorrectAnswer'), dataIndex: 'correctAnswer', width: 180 },
                          { title: t('teacher.workspace.colActions'), width: 140, render: (_, item) => (
                            <Space size={0}>
                              <Button type="link" onClick={() => openQuestion(item)}>{t('teacher.workspace.edit')}</Button>
                              <Popconfirm title={t('teacher.workspace.deleteQuestionConfirm')} okText={t('common.delete')} cancelText={t('common.cancel')} onConfirm={() => { void removeQuestion(item) }}>
                                <Button type="link" danger>{t('common.delete')}</Button>
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
              <h3>{t('teacher.workspace.looseQuestionsTitle')}</h3>
              <Table<TeacherQuestion>
                rowKey="id"
                dataSource={looseQuestions}
                pagination={false}
                columns={[
                  { title: t('teacher.workspace.colStem'), dataIndex: 'content' },
                  { title: t('teacher.workspace.colDifficulty'), dataIndex: 'difficulty', width: 100, render: (value: string) => <Tag>{value}</Tag> },
                  { title: t('teacher.workspace.colCorrectAnswer'), dataIndex: 'correctAnswer', width: 180 },
                  { title: t('teacher.workspace.colActions'), width: 90, render: (_, item) => (
                    <Popconfirm title={t('teacher.workspace.deleteQuestionConfirm')} okText={t('common.delete')} cancelText={t('common.cancel')} onConfirm={() => { void removeQuestion(item) }}>
                      <Button type="link" danger>{t('common.delete')}</Button>
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
      <Card title={t('teacher.workspace.qualityCardTitle')} extra={<span>{quality ? new Date(quality.createdAt).toLocaleString() : t('teacher.workspace.notGeneratedYet')}</span>}>
        <div className={styles.reportControls}>
          <div>{selector}</div>
          <Input.TextArea
            rows={2}
            maxLength={1000}
            value={qualityRubric}
            placeholder={t('teacher.workspace.rubricPlaceholder')}
            onChange={(event) => setQualityRubric(event.target.value)}
          />
          <Button type="primary" loading={qualityLoading} disabled={!content.chapters.length} onClick={() => { void runQualityReport() }}>
            {quality ? t('teacher.workspace.regenerateReport') : t('teacher.workspace.generateReport')}
          </Button>
        </div>
        {quality ? (
          <>
            <div className={styles.overallScore}><Progress type="circle" percent={quality.report.overallScore} /><strong>{t('teacher.workspace.overallScore')}</strong></div>
            <div className={styles.dimensionGrid}>
              {quality.report.dimensions.map((item) => (
                <Card size="small" title={dimensionLabel(item.name, locale)} key={item.name}>
                  <Progress percent={item.score} />
                  <div className={styles.reportMarkdown}>
                    <CourseMarkdownBody source={item.comment} codeClassName={styles.codeBlock} />
                  </div>
                </Card>
              ))}
            </div>
            <Typography.Title level={4}>{t('teacher.workspace.summaryTitle')}</Typography.Title>
            <div className={styles.reportMarkdown}>
              <CourseMarkdownBody source={quality.report.summaryMarkdown} codeClassName={styles.codeBlock} />
            </div>
            <Typography.Title level={4}>{t('teacher.workspace.suggestionsTitle')}</Typography.Title>
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
              : <Typography.Paragraph type="secondary">{t('teacher.workspace.noSuggestions')}</Typography.Paragraph>}
            {quality.meta ? <StudentGenerationMetaLike meta={quality.meta} /> : null}
          </>
        ) : <Alert type="info" showIcon message={t('teacher.workspace.afterGenerateHint')} />}
      </Card>
    </div>
  )

  const courseMap = bubbleTree ? (
    <div className={styles.mapTab}>
      <CourseBubbleMap tree={bubbleTree} />
    </div>
  ) : (
    <div className={styles.mapTabEmpty}>{t('teacher.workspace.mapEmpty')}</div>
  )

  return (
    <main className={styles.page}>
      <header className={styles.topBar}>
        <div className={styles.topLeft}>
          <Button type="link" className={styles.backBtn} icon={<ArrowLeftOutlined />} onClick={() => navigate('/teacher')}>
            {t('teacher.workspace.courseHub')}
          </Button>
          <span className={styles.topSep}>/</span>
          <span className={styles.courseTitle}>{course.title}</span>
        </div>
        <div className={styles.topActions}>
          <Button type="primary" loading={saving} onClick={() => { void saveContent() }}>
            {t('teacher.workspace.saveCourse')}
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
              label: t('teacher.workspace.tabStructure'),
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
              label: t('teacher.workspace.tabResources'),
              children: <TeacherCourseResourcesTab courseId={id} content={content} />,
            },
            {
              key: 'preview',
              label: t('teacher.workspace.tabPreview'),
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
            { key: 'map', label: t('teacher.workspace.tabMap'), children: courseMap },
            { key: 'quality', label: t('teacher.workspace.tabQuality'), children: qualityReport },
            { key: 'questions', label: t('teacher.workspace.tabQuestionBank', { count: questions.length }), children: questionBank },
          ]}
        />
      </div>

      <Modal title={editingQuestion ? t('teacher.workspace.editQuestion') : t('teacher.workspace.addQuestion')} open={questionOpen} width={720} okText={t('common.save')} confirmLoading={questionSaving} onOk={() => { void saveQuestion() }} onCancel={() => setQuestionOpen(false)}>
        <QuestionEditor value={questionDraft} onChange={setQuestionDraft} />
      </Modal>

      <Modal title={t('teacher.workspace.importPreviewTitle')} open={importPreview !== null} width={760} okText={t('teacher.workspace.confirmImport')} okButtonProps={{ disabled: Boolean(importPreview?.errors.length) }} confirmLoading={importing} onOk={() => { void confirmImport() }} onCancel={() => setImportPreview(null)}>
        <Alert type={importPreview?.errors.length ? 'error' : 'success'} showIcon message={
          importPreview?.errors.length
            ? t('teacher.workspace.importErrors', { count: importPreview.errors.length })
            : t('teacher.workspace.importValid', { count: importPreview?.validCount ?? 0 })
        } />
        {importPreview?.errors.map((item, index) => (
          <p key={`${item.row}-${index}`}>
            {t('teacher.workspace.importErrorRow', {
              row: item.row || t('common.dash'),
              field: item.field,
              message: item.message,
            })}
          </p>
        ))}
      </Modal>

      <Modal title={t('teacher.workspace.aiQuestionsTitle')} open={aiOpen} width={820} okText={aiQuestions.length ? t('teacher.workspace.adoptCount', { count: aiQuestions.length }) : t('teacher.workspace.generatePreview')} confirmLoading={aiGenerating} onOk={() => { if (aiQuestions.length) void adoptAIQuestions(); else void runAI() }} onCancel={() => { setAiOpen(false); setAiQuestions([]) }}>
        <div className={styles.aiProvider}>{selector}</div>
        <Form form={aiForm} layout="vertical">
          <Space size="large" align="start">
            <Form.Item name="count" label={t('teacher.workspace.questionCount')} rules={[{ required: true }]}><InputNumber min={1} max={20} /></Form.Item>
            <Form.Item name="difficulty" label={t('teacher.workspace.difficulty')} rules={[{ required: true }]}><Select style={{ width: 140 }} options={difficultyOptions} /></Form.Item>
          </Space>
          <Form.Item name="extraRequirements" label={t('teacher.workspace.extraRequirements')}><Input.TextArea rows={2} /></Form.Item>
        </Form>
        {aiQuestions.map((item, index) => <Card size="small" title={t('teacher.workspace.previewQuestion', { index: index + 1 })} key={index} className={styles.aiQuestion}>
          <QuestionEditor value={item} onChange={(next) => setAiQuestions(aiQuestions.map((question, current) => current === index ? next : question))} />
        </Card>)}
      </Modal>

      <Modal
        title={fragmentOpen === 'chapter' ? t('teacher.workspace.aiChapter') : t('teacher.workspace.aiActivity')}
        open={fragmentOpen !== null}
        okText={t('teacher.workspace.nextConfirmReplace')}
        confirmLoading={fragmentGenerating}
        onOk={() => { void runFragmentGeneration() }}
        onCancel={() => setFragmentOpen(null)}
      >
        <Alert type="warning" showIcon message={t('teacher.workspace.fragmentWarning')} style={{ marginBottom: 16 }} />
        <div className={styles.aiProvider}>{selector}</div>
        <Form layout="vertical">
          <Form.Item label={t('teacher.workspace.targetChapterKey')} required>
            <Input value={fragmentTarget.chapterKey} onChange={(event) => setFragmentTarget({ ...fragmentTarget, chapterKey: event.target.value })} />
          </Form.Item>
          {fragmentOpen === 'activity' ? (
            <Form.Item label={t('teacher.workspace.targetActivityKey')} required>
              <Input value={fragmentTarget.activityKey} onChange={(event) => setFragmentTarget({ ...fragmentTarget, activityKey: event.target.value })} />
            </Form.Item>
          ) : null}
          <Form.Item label={t('teacher.workspace.topic')}>
            <Input value={fragmentTarget.topic} onChange={(event) => setFragmentTarget({ ...fragmentTarget, topic: event.target.value })} />
          </Form.Item>
          <Form.Item label={t('teacher.workspace.extraRequirements')}>
            <Input.TextArea rows={3} maxLength={2000} showCount value={fragmentTarget.requirements} onChange={(event) => setFragmentTarget({ ...fragmentTarget, requirements: event.target.value })} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('teacher.workspace.outlineTitle')}
        open={outlineOpen}
        okText={t('teacher.workspace.generateOutline')}
        confirmLoading={outlineGenerating}
        onOk={() => { void runOutlineGeneration() }}
        onCancel={() => setOutlineOpen(false)}
      >
        <Alert
          type="info"
          showIcon
          message={t('teacher.workspace.outlineInfo')}
          style={{ marginBottom: 16 }}
        />
        <div className={styles.aiProvider}>{selector}</div>
        <Form form={outlineForm} layout="vertical">
          <Form.Item name="topic" label={t('teacher.workspace.courseTopic')} rules={[{ required: true, whitespace: true, message: t('teacher.workspace.topicRequired') }]}>
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item name="audience" label={t('teacher.workspace.audience')}>
            <Select options={audienceOptions} />
          </Form.Item>
          <Form.Item name="language" label={t('teacher.workspace.language')}>
            <Select options={languageOptions} />
          </Form.Item>
          <Form.Item name="chapterCountHint" label={t('teacher.workspace.chapterCountHint')}>
            <InputNumber min={1} max={20} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="extraRequirements" label={t('teacher.workspace.structureRequirements')}>
            <Input.TextArea rows={3} maxLength={2000} showCount placeholder={t('teacher.workspace.structureRequirementsPlaceholder')} />
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
  const { t } = useI18n()
  const chapter = content.chapters[chapterIndex]
  const activity = chapter?.activities[activityIndex]

  const activityTypeOptions = [
    { value: 'rich_text', label: t('teacher.workspace.activityTypeRichText') },
    { value: 'coding', label: t('teacher.workspace.activityTypeCoding') },
    { value: 'mock_interview', label: t('teacher.workspace.activityTypeMockInterview') },
    { value: 'quiz_game', label: t('teacher.workspace.activityTypeQuizGame') },
  ]

  const audienceOptions = [
    { value: 'CS', label: t('teacher.workspace.audienceCS') },
    { value: 'NON_CS', label: t('teacher.workspace.audienceNonCS') },
    { value: 'GENERAL', label: t('teacher.workspace.audienceGeneral') },
  ]

  const languageOptions = [
    { value: 'zh-CN', label: t('teacher.workspace.langZhCN') },
    { value: 'en-US', label: t('teacher.workspace.langEnUS') },
  ]

  if (!chapter || !activity) {
    return <Alert type="info" message={t('teacher.workspace.addChapterSectionFirst')} />
  }

  return (
    <div className={styles.workspace}>
      <div className={styles.workflowBar}>
        <Segmented
          value={editPhase}
          onChange={(value) => setEditPhase(value as 'structure' | 'content')}
          options={[
            { label: t('teacher.workspace.phaseStructure'), value: 'structure' },
            {
              label: fillProgress.pending > 0
                ? t('teacher.workspace.phaseContentPending', { count: fillProgress.pending })
                : t('teacher.workspace.phaseContent'),
              value: 'content',
            },
          ]}
        />
        {editPhase === 'structure' ? (
          <Button type="primary" onClick={() => setEditPhase('content')}>
            {t('teacher.workspace.confirmStructureStartFill')}
          </Button>
        ) : (
          <div className={styles.fillMeter}>
            <span>{t('teacher.workspace.bodyProgress', { filled: fillProgress.filled, total: fillProgress.total })}</span>
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
          message={t('teacher.workspace.batchGenerating', { done: fillBatch.done, total: fillBatch.total })}
          action={
            <Button size="small" danger onClick={onCancelFillBatch}>
              {t('teacher.workspace.stopBatch')}
            </Button>
          }
        />
      ) : null}
      <div className={styles.columns}>
      <aside className={styles.syllabus}>
        <h2>{t('teacher.workspace.syllabusTitle')}</h2>
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
                  <ul className={styles.sectionList} aria-label={t('teacher.workspace.sectionListAria', { title: item.title })}>
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
                            <span className={styles.sectionPending}>{t('teacher.workspace.pendingBadge')}</span>
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
                  title: translateApp(getAppLocale(), 'teacher.workspace.newChapter'),
                  summary: '',
                  order: content.chapters.length + 1,
                  activities: [{
                    key: `activity-${Date.now()}-1`,
                    type: 'rich_text',
                    title: translateApp(getAppLocale(), 'teacher.workspace.learningContent'),
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
          {t('teacher.workspace.addChapter')}
        </button>
      </aside>

      <section className={styles.contentPane}>
        {editPhase === 'structure' ? (
          <>
            <Alert
              type="info"
              showIcon
              message={t('teacher.workspace.structureStepTitle')}
              description={t('teacher.workspace.structureStepDesc')}
              style={{ marginBottom: 18 }}
            />
            <p className={styles.contentKicker}>{t('teacher.workspace.kickerChapterOutline')}</p>
            <div className={styles.formGrid}>
              <label>{t('teacher.workspace.chapterTitle')}<Input value={chapter.title} onChange={(event) => updateChapter(chapterIndex, { title: event.target.value })} /></label>
              <label>{t('teacher.workspace.chapterSummary')}<Input value={chapter.summary} onChange={(event) => updateChapter(chapterIndex, { summary: event.target.value })} /></label>
            </div>
            <p className={styles.contentKicker} style={{ marginTop: 22 }}>{t('teacher.workspace.kickerSections')}</p>
            <ul className={styles.sectionOutlineList}>
              {chapter.activities.map((item, index) => (
                <li
                  key={item.key}
                  className={index === activityIndex ? styles.sectionRowActive : styles.sectionRow}
                >
                  <button type="button" className={styles.sectionRowSelect} onClick={() => setActivityIndex(index)}>
                    {t('teacher.workspace.edit')}
                  </button>
                  <label className={styles.sectionRowTitle}>
                    {t('teacher.workspace.sectionTitle')}
                    <Input
                      value={item.title}
                      onChange={(event) => updateActivity(chapterIndex, index, { ...item, title: event.target.value })}
                    />
                  </label>
                  <label className={styles.sectionRowType}>
                    {t('teacher.workspace.type')}
                    <Select
                      value={item.type}
                      options={activityTypeOptions}
                      onChange={(type: CourseActivity['type']) => updateActivity(chapterIndex, index, { ...item, type, payload: defaultPayload(type) })}
                    />
                  </label>
                  {isPendingActivityContent(item) ? <Tag color="orange">{t('teacher.workspace.pendingBodyTag')}</Tag> : <Tag color="green">{t('teacher.workspace.hasBodyTag')}</Tag>}
                </li>
              ))}
            </ul>
          </>
        ) : (
          <>
            <p className={styles.contentKicker}>{t('teacher.workspace.kickerFillContent')}</p>
            <h2 className={styles.contentTitle}>{activity.title}</h2>
            <p className={styles.contentType}>{activityLabel(activity.type)} · {chapter.title}</p>
            {isPendingActivityContent(activity) ? (
              <Alert type="warning" showIcon message={t('teacher.workspace.sectionBodyNotGenerated')} style={{ marginBottom: 16 }} />
            ) : null}
            <label style={{ display: 'grid', gap: 7, fontWeight: 600 }}>
              {t('teacher.workspace.sectionBody')}
              <Input.TextArea
                rows={12}
                value={activityText(activity)}
                onChange={(event) => updateActivity(chapterIndex, activityIndex, withActivityText(activity, event.target.value))}
              />
            </label>
            <Space wrap style={{ marginTop: 16 }}>
              <Button loading={fillGenerating && !fillBatch} type="primary" onClick={onFillCurrentActivity}>
                {t('teacher.workspace.aiFillSection')}
              </Button>
              <Button
                loading={fillGenerating && Boolean(fillBatch)}
                disabled={fillProgress.pending === 0}
                onClick={onFillAllPending}
              >
                {t('teacher.workspace.fillAllPending', { count: fillProgress.pending })}
              </Button>
              <Button disabled={fillGenerating} onClick={() => onJumpPending('prev')}>{t('teacher.workspace.prevPending')}</Button>
              <Button disabled={fillGenerating} onClick={() => onJumpPending('next')}>{t('teacher.workspace.nextPending')}</Button>
            </Space>
          </>
        )}
      </section>

      <aside className={styles.toolsPane}>
        <h2>{editPhase === 'structure' ? t('teacher.workspace.toolsStructure') : t('teacher.workspace.toolsContent')}</h2>
        <div className={styles.toolBlock}>
          <h3>{t('teacher.workspace.courseSettings')}</h3>
          <div className={styles.formGrid} style={{ gridTemplateColumns: '1fr' }}>
            <label>{t('teacher.workspace.courseDescription')}<Input.TextArea rows={2} value={content.description} onChange={(event) => setContent({ ...content, description: event.target.value })} /></label>
            <label>{t('teacher.workspace.audience')}<Select value={content.audience} options={audienceOptions} onChange={(audience) => setContent({ ...content, audience })} /></label>
            <label>{t('teacher.workspace.language')}<Select value={content.language} options={languageOptions} onChange={(language) => setContent({ ...content, language })} /></label>
            <label>{t('teacher.workspace.estimatedMinutes')}<InputNumber min={0} style={{ width: '100%' }} value={content.estimatedMinutes} onChange={(value) => setContent({ ...content, estimatedMinutes: value ?? 0 })} /></label>
          </div>
        </div>
        {editPhase === 'structure' ? (
          <div className={styles.toolBlock}>
            <h3>{t('teacher.workspace.aiAndStructure')}</h3>
            <div className={styles.toolActions}>
              <Button block type="primary" onClick={onOpenOutline}>{t('teacher.workspace.aiOutline')}</Button>
              <Button block icon={<PlusOutlined />} onClick={() => updateChapter(chapterIndex, { activities: [...chapter.activities, {
                key: `activity-${Date.now()}`, type: 'rich_text', title: translateApp(getAppLocale(), 'teacher.workspace.newSection'), order: chapter.activities.length + 1, payload: defaultPayload('rich_text'),
              }] })}>{t('teacher.workspace.addSection')}</Button>
              {chapter.activities.length > 1 ? (
                <Popconfirm title={t('teacher.workspace.deleteSectionConfirm')} onConfirm={() => {
                  updateChapter(chapterIndex, { activities: chapter.activities.filter((_, index) => index !== activityIndex) })
                  setActivityIndex(0)
                }}>
                  <Button block danger>{t('teacher.workspace.deleteCurrentSection')}</Button>
                </Popconfirm>
              ) : null}
              {content.chapters.length > 1 ? (
                <Popconfirm title={t('teacher.workspace.deleteChapterConfirm')} onConfirm={() => {
                  setContent({ ...content, chapters: content.chapters.filter((_, index) => index !== chapterIndex) })
                  setChapterIndex(0)
                  setActivityIndex(0)
                }}>
                  <Button block danger>{t('teacher.workspace.deleteCurrentChapter')}</Button>
                </Popconfirm>
              ) : null}
            </div>
          </div>
        ) : (
          <div className={styles.toolBlock}>
            <h3>{t('teacher.workspace.aiFill')}</h3>
            <div className={styles.toolActions}>
              <Button block type="primary" loading={fillGenerating && !fillBatch} onClick={onFillCurrentActivity}>{t('teacher.workspace.aiFillSection')}</Button>
              <Button
                block
                loading={fillGenerating && Boolean(fillBatch)}
                disabled={fillProgress.pending === 0}
                onClick={onFillAllPending}
              >
                {t('teacher.workspace.fillAllPending', { count: fillProgress.pending })}
              </Button>
              <Button block disabled={fillGenerating} onClick={() => onJumpPending('next')}>{t('teacher.workspace.jumpNextPending')}</Button>
              <Button block disabled={fillGenerating} onClick={() => onOpenFragment('activity', chapter.key, activity.key)}>{t('teacher.workspace.advancedCustomGenerate')}</Button>
            </div>
          </div>
        )}
        <p className={styles.toolHint}>
          {editPhase === 'structure'
            ? t('teacher.workspace.hintStructure')
            : t('teacher.workspace.hintContent')}
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
  const { t } = useI18n()
  const chapter = content.chapters[chapterIndex]
  const activity = chapter?.activities[activityIndex]

  if (!chapter || !activity) {
    return <Alert type="info" message={t('teacher.workspace.noPreviewContent')} />
  }

  return (
    <div className={`${styles.workspace} ${styles.previewWorkspace}`}>
      <div className={styles.columns}>
        <aside className={styles.syllabus}>
          <h2>{t('teacher.workspace.syllabusTitle')}</h2>
          <div className={styles.previewSyllabusScroll}>
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
                      <ul className={styles.sectionList} aria-label={t('teacher.workspace.sectionListAria', { title: item.title })}>
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
                                <span className={styles.sectionPending}>{t('teacher.workspace.pendingBadge')}</span>
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

        <section className={`${styles.contentPane} ${styles.previewReadOnly}`}>
          <div className={styles.previewContentScroll}>
            <div className={styles.previewReading}>
              <span className={styles.previewBadge}>{t('teacher.workspace.previewBadge')}</span>
              <p className={styles.contentKicker}>{t('teacher.workspace.kickerCurrentChapter')}</p>
              <h1 className={styles.contentTitle}>{activity.title}</h1>
              <p className={styles.contentType}>{activityLabel(activity.type)}</p>
              {chapter.summary ? <p className={styles.chapterSummary}>{chapter.summary}</p> : null}
              <div className={styles.previewContentBody}>
                <CourseMarkdownBody source={activityText(activity)} codeClassName={styles.codeBlock} />
              </div>
            </div>
          </div>
        </section>

        <aside className={styles.toolsPane}>
          <h2>{t('teacher.workspace.aiAssistantTitle')}</h2>
          <div className={styles.previewToolsScroll}>
            <p className={styles.toolHint}>{t('teacher.workspace.previewHint')}</p>
            <div className={styles.aiPanelPlaceholder}>
              <span>{t('teacher.workspace.placeholderSummary')}</span>
              <span>{t('teacher.workspace.placeholderExplain')}</span>
              <span>{t('teacher.workspace.placeholderPractice')}</span>
              <span>{t('teacher.workspace.placeholderQa')}</span>
            </div>
          </div>
        </aside>
      </div>
    </div>
  )
}

function QuestionEditor({ value, onChange }: { value: TeacherQuestionInput; onChange: (value: TeacherQuestionInput) => void }) {
  const { t } = useI18n()
  const difficultyOptions = [
    { value: 'easy', label: t('teacher.workspace.difficultyEasy') },
    { value: 'medium', label: t('teacher.workspace.difficultyMedium') },
    { value: 'hard', label: t('teacher.workspace.difficultyHard') },
  ]

  return (
    <div className={styles.questionEditor}>
      <label className={styles.questionField}>
        {t('teacher.workspace.stem')}
        <Input.TextArea
          rows={3}
          value={value.content}
          className={styles.questionTextArea}
          onChange={(event) => onChange({ ...value, content: event.target.value })}
        />
      </label>
      <label className={styles.questionField}>
        {t('teacher.workspace.difficulty')}
        <Select
          value={value.difficulty}
          options={difficultyOptions}
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
        <Button disabled={value.options.length >= 8} onClick={() => onChange({ ...value, options: [...value.options, ''] })}>{t('teacher.workspace.addOption')}</Button>
        <Button disabled={value.options.length <= 2} onClick={() => onChange({ ...value, options: value.options.slice(0, -1) })}>{t('teacher.workspace.removeOption')}</Button>
      </Space>
      <label className={styles.questionField}>
        {t('teacher.workspace.correctAnswer')}
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
          <Typography.Text type="secondary">{t('teacher.workspace.fillOptionsFirst')}</Typography.Text>
        )}
      </label>
      <label className={styles.questionField}>
        {t('teacher.workspace.explanation')}
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

function dimensionLabel(name: string, locale = getAppLocale()) {
  const keys: Record<string, string> = {
    coverage: 'teacher.workspace.dimensionCoverage',
    clarity: 'teacher.workspace.dimensionClarity',
    practice: 'teacher.workspace.dimensionPractice',
    audience_fit: 'teacher.workspace.dimensionAudienceFit',
  }
  const key = keys[name]
  return key ? translateApp(locale, key) : name
}

function StudentGenerationMetaLike({ meta }: { meta: AiGenerationMeta }) {
  const { t } = useI18n()
  return (
    <Alert
      type={meta.warnings.length || meta.fallbackUsed ? 'warning' : 'success'}
      showIcon
      message={t('teacher.workspace.generationInfo', {
        model: meta.modelUsed || t('teacher.workspace.defaultModel'),
        latency: meta.latencyMs,
      })}
      description={[
        meta.templateId ? t('teacher.workspace.templateId', { id: meta.templateId }) : '',
        meta.fallbackUsed ? t('teacher.workspace.fallbackUsed') : '',
        ...meta.warnings,
      ].filter(Boolean).join('；') || t('teacher.workspace.evalComplete')}
    />
  )
}
