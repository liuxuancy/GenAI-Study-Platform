// 讲师首页：课程列表、新建、勾选自己所在的班发布。
import { useEffect, useState } from 'react'
import { Alert, Checkbox, Form, Input, InputNumber, Modal, Radio, Select, Typography } from 'antd'
import { Navigate, useNavigate } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import { GenerationMetaAlert } from '@/core/ai/GenerationMetaAlert'
import type { AiGenerationMeta, GenerateCourseRequest } from '@/core/api/ai'
import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'
import { ApiError } from '@/core/http'
import {
  createTeacherCourse,
  getTeacherClasses,
  generateTeacherCourse,
  getTeacherCourses,
  publishTeacherCourse,
  type TeacherClass,
  type TeacherCourse,
} from '@/modules/teacher/api'
import styles from '@/modules/teacher/TeacherHomePage.module.css'

export function TeacherHomePage() {
  const { t } = useI18n()
  const currentUser = useAuthStore((state) => state.currentUser)
  const navigate = useNavigate()
  const [courses, setCourses] = useState<TeacherCourse[] | null>(null)
  const [classes, setClasses] = useState<TeacherClass[]>([])
  const [loadError, setLoadError] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState('')
  const [publishCourse, setPublishCourse] = useState<TeacherCourse | null>(null)
  const [selectedClassIds, setSelectedClassIds] = useState<number[]>([])
  const [publishing, setPublishing] = useState(false)
  const [publishError, setPublishError] = useState('')
  const [form] = Form.useForm<{ title: string; description?: string }>()
  const [generatingId, setGeneratingId] = useState<number | null>(null)
  const [generateCourse, setGenerateCourse] = useState<TeacherCourse | null>(null)
  const [generateError, setGenerateError] = useState('')
  const [generateMeta, setGenerateMeta] = useState<AiGenerationMeta | null>(null)
  const [generateForm] = Form.useForm<GenerateCourseRequest>()
  const [generateMode, setGenerateMode] = useState<'outline' | 'full'>('outline')
  const {
    provider: aiProvider,
    selector: aiProviderSelector,
    error: aiProviderError,
    optionsAvailable: aiOptionsAvailable,
  } = useAiProviderSelect()

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_TEACHER') {
      return
    }
    let cancelled = false
    Promise.all([getTeacherCourses(), getTeacherClasses()])
      .then(([courseItems, classItems]) => {
        if (!cancelled) {
          setCourses(courseItems)
          setClasses(classItems)
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setLoadError(error.message)
        }
      })
    return () => {
      cancelled = true
    }
  }, [currentUser?.role])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_TEACHER') {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }

  const classNameById = new Map(classes.map((item) => [item.id, item.name]))

  function reloadCourses() {
    return getTeacherCourses()
      .then((items) => {
        setCourses(items)
      })
      .catch((error: Error) => setLoadError(error.message))
  }

  function openPublish(course: TeacherCourse) {
    const known = new Set(classes.map((item) => item.id))
    setPublishCourse(course)
    setSelectedClassIds(course.classIds.filter((id) => known.has(id)))
    setPublishError('')
  }

  async function submitCreate() {
    const values = await form.validateFields()
    setCreating(true)
    setCreateError('')
    try {
      await createTeacherCourse({
        title: values.title.trim(),
        description: values.description?.trim() ?? '',
      })
      setCreateOpen(false)
      form.resetFields()
      await reloadCourses()
    } catch (error) {
      setCreateError(error instanceof ApiError ? error.message : t('teacher.home.createFailed'))
    } finally {
      setCreating(false)
    }
  }

  async function submitPublish() {
    if (!publishCourse) {
      return
    }
    setPublishing(true)
    setPublishError('')
    try {
      await publishTeacherCourse(publishCourse.id, selectedClassIds)
      setPublishCourse(null)
      await reloadCourses()
    } catch (error) {
      setPublishError(error instanceof ApiError ? error.message : t('teacher.home.publishFailed'))
    } finally {
      setPublishing(false)
    }
  }

  const courseItems = courses ?? []
  const publishedCount = courseItems.filter((course) => course.published).length
  const draftCount = courseItems.length - publishedCount
  const needsAttention = courseItems.filter((course) => !course.hasContent || (course.published && course.classIds.length === 0)).length

  function openGeneration(course: TeacherCourse) {
    setGenerateCourse(course)
    setGenerateError('')
    setGenerateMeta(null)
    generateForm.setFieldsValue({
      topic: course.title,
      audience: course.audience || 'GENERAL',
      language: course.language || 'zh-CN',
      extraRequirements: '',
      chapterCountHint: course.chapterCount || undefined,
    })
  }

  async function startGeneration() {
    if (!generateCourse) return
    const values = await generateForm.validateFields()
    const course = generateCourse
    setGeneratingId(course.id)
    setGenerateError('')
    setGenerateMeta(null)
    try {
      const result = await generateTeacherCourse(course.id, {
        ...values,
        provider: aiProvider,
        outlineOnly: generateMode === 'outline',
      })
      setGenerateMeta(result.meta)
      await reloadCourses()
    } catch (error) {
      setGenerateError(error instanceof Error ? error.message : t('teacher.home.generateFailed'))
    } finally {
      setGeneratingId(null)
    }
  }

  return (
    <>
      <div className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>{t('teacher.home.pageEyebrow')}</p>
          <h1>{t('teacher.home.pageTitle')}</h1>
          <p>{t('teacher.home.pageSubtitle')}</p>
        </div>
        <button
          className={styles.primaryAction}
          type="button"
          onClick={() => {
            setCreateError('')
            form.resetFields()
            setCreateOpen(true)
          }}
        >
          {t('teacher.home.createCourse')}
        </button>
      </div>

      <section className={styles.summaryGrid} aria-label={t('teacher.home.summaryAria')}>
        <article><strong>{courseItems.length}</strong><span>{t('teacher.home.summaryAll')}</span></article>
        <article><strong>{draftCount}</strong><span>{t('teacher.home.draft')}</span></article>
        <article><strong>{publishedCount}</strong><span>{t('teacher.home.published')}</span></article>
        <article><strong>{needsAttention}</strong><span>{t('teacher.home.summaryNeedsAttention')}</span></article>
      </section>

      <div className={styles.modelBar}>
        <div><strong>{t('teacher.home.aiBarTitle')}</strong><span>{t('teacher.home.aiBarHint')}</span></div>
        {aiProviderSelector}
      </div>
      {aiProviderError ? <Alert type="error" showIcon message={aiProviderError} className={styles.alert} /> : null}

      {loadError ? <Alert type="error" showIcon message={loadError} className={styles.alert} /> : null}
      {generateError ? <Alert type="error" showIcon message={generateError} className={styles.alert} /> : null}

      <section className={styles.courseSection}>
        <div className={styles.sectionHeading}>
          <div><p className={styles.eyebrow}>{t('teacher.home.recentEyebrow')}</p><h2>{t('teacher.home.recentTitle')}</h2></div>
          <span>{t('teacher.home.courseCount', { count: courseItems.length })}</span>
        </div>
        {courses === null && !loadError ? (
          <div className={styles.skeletonGrid} aria-label={t('teacher.home.loadingAria')}><span /><span /><span /></div>
        ) : courseItems.length === 0 ? (
          <div className={styles.emptyState}><strong>{t('teacher.home.emptyTitle')}</strong><p>{t('teacher.home.emptyBody')}</p></div>
        ) : (
          <div className={styles.courseGrid}>
            {courseItems.map((course) => (
              <article className={styles.courseCard} key={course.id}>
                <div className={`${styles.cardArtwork} ${styles[`palette${course.id % 4}`] ?? ''}`}>
                  <span>{course.chapterCount || '—'}</span>
                  <small>{course.chapterCount ? t('teacher.home.chaptersTag') : t('teacher.home.draftTag')}</small>
                </div>
                <div className={styles.cardBody}>
                  <div className={styles.cardTags}>
                    <span className={course.published ? styles.published : undefined}>
                      {course.published ? t('teacher.home.published') : t('teacher.home.draft')}
                    </span>
                    <span>
                      {course.classIds.length
                        ? t('teacher.home.classCount', { count: course.classIds.length })
                        : t('teacher.home.noClass')}
                    </span>
                  </div>
                  <h3>{course.title}</h3>
                  <p>{course.description || t('teacher.home.noDescription')}</p>
                  <div className={styles.classNames}>
                    {course.classIds.length
                      ? course.classIds
                          .map((id) => classNameById.get(id) ?? t('teacher.home.classFallback', { id }))
                          .join('、')
                      : course.hasContent
                        ? t('teacher.home.contentReady')
                        : t('teacher.home.needsContent')}
                  </div>
                  <div className={styles.cardActions}>
                    <button type="button" onClick={() => navigate(`/teacher/courses/${course.id}`)}>
                      {t('teacher.home.enterWorkspace')}
                    </button>
                    <button type="button" disabled={generatingId === course.id || !aiOptionsAvailable} onClick={() => openGeneration(course)}>
                      {generatingId === course.id
                        ? t('teacher.home.generating')
                        : course.hasContent
                          ? t('teacher.home.regenerate')
                          : t('teacher.home.aiGenerate')}
                    </button>
                    <button type="button" onClick={() => openPublish(course)}>{t('teacher.home.publishSettings')}</button>
                    <button type="button" onClick={() => navigate(`/teacher/learning?courseId=${course.id}`)}>
                      {t('teacher.home.learningProgress')}
                    </button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      <Modal
        title={
          generateCourse
            ? t('teacher.home.generateModalPrefix', { title: generateCourse.title })
            : t('teacher.home.generateModalDefault')
        }
        open={generateCourse !== null}
        okText={
          generatingId
            ? t('teacher.home.generating')
            : generateMode === 'outline'
              ? t('teacher.home.generateOutline')
              : t('teacher.home.generateFull')
        }
        confirmLoading={generatingId !== null}
        okButtonProps={{ disabled: !aiOptionsAvailable }}
        onCancel={() => { setGenerateCourse(null); setGenerateMode('outline') }}
        onOk={() => { void startGeneration() }}
      >
        {generateError ? <Alert type="error" showIcon message={generateError} style={{ marginBottom: 16 }} /> : null}
        {generateMeta ? <GenerationMetaAlert meta={generateMeta} /> : null}
        <Typography.Paragraph type="secondary">{t('teacher.home.generateHint')}</Typography.Paragraph>
        <Radio.Group
          value={generateMode}
          onChange={(event) => setGenerateMode(event.target.value)}
          style={{ marginBottom: 16 }}
          optionType="button"
          options={[
            { label: t('teacher.home.modeOutline'), value: 'outline' },
            { label: t('teacher.home.modeFull'), value: 'full' },
          ]}
        />
        <Form form={generateForm} layout="vertical">
          <Form.Item
            name="topic"
            label={t('ai.generate.topic')}
            rules={[{ required: true, whitespace: true, message: t('ai.generate.topicRequired') }]}
          >
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item name="audience" label={t('ai.generate.audience')}>
            <Select
              options={[
                { value: 'GENERAL', label: t('ai.generate.audienceGeneral') },
                { value: 'CS', label: t('ai.generate.audienceCs') },
                { value: 'NON_CS', label: t('ai.generate.audienceNonCs') },
              ]}
            />
          </Form.Item>
          <Form.Item name="language" label={t('ai.generate.language')}>
            <Select
              options={[
                { value: 'zh-CN', label: t('ai.generate.langZh') },
                { value: 'en-US', label: t('ai.generate.langEn') },
              ]}
            />
          </Form.Item>
          <Form.Item name="chapterCountHint" label={t('ai.generate.chapterCountHint')}>
            <InputNumber min={1} max={20} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="extraRequirements" label={t('ai.generate.extraRequirements')}>
            <Input.TextArea rows={3} maxLength={2000} showCount />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('teacher.home.newCourseTitle')}
        open={createOpen}
        okText={t('teacher.home.createOk')}
        confirmLoading={creating}
        onCancel={() => setCreateOpen(false)}
        onOk={() => {
          void submitCreate()
        }}
      >
        {createError ? <Alert type="error" showIcon message={createError} style={{ marginBottom: 16 }} /> : null}
        <Form form={form} layout="vertical" requiredMark={false}>
          <Form.Item
            name="title"
            label={t('teacher.home.courseNameLabel')}
            rules={[
              { required: true, whitespace: true, message: t('teacher.home.courseNameRequired') },
              { max: 200, message: t('teacher.home.courseNameTooLong') },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="description"
            label={t('teacher.home.descriptionLabel')}
            rules={[{ max: 2000, message: t('teacher.home.descriptionTooLong') }]}
          >
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={publishCourse ? t('teacher.home.publishTitle', { title: publishCourse.title }) : t('teacher.home.publishDefault')}
        open={publishCourse !== null}
        okText={t('teacher.home.saveOk')}
        confirmLoading={publishing}
        okButtonProps={{
          disabled: classes.length === 0 || (!publishCourse?.published && selectedClassIds.length === 0),
        }}
        onCancel={() => setPublishCourse(null)}
        onOk={() => {
          void submitPublish()
        }}
      >
        {publishError ? <Alert type="error" showIcon message={publishError} style={{ marginBottom: 16 }} /> : null}
        {classes.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message={t('teacher.home.notInClass')}
          />
        ) : (
          <>
            <Typography.Paragraph type="secondary">
              {publishCourse?.published
                ? t('teacher.home.publishHintPublished')
                : t('teacher.home.publishHintFirst')}
            </Typography.Paragraph>
            <Checkbox.Group
              style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
              options={classes.map((item) => ({ label: item.name, value: item.id }))}
              value={selectedClassIds}
              onChange={(values) => setSelectedClassIds(values as number[])}
            />
          </>
        )}
      </Modal>
    </>
  )
}
